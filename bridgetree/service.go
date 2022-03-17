package bridgetree

import (
	"context"
	"encoding/hex"

	"github.com/hermeznetwork/hermez-bridge/bridgetree/pb"
)

const limit = 25

type BridgeService struct {
	storage    bridgeServiceStorage
	bridgeCtrl *BridgeTree
}

func (s *BridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	deposits, err := s.storage.GetDeposits(ctx, uint(req.Offset), uint(0), limit)
	if err != nil {
		return nil, err
	}

	var pbDeposits []*pb.Deposit
	for _, deposit := range deposits {
		pbDeposits = append(pbDeposits, &pb.Deposit{
			OrigNet:    uint32(deposit.OriginalNetwork),
			TokenAddr:  deposit.TokenAddress.Hex(),
			Amount:     deposit.Amount.String(),
			DestNet:    uint32(deposit.DestinationNetwork),
			DestAddr:   deposit.DestinationAddress.Hex(),
			BlockNum:   deposit.BlockNumber,
			DepositCnt: uint64(deposit.DepositCount),
		})
	}

	return &pb.GetBridgesResponse{
		Deposits: pbDeposits,
	}, nil
}

func (s *BridgeService) GetClaims(ctx context.Context, req *pb.GetClaimsRequest) (*pb.GetClaimsResponse, error) {
	claims, err := s.storage.GetClaims(ctx, uint(0), limit, uint(req.Offset))
	if err != nil {
		return nil, err
	}

	var pbClaims []*pb.Claim
	for _, claim := range claims {
		pbClaims = append(pbClaims, &pb.Claim{
			Index:     uint64(claim.Index),
			OrigNet:   uint32(claim.OriginalNetwork),
			TokenAddr: claim.Token.Hex(),
			Amount:    claim.Amount.String(),
			DestNet:   uint32(claim.DestinationNetwork),
			DestAddr:  claim.DestinationAddress.Hex(),
			BlockNum:  claim.BlockNumber,
		})
	}

	return &pb.GetClaimsResponse{
		Claims: pbClaims,
	}, nil
}

func (s *BridgeService) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	merkleProof, exitRoot, err := s.bridgeCtrl.GetClaim(uint(req.OrigNet), uint(req.DepositCnt))
	if err != nil {
		return nil, err
	}

	var proof []string
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, hex.EncodeToString(merkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			MerkleProof:    proof,
			ExitRootNum:    exitRoot.GlobalExitRootNum.Uint64(),
			MainExitRoot:   exitRoot.ExitRoots[0].Hex(),
			RollupExitRoot: exitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

func (s *BridgeService) GetClaimStatus(ctx context.Context, req *pb.GetClaimStatusRequest) (*pb.GetClaimStatusResponse, error) {
	exitRoot, err := s.bridgeCtrl.storage.GetLatestExitRoot(ctx)
	if err != nil {
		return nil, err
	}

	tID := s.bridgeCtrl.networkIDs[uint64(req.OrigNet)]
	ctx = context.WithValue(ctx, contextKeyNetwork, tID) //nolint

	depositCnt, err := s.bridgeCtrl.exitRootTrees[tID].getCntByRoot(ctx, exitRoot.ExitRoots[tID])
	if err != nil {
		return nil, err
	}

	var ready bool
	if depositCnt <= uint(req.DepositCnt) {
		ready = true
	} else {
		ready = false
	}

	return &pb.GetClaimStatusResponse{
		Ready: ready,
	}, nil
}
