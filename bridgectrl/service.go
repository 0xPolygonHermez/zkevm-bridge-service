package bridgectrl

import (
	"context"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/utils/gerror"
)

const (
	limit   = 25
	version = "v1"
)

type bridgeService struct {
	storage    BridgeServiceStorage
	bridgeCtrl *BridgeController
	pb.UnimplementedBridgeServiceServer
}

// NewBridgeService creates new bridge service.
func NewBridgeService(storage BridgeServiceStorage, bridgeCtrl *BridgeController) pb.BridgeServiceServer {
	return &bridgeService{
		storage:    storage,
		bridgeCtrl: bridgeCtrl,
	}
}

// CheckAPI returns api version.
func (s *bridgeService) CheckAPI(ctx context.Context, req *pb.CheckAPIRequest) (*pb.CheckAPIResponse, error) {
	return &pb.CheckAPIResponse{
		Api: version,
	}, nil
}

// GetBridges returns bridges for the specific smart contract address both in L1 and L2.
func (s *bridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	deposits, err := s.storage.GetDeposits(ctx, req.DestAddr, limit, uint(req.Offset))
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
			NetworkId:  uint32(deposit.NetworkID),
			TxHash:     deposit.TxHash.String(),
		})
	}

	return &pb.GetBridgesResponse{
		Deposits: pbDeposits,
	}, nil
}

// GetClaims returns claims for the specific smart contract address both in L1 and L2.
func (s *bridgeService) GetClaims(ctx context.Context, req *pb.GetClaimsRequest) (*pb.GetClaimsResponse, error) {
	claims, err := s.storage.GetClaims(ctx, req.DestAddr, limit, uint(req.Offset)) //nolint:gomnd
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
			NetworkId: uint32(claim.NetworkID),
			DestAddr:  claim.DestinationAddress.Hex(),
			BlockNum:  claim.BlockNumber,
			TxHash:    claim.TxHash.String(),
		})
	}

	return &pb.GetClaimsResponse{
		Claims: pbClaims,
	}, nil
}

// GetProof returns the merkle proof for the specific deposit.
func (s *bridgeService) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	merkleProof, exitRoot, err := s.bridgeCtrl.GetClaim(uint(req.NetId), uint(req.DepositCnt))
	if err != nil {
		return nil, err
	}

	var proof []string
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			MerkleProof:    proof,
			ExitRootNum:    exitRoot.GlobalExitRootNum.Uint64(),
			L2ExitRootNum:  exitRoot.GlobalExitRootL2Num.Uint64(),
			MainExitRoot:   exitRoot.ExitRoots[0].Hex(),
			RollupExitRoot: exitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetClaimStatus returns the claim status whether it is able to send a claim transaction or not.
func (s *bridgeService) GetClaimStatus(ctx context.Context, req *pb.GetClaimStatusRequest) (*pb.GetClaimStatusResponse, error) {
	var (
		exitRoot *etherman.GlobalExitRoot
		err      error
	)
	if req.NetId == uint32(MainNetworkID) {
		exitRoot, err = s.bridgeCtrl.storage.GetLatestL1SyncedExitRoot(ctx)
	} else {
		exitRoot, err = s.bridgeCtrl.storage.GetLatestL2SyncedExitRoot(ctx)
	}

	if err == gerror.ErrStorageNotFound {
		return &pb.GetClaimStatusResponse{
			Ready: false,
		}, nil
	} else if err != nil {
		return nil, err
	}

	tID := s.bridgeCtrl.networkIDs[uint(req.NetId)]
	ctx = context.WithValue(ctx, contextKeyNetwork, tID) //nolint
	tID--
	depositCnt, err := s.bridgeCtrl.exitTrees[tID].getDepositCntByRoot(ctx, exitRoot.ExitRoots[tID])
	if err != nil {
		return nil, err
	}

	var ready bool
	if depositCnt >= uint(req.DepositCnt) {
		ready = true
	} else {
		ready = false
	}

	return &pb.GetClaimStatusResponse{
		Ready: ready,
	}, nil
}

// GetTokenWrapped returns the token wrapped created for a specific network
func (s *bridgeService) GetTokenWrapped(ctx context.Context, req *pb.GetTokenWrappedRequest) (*pb.GetTokenWrappedResponse, error) {
	tokenWrapped, err := s.bridgeCtrl.GetTokenWrapped(uint(req.OrigNet), common.HexToAddress(req.OrigTokenAddr))
	if err != nil {
		return nil, err
	}
	return &pb.GetTokenWrappedResponse{
		Tokenwrapped: &pb.TokenWrapped{
			OrigNet:           uint32(tokenWrapped.OriginalNetwork),
			OriginalTokenAddr: tokenWrapped.OriginalTokenAddress.Hex(),
			WrappedTokenAddr:  tokenWrapped.WrappedTokenAddress.Hex(),
			NetworkId:         uint32(tokenWrapped.NetworkID),
		},
	}, nil
}
