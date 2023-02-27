package server

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
)

type bridgeService struct {
	storage          bridgeServiceStorage
	networkIDs       map[uint]uint8
	height           uint8
	defaultPageLimit uint32
	maxPageLimit     uint32
	version          string
	cache            *lru.Cache[string, [][]byte]
	pb.UnimplementedBridgeServiceServer
}

// NewBridgeService creates new bridge service.
func NewBridgeService(cfg Config, height uint8, networks []uint, storage bridgeServiceStorage) pb.BridgeServiceServer {
	var networkIDs = make(map[uint]uint8)
	for i, network := range networks {
		networkIDs[network] = uint8(i)
	}
	cache, err := lru.New[string, [][]byte](int(cfg.CacheSize))
	if err != nil {
		panic(err)
	}
	return &bridgeService{
		storage:          storage,
		height:           height,
		networkIDs:       networkIDs,
		defaultPageLimit: cfg.DefaultPageLimit,
		maxPageLimit:     cfg.MaxPageLimit,
		version:          cfg.BridgeVersion,
		cache:            cache,
	}
}

func (s *bridgeService) getNetworkID(networkID uint) (uint8, error) {
	tID, found := s.networkIDs[networkID]
	if !found {
		return 0, gerror.ErrNetworkNotRegister
	}
	return tID, nil
}

// getNode returns the children hash pairs for a given parent hash.
func (s *bridgeService) getNode(ctx context.Context, parentHash [bridgectrl.KeyLen]byte) (left, right [bridgectrl.KeyLen]byte, err error) {
	value, ok := s.cache.Get(string(parentHash[:]))
	if !ok {
		var err error
		value, err = s.storage.Get(ctx, parentHash[:], nil)
		if err != nil {
			return left, right, fmt.Errorf("parentHash: %v, error: %w", parentHash, err)
		}
		s.cache.Add(string(parentHash[:]), value)
	}
	copy(left[:], value[0])
	copy(right[:], value[1])
	return left, right, nil
}

// getProof returns the merkle proof for a given index and root.
func (s *bridgeService) getProof(index uint, root [bridgectrl.KeyLen]byte) ([][bridgectrl.KeyLen]byte, error) {
	var siblings [][bridgectrl.KeyLen]byte

	cur := root
	ctx := context.Background()
	// It starts in height-1 because 0 is the level of the leafs
	for h := int(s.height - 1); h >= 0; h-- {
		left, right, err := s.getNode(ctx, cur)
		if err != nil {
			return nil, fmt.Errorf("height: %d, cur: %v, error: %w", h, cur, err)
		}
		/*
					*        Root                (level h=3 => height=4)
					*      /     \
					*	 O5       O6             (level h=2)
					*	/ \      / \
					*  O1  O2   O3  O4           (level h=1)
			        *  /\   /\   /\ /\
					* 0  1 2  3 4 5 6 7 Leafs    (level h=0)
					* Example 1:
					* Choose index = 3 => 011 binary
					* Assuming we are in level 1 => h=1; 1<<h = 010 binary
					* Now, let's do AND operation => 011&010=010 which is higher than 0 so we need the left sibling (O1)
					* Example 2:
					* Choose index = 4 => 100 binary
					* Assuming we are in level 1 => h=1; 1<<h = 010 binary
					* Now, let's do AND operation => 100&010=000 which is not higher than 0 so we need the right sibling (O4)
					* Example 3:
					* Choose index = 4 => 100 binary
					* Assuming we are in level 2 => h=2; 1<<h = 100 binary
					* Now, let's do AND operation => 100&100=100 which is higher than 0 so we need the left sibling (O5)
		*/

		if index&(1<<h) > 0 {
			siblings = append(siblings, left)
			cur = right
		} else {
			siblings = append(siblings, right)
			cur = left
		}
	}

	// We need to invert the siblings to go from leafs to the top
	for st, en := 0, len(siblings)-1; st < en; st, en = st+1, en-1 {
		siblings[st], siblings[en] = siblings[en], siblings[st]
	}

	return siblings, nil
}

// getClaimReadiness returns true if the given deposit is ready for claim.
func (s *bridgeService) getClaimReadiness(ctx context.Context, depositCount uint, tID uint8) (*etherman.GlobalExitRoot, bool, error) {
	exitRoot, err := s.storage.GetLatestExitRoot(ctx, tID != 0, nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return nil, false, err
		}
		return nil, false, nil
	}
	depositCnt, err := s.storage.GetDepositCountByRoot(ctx, exitRoot.ExitRoots[tID][:], uint8(tID), nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return nil, false, err
		}
		depositCnt = 0
	}
	return exitRoot, depositCnt >= depositCount, nil
}

// getDepositStatus returns deposit with ready_for_claim status.
func (s *bridgeService) getDepositStatus(ctx context.Context, depositCount uint, networkID uint, destNetworkID uint) (string, bool, error) {
	var (
		claimTxHash string
	)
	// Get the claim tx hash
	claim, err := s.storage.GetClaim(ctx, depositCount, destNetworkID, nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return "", false, err
		}
	} else {
		claimTxHash = claim.TxHash.String()
	}
	tID, err := s.getNetworkID(networkID)
	if err != nil {
		return "", false, gerror.ErrNetworkNotRegister
	}
	// Get the claim readiness
	_, readiness, err := s.getClaimReadiness(ctx, depositCount, tID)
	return claimTxHash, readiness, err
}

