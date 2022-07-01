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
	defaultPageLimit = 25
	maxPageLimit     = 100
	version          = "v1"
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

// GetBridges returns bridges for the destination address both in L1 and L2.
func (s *bridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	totalCount, err := s.storage.GetDepositCount(ctx, req.DestAddr)
	if err != nil {
		return nil, err
	}
	deposits, err := s.storage.GetDeposits(ctx, req.DestAddr, uint(limit), uint(req.Offset))
	if err != nil {
		return nil, err
	}

	var pbDeposits []*pb.Deposit
	for _, deposit := range deposits {
		claimTxHash, readyForClaim, err := s.getDepositStatus(ctx, deposit.DepositCount, deposit.NetworkID, deposit.DestinationNetwork)
		if err != nil {
			return nil, err
		}
		pbDeposits = append(
			pbDeposits, &pb.Deposit{
				OrigNet:       uint32(deposit.OriginalNetwork),
				TokenAddr:     deposit.TokenAddress.Hex(),
				Amount:        deposit.Amount.String(),
				DestNet:       uint32(deposit.DestinationNetwork),
				DestAddr:      deposit.DestinationAddress.Hex(),
				BlockNum:      deposit.BlockNumber,
				DepositCnt:    uint64(deposit.DepositCount),
				NetworkId:     uint32(deposit.NetworkID),
				TxHash:        deposit.TxHash.String(),
				ClaimTxHash:   claimTxHash,
				ReadyForClaim: readyForClaim,
			},
		)
	}

	return &pb.GetBridgesResponse{
		Deposits: pbDeposits,
		TotalCnt: totalCount,
	}, nil
}

// GetClaims returns claims for the specific smart contract address both in L1 and L2.
func (s *bridgeService) GetClaims(ctx context.Context, req *pb.GetClaimsRequest) (*pb.GetClaimsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	totalCount, err := s.storage.GetClaimCount(ctx, req.DestAddr)
	if err != nil {
		return nil, err
	}
	claims, err := s.storage.GetClaims(ctx, req.DestAddr, uint(limit), uint(req.Offset)) //nolint:gomnd
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
		Claims:   pbClaims,
		TotalCnt: totalCount,
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

// GetDepositStatus returns the claim status whether it is able to send a claim transaction or not.
func (s *bridgeService) GetBridge(ctx context.Context, req *pb.GetBridgeRequest) (*pb.GetBridgeResponse, error) {
	deposit, err := s.storage.GetDeposit(ctx, uint(req.DepositCnt), uint(req.NetId))
	if err != nil {
		return nil, err
	}

	claimTxHash, readyForClaim, err := s.getDepositStatus(ctx, uint(req.DepositCnt), uint(req.NetId), deposit.DestinationNetwork)
	if err != nil {
		return nil, err
	}

	return &pb.GetBridgeResponse{
		Deposit: &pb.Deposit{
			OrigNet:       uint32(deposit.OriginalNetwork),
			TokenAddr:     deposit.TokenAddress.Hex(),
			Amount:        deposit.Amount.String(),
			DestNet:       uint32(deposit.DestinationNetwork),
			DestAddr:      deposit.DestinationAddress.Hex(),
			BlockNum:      deposit.BlockNumber,
			DepositCnt:    uint64(deposit.DepositCount),
			NetworkId:     uint32(deposit.NetworkID),
			TxHash:        deposit.TxHash.String(),
			ClaimTxHash:   claimTxHash,
			ReadyForClaim: readyForClaim,
		},
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

func (s *bridgeService) getDepositStatus(ctx context.Context, depositCount uint, networkID uint, destNetworkID uint) (string, bool, error) {
	var (
		claimTxHash string
		exitRoot    *etherman.GlobalExitRoot
	)
	// Get the claim tx hash
	claim, err := s.storage.GetClaim(ctx, depositCount, destNetworkID)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return "", false, err
		}
	} else {
		claimTxHash = claim.TxHash.String()
	}
	// Get the claim readiness
	if networkID == MainNetworkID {
		exitRoot, err = s.bridgeCtrl.storage.GetLatestL1SyncedExitRoot(ctx)
	} else {
		exitRoot, err = s.bridgeCtrl.storage.GetLatestL2SyncedExitRoot(ctx)
	}

	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return "", false, err
		}
		return claimTxHash, false, nil
	}

	tID, found := s.bridgeCtrl.networkIDs[networkID]
	if !found {
		return "", false, gerror.ErrNetworkNotRegister
	}
	tID--
	depositCnt, err := s.bridgeCtrl.storage.GetDepositCountByRoot(ctx, exitRoot.ExitRoots[tID][:], uint8(tID+1))
	if err != nil {
		return "", false, err
	}

	return claimTxHash, depositCnt > depositCount, nil
}