// CheckAPI returns api version.
func (s *bridgeService) CheckAPI(ctx context.Context, req *pb.CheckAPIRequest) (*pb.CheckAPIResponse, error) {
	return &pb.CheckAPIResponse{
		Api: s.version,
	}, nil
}

// GetBridges returns bridges for the destination address both in L1 and L2.
func (s *bridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}
	totalCount, err := s.storage.GetDepositCount(ctx, req.DestAddr, nil)
	if err != nil {
		return nil, err
	}
	deposits, err := s.storage.GetDeposits(ctx, req.DestAddr, uint(limit), uint(req.Offset), nil)
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
				LeafType:      uint32(deposit.LeafType),
				OrigNet:       uint32(deposit.OriginalNetwork),
				OrigAddr:      deposit.OriginalAddress.Hex(),
				Amount:        deposit.Amount.String(),
				DestNet:       uint32(deposit.DestinationNetwork),
				DestAddr:      deposit.DestinationAddress.Hex(),
				BlockNum:      deposit.BlockNumber,
				DepositCnt:    uint64(deposit.DepositCount),
				NetworkId:     uint32(deposit.NetworkID),
				TxHash:        deposit.TxHash.String(),
				ClaimTxHash:   claimTxHash,
				Metadata:      "0x" + hex.EncodeToString(deposit.Metadata),
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
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}
	totalCount, err := s.storage.GetClaimCount(ctx, req.DestAddr, nil)
	if err != nil {
		return nil, err
	}
	claims, err := s.storage.GetClaims(ctx, req.DestAddr, uint(limit), uint(req.Offset), nil) //nolint:gomnd
	if err != nil {
		return nil, err
	}

	var pbClaims []*pb.Claim
	for _, claim := range claims {
		pbClaims = append(pbClaims, &pb.Claim{
			Index:     uint64(claim.Index),
			OrigNet:   uint32(claim.OriginalNetwork),
			OrigAddr:  claim.OriginalAddress.Hex(),
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
	networkID := uint(req.NetId)
	tID, err := s.getNetworkID(networkID)
	if err != nil {
		return nil, err
	}

	globalExitRoot, readiness, err := s.getClaimReadiness(ctx, uint(req.DepositCnt), tID)
	if err != nil {
		return nil, err
	}
	if !readiness {
		return nil, gerror.ErrDepositNotSynced
	}

	merkleProof, err := s.getProof(uint(req.DepositCnt), globalExitRoot.ExitRoots[tID])
	if err != nil {
		return nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
	}

	var proof []string
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			MerkleProof:    proof,
			MainExitRoot:   globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot: globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetBridge returns the bridge  with status whether it is able to send a claim transaction or not.
func (s *bridgeService) GetBridge(ctx context.Context, req *pb.GetBridgeRequest) (*pb.GetBridgeResponse, error) {
	deposit, err := s.storage.GetDeposit(ctx, uint(req.DepositCnt), uint(req.NetId), nil)
	if err != nil {
		return nil, err
	}

	claimTxHash, readyForClaim, err := s.getDepositStatus(ctx, uint(req.DepositCnt), uint(req.NetId), deposit.DestinationNetwork)
	if err != nil {
		return nil, err
	}

	return &pb.GetBridgeResponse{
		Deposit: &pb.Deposit{
			LeafType:      uint32(deposit.LeafType),
			OrigNet:       uint32(deposit.OriginalNetwork),
			OrigAddr:      deposit.OriginalAddress.Hex(),
			Amount:        deposit.Amount.String(),
			DestNet:       uint32(deposit.DestinationNetwork),
			DestAddr:      deposit.DestinationAddress.Hex(),
			BlockNum:      deposit.BlockNumber,
			DepositCnt:    uint64(deposit.DepositCount),
			NetworkId:     uint32(deposit.NetworkID),
			TxHash:        deposit.TxHash.String(),
			ClaimTxHash:   claimTxHash,
			Metadata:      "0x" + hex.EncodeToString(deposit.Metadata),
			ReadyForClaim: readyForClaim,
		},
	}, nil
}

// GetTokenWrapped returns the token wrapped created for a specific network
func (s *bridgeService) GetTokenWrapped(ctx context.Context, req *pb.GetTokenWrappedRequest) (*pb.GetTokenWrappedResponse, error) {
	tokenWrapped, err := s.storage.GetTokenWrapped(ctx, uint(req.OrigNet), common.HexToAddress(req.OrigTokenAddr), nil)
	if err != nil {
		return nil, err
	}
	return &pb.GetTokenWrappedResponse{
		Tokenwrapped: &pb.TokenWrapped{
			OrigNet:           uint32(tokenWrapped.OriginalNetwork),
			OriginalTokenAddr: tokenWrapped.OriginalTokenAddress.Hex(),
			WrappedTokenAddr:  tokenWrapped.WrappedTokenAddress.Hex(),
			NetworkId:         uint32(tokenWrapped.NetworkID),
			Name:              tokenWrapped.Name,
			Symbol:            tokenWrapped.Symbol,
			Decimals:          uint32(tokenWrapped.Decimals),
		},
	}, nil
}
