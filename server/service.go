package server

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/0xPolygon/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygon/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygon/zkevm-bridge-service/etherman"
	"github.com/0xPolygon/zkevm-bridge-service/log"
	"github.com/0xPolygon/zkevm-bridge-service/server/metrics"
	"github.com/0xPolygon/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
)

type bridgeService struct {
	storage          bridgeServiceStorage
	networkIDs       map[uint32]uint8
	height           uint8
	defaultPageLimit uint32
	maxPageLimit     uint32
	version          string
	finalizedGER     bool
	cache            *lru.Cache[string, [][]byte]
	pb.UnimplementedBridgeServiceServer
}

// NewBridgeService creates new bridge service.
func NewBridgeService(cfg Config, height uint8, networks []uint32, storage interface{}) *bridgeService {
	var networkIDs = make(map[uint32]uint8)
	for i, network := range networks {
		networkIDs[network] = uint8(i) // nolint:gosec
	}
	cache, err := lru.New[string, [][]byte](cfg.CacheSize)
	if err != nil {
		panic(err)
	}
	if cfg.FinalizedGEREnabled {
		log.Info("Finalized flag activated to compute proofs")
	}
	return &bridgeService{
		storage:          storage.(bridgeServiceStorage),
		height:           height,
		networkIDs:       networkIDs,
		defaultPageLimit: cfg.DefaultPageLimit,
		maxPageLimit:     cfg.MaxPageLimit,
		version:          cfg.BridgeVersion,
		finalizedGER:     cfg.FinalizedGEREnabled,    
		cache:            cache,
	}
}

// getNode returns the children hash pairs for a given parent hash.
func (s *bridgeService) getNode(ctx context.Context, parentHash [bridgectrl.KeyLen]byte, dbTx interface{}) (left, right [bridgectrl.KeyLen]byte, err error) {
	value, ok := s.cache.Get(string(parentHash[:]))
	if !ok {
		var err error
		value, err = s.storage.Get(ctx, parentHash[:], dbTx)
		if err != nil {
			return left, right, fmt.Errorf("parentHash: %s, error: %v", common.BytesToHash(parentHash[:]).String(), err)
		}
		s.cache.Add(string(parentHash[:]), value)
	}
	copy(left[:], value[0])
	copy(right[:], value[1])
	return left, right, nil
}

// getProof returns the merkle proof for a given index and root.
func (s *bridgeService) getProof(ctx context.Context, index uint32, root [bridgectrl.KeyLen]byte, dbTx interface{}) ([][bridgectrl.KeyLen]byte, error) {
	var siblings [][bridgectrl.KeyLen]byte

	cur := root
	// It starts in height-1 because 0 is the level of the leafs
	for h := int(s.height - 1); h >= 0; h-- {
		left, right, err := s.getNode(ctx, cur, dbTx)
		if err != nil {
			return nil, fmt.Errorf("height: %d, cur: %s, error: %v", h, common.BytesToHash(cur[:]).String(), err)
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

// getRollupExitProof returns the merkle proof for the zkevm leaf.
func (s *bridgeService) getRollupExitProof(ctx context.Context, rollupIndex uint32, root common.Hash, dbTx interface{}) ([][bridgectrl.KeyLen]byte, common.Hash, error) {
	// Get leaves given the root
	leaves, err := s.storage.GetRollupExitLeavesByRoot(ctx, root, dbTx)
	if err != nil {
		err = fmt.Errorf("error getting leaves by ger: %s, error: %w", root.String(), err)
		return nil, common.Hash{}, err
	}
	// Compute Siblings
	var ls [][bridgectrl.KeyLen]byte
	for _, l := range leaves {
		var aux [bridgectrl.KeyLen]byte
		copy(aux[:], l.Leaf.Bytes())
		ls = append(ls, aux)
	}
	siblings, r, err := bridgectrl.ComputeSiblings(rollupIndex, ls, s.height)
	if err != nil {
		return nil, common.Hash{}, err
	} else if root != r {
		log.Warnf("error checking calculated root: required: %s, calculated:%s", root.String(), r.String())
		return nil, common.Hash{}, fmt.Errorf("error checking calculated root: required:%s, calculated: %s", root.String(), r.String())
	}
	if len(siblings) == 0 || len(ls) == 0 {
		return nil, common.Hash{}, fmt.Errorf("no siblings found for root: %s", root.String())
	}
	if len(ls) <= int(rollupIndex) {
		return siblings, common.Hash{}, fmt.Errorf("error getting rollupLeaf. Not synced yet")
	}
	return siblings, ls[rollupIndex], nil
}

// GetClaimProof returns the merkle proof to claim the given deposit.
func (s *bridgeService) GetClaimProof(ctx context.Context, depositCnt, networkID uint32, dbTx interface{}) (*etherman.GlobalExitRoot, [][bridgectrl.KeyLen]byte, [][bridgectrl.KeyLen]byte, error) {
	deposit, err := s.storage.GetDeposit(ctx, depositCnt, networkID, dbTx)
	if err != nil {
		return nil, nil, nil, err
	}

	if !deposit.ReadyForClaim {
		return nil, nil, nil, gerror.ErrDepositNotSynced
	}

	var globalExitRoot *etherman.GlobalExitRoot
	if s.finalizedGER && deposit.DestinationNetwork != 0 && networkID != 0 { // This finalizedGER flag must be disabled if all networks are synced in the same bridge service.
		globalExitRoot, err = s.storage.GetLatestTrustedExitRoot(ctx, networkID, dbTx)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		globalExitRoot, err = s.storage.GetLatestExitRoot(ctx, networkID, deposit.DestinationNetwork, dbTx)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var (
		merkleProof       [][bridgectrl.KeyLen]byte
		rollupMerkleProof [][bridgectrl.KeyLen]byte
		rollupLeaf        common.Hash
	)
	if networkID == 0 { // Mainnet
		merkleProof, err = s.getProof(ctx, depositCnt, globalExitRoot.ExitRoots[0], dbTx)
		if err != nil {
			log.Error("error getting merkleProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
		}
		rollupMerkleProof = emptyProof()
	} else { // Rollup
		rollupMerkleProof, rollupLeaf, err = s.getRollupExitProof(ctx, networkID-1, globalExitRoot.ExitRoots[1], dbTx)
		if err != nil {
			log.Error("error getting rollupProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the rollup proof failed, error: %v, network: %d", err, networkID)
		}
		merkleProof, err = s.getProof(ctx, depositCnt, rollupLeaf, dbTx)
		if err != nil {
			log.Error("error getting merkleProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
		}
	}

	return globalExitRoot, merkleProof, rollupMerkleProof, nil
}

// checkDepositIncludedInRoot verifies that the deposit leaf was already part of the
// exit tree when the given root was computed. Without this check, getProof can walk
// one index past the last leaf of the tree (the frontier nodes persisted by addLeaf
// have zero-hash right children) and return a valid proof of the EMPTY leaf instead
// of failing, presenting it as a legitimate claim proof for a non-included deposit.
func (s *bridgeService) checkDepositIncludedInRoot(ctx context.Context, depositCnt, networkID uint32, root common.Hash, dbTx interface{}) error {
	lastDepositCnt, err := s.storage.GetDepositCountByRoot(ctx, root[:], networkID, dbTx)
	if err != nil {
		return fmt.Errorf("error getting deposit count for exit root: %s, network: %d. Err: %w", root.String(), networkID, err)
	}
	if depositCnt > lastDepositCnt {
		return fmt.Errorf("deposit %d for network %d is not included in the exit root %s (last included deposit: %d)",
			depositCnt, networkID, root.String(), lastDepositCnt)
	}
	return nil
}

// GetClaimProofbyGER returns the merkle proof to claim the given deposit.
func (s *bridgeService) GetClaimProofbyGER(ctx context.Context, depositCnt, networkID uint32, GER common.Hash, dbTx interface{}) (*etherman.GlobalExitRoot, [][bridgectrl.KeyLen]byte, [][bridgectrl.KeyLen]byte, error) {
	if dbTx == nil { // if the call comes from the rest API
		deposit, err := s.storage.GetDeposit(ctx, depositCnt, networkID, nil)
		if err != nil {
			err = fmt.Errorf("error getting deposit %d for network: %d. Err: %w", depositCnt, networkID, err)
			return nil, nil, nil, err
		}

		if !deposit.ReadyForClaim {
			log.Warnf("Deposit not ready for claim. Deposit: %d, Network: %d", depositCnt, networkID)
			//return nil, nil, nil, gerror.ErrDepositNotSynced
		}
	}

	globalExitRoot, err := s.storage.GetL1ExitRootByGER(ctx, GER, dbTx)
	if err != nil {
		err = fmt.Errorf("error getting GlobalExitRoot data for GER: %s. Err: %w", GER.String(), err)
		return nil, nil, nil, err
	}

	var (
		merkleProof       [][bridgectrl.KeyLen]byte
		rollupMerkleProof [][bridgectrl.KeyLen]byte
		rollupLeaf        common.Hash
	)
	if networkID == 0 { // Mainnet
		if err := s.checkDepositIncludedInRoot(ctx, depositCnt, networkID, globalExitRoot.ExitRoots[0], dbTx); err != nil {
			log.Errorf("deposit not included in root. Error: %v", err)
			return nil, nil, nil, err
		}
		merkleProof, err = s.getProof(ctx, depositCnt, globalExitRoot.ExitRoots[0], dbTx)
		if err != nil {
			log.Errorf("error getting merkleProof. Error: %w", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed (MAINNET), error: %v, network: %d", err, networkID)
		}
		rollupMerkleProof = emptyProof()
	} else { // Rollup
		rollupMerkleProof, rollupLeaf, err = s.getRollupExitProof(ctx, networkID-1, globalExitRoot.ExitRoots[1], dbTx)
		if err != nil {
			log.Errorf("error getting rollupProof. Error: %w", err)
			return nil, nil, nil, fmt.Errorf("getting the rollupexit proof failed, error: %v, network: %d", err, networkID)
		}
		if err := s.checkDepositIncludedInRoot(ctx, depositCnt, networkID, rollupLeaf, dbTx); err != nil {
			log.Errorf("deposit not included in root. Error: %v", err)
			return nil, nil, nil, err
		}
		merkleProof, err = s.getProof(ctx, depositCnt, rollupLeaf, dbTx)
		if err != nil {
			log.Errorf("error getting merkleProof. Error: %w", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed (ROLLUP), error: %v, network: %d", err, networkID)
		}
	}

	return globalExitRoot, merkleProof, rollupMerkleProof, nil
}

// GetClaimProofForCompressed returns the merkle proof to claim the given deposit.
func (s *bridgeService) GetClaimProofForCompressed(ctx context.Context, ger common.Hash, depositCnt, networkID uint32, dbTx interface{}) (*etherman.GlobalExitRoot, [][bridgectrl.KeyLen]byte, [][bridgectrl.KeyLen]byte, error) {
	if dbTx == nil { // if the call comes from the rest API
		deposit, err := s.storage.GetDeposit(ctx, depositCnt, networkID, nil)
		if err != nil {
			return nil, nil, nil, err
		}

		if !deposit.ReadyForClaim {
			return nil, nil, nil, gerror.ErrDepositNotSynced
		}
	}

	globalExitRoot, err := s.storage.GetL1ExitRootByGER(ctx, ger, dbTx)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		merkleProof       [][bridgectrl.KeyLen]byte
		rollupMerkleProof [][bridgectrl.KeyLen]byte
		rollupLeaf        common.Hash
	)
	if networkID == 0 { // Mainnet
		if err := s.checkDepositIncludedInRoot(ctx, depositCnt, networkID, globalExitRoot.ExitRoots[0], dbTx); err != nil {
			log.Errorf("deposit not included in root. Error: %v", err)
			return nil, nil, nil, err
		}
		merkleProof, err = s.getProof(ctx, depositCnt, globalExitRoot.ExitRoots[0], dbTx)
		if err != nil {
			log.Error("error getting merkleProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
		}
		rollupMerkleProof = emptyProof()
	} else { // Rollup
		rollupMerkleProof, rollupLeaf, err = s.getRollupExitProof(ctx, networkID-1, globalExitRoot.ExitRoots[1], dbTx)
		if err != nil {
			log.Error("error getting rollupProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the rollup proof failed, error: %v, network: %d", err, networkID)
		}
		if err := s.checkDepositIncludedInRoot(ctx, depositCnt, networkID, rollupLeaf, dbTx); err != nil {
			log.Errorf("deposit not included in root. Error: %v", err)
			return nil, nil, nil, err
		}
		merkleProof, err = s.getProof(ctx, depositCnt, rollupLeaf, dbTx)
		if err != nil {
			log.Error("error getting merkleProof. Error: ", err)
			return nil, nil, nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
		}
	}

	return globalExitRoot, merkleProof, rollupMerkleProof, nil
}

func emptyProof() [][bridgectrl.KeyLen]byte {
	var proof [][bridgectrl.KeyLen]byte
	for i := 0; i < 32; i++ {
		proof = append(proof, common.Hash{})
	}
	return proof
}

// GetDepositStatus returns deposit with ready_for_claim status.
func (s *bridgeService) GetDepositStatus(ctx context.Context, depositCount, originNetworkID, destNetworkID uint32) (string, error) {
	var (
		claimTxHash string
	)
	// Get the claim tx hash
	claim, err := s.storage.GetClaim(ctx, depositCount, originNetworkID, destNetworkID, nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return "", err
		}
	} else {
		claimTxHash = claim.TxHash.String()
	}
	return claimTxHash, nil
}

// CheckAPI returns api version.
// Bridge rest API endpoint
func (s *bridgeService) CheckAPI(ctx context.Context, req *pb.CheckAPIRequest) (*pb.CheckAPIResponse, error) {
	metrics.CheckAPICounter()
	start := time.Now()
	defer func() {
		metrics.CheckAPILatency(time.Since(start))
	}()
	return &pb.CheckAPIResponse{
		Api: s.version,
	}, nil
}

// GetBridges returns bridges for the destination address both in L1 and L2.
// Bridge rest API endpoint
func (s *bridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	metrics.GetBridgesCounter()
	start := time.Now()
	defer func() {
		metrics.GetBridgesLatency(time.Since(start))
	}()
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}
	// Check if the user actually provided these fields
    var networkID, destinationNetworkID *uint32
    if req.NetId != nil {
        networkID = req.NetId  // User explicitly provided this value (could be 0 or any positive number)
        log.Infof("User provided net_id: %d", *networkID)
    }

    if req.DestNet != nil {
        destinationNetworkID = req.DestNet  // User explicitly provided this value
        log.Infof("User provided dest_net: %d", *destinationNetworkID)
    }

	totalCount, err := s.storage.GetDepositCount(ctx, req.DestAddr, networkID, destinationNetworkID, nil)
	if err != nil {
		return nil, err
	}
	deposits, err := s.storage.GetDeposits(ctx, req.DestAddr, networkID, destinationNetworkID, limit, req.Offset, nil)
	if err != nil {
		return nil, err
	}

	var pbDeposits []*pb.Deposit
	for _, deposit := range deposits {
		claimTxHash, err := s.GetDepositStatus(ctx, deposit.DepositCount, deposit.NetworkID, deposit.DestinationNetwork)
		if err != nil {
			return nil, err
		}
		mainnetFlag := deposit.NetworkID == 0
		var rollupIndex uint32
		if !mainnetFlag {
			rollupIndex = deposit.NetworkID - 1
		}
		localExitRootIndex := deposit.DepositCount
		pbDeposits = append(
			pbDeposits, &pb.Deposit{
				LeafType:      uint32(deposit.LeafType),
				OrigNet:       deposit.OriginalNetwork,
				OrigAddr:      deposit.OriginalAddress.Hex(),
				Amount:        deposit.Amount.String(),
				DestNet:       deposit.DestinationNetwork,
				DestAddr:      deposit.DestinationAddress.Hex(),
				BlockNum:      deposit.BlockNumber,
				DepositCnt:    deposit.DepositCount,
				NetworkId:     deposit.NetworkID,
				TxHash:        deposit.TxHash.String(),
				ClaimTxHash:   claimTxHash,
				Metadata:      "0x" + hex.EncodeToString(deposit.Metadata),
				ReadyForClaim: deposit.ReadyForClaim,
				GlobalIndex:   etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex).String(),
			},
		)
	}

	return &pb.GetBridgesResponse{
		Deposits: pbDeposits,
		TotalCnt: totalCount,
	}, nil
}

// GetClaims returns claims for the specific smart contract address both in L1 and L2.
// Bridge rest API endpoint
func (s *bridgeService) GetClaims(ctx context.Context, req *pb.GetClaimsRequest) (*pb.GetClaimsResponse, error) {
	metrics.GetClaimsCounter()
	start := time.Now()
	defer func() {
		metrics.GetClaimsLatency(time.Since(start))
	}()
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
	claims, err := s.storage.GetClaims(ctx, req.DestAddr, limit, req.Offset, nil) //nolint:mnd
	if err != nil {
		return nil, err
	}

	var pbClaims []*pb.Claim
	for _, claim := range claims {
		pbClaims = append(pbClaims, &pb.Claim{
			Index:       claim.Index,
			OrigNet:     claim.OriginalNetwork,
			OrigAddr:    claim.OriginalAddress.Hex(),
			Amount:      claim.Amount.String(),
			NetworkId:   claim.NetworkID,
			DestAddr:    claim.DestinationAddress.Hex(),
			BlockNum:    claim.BlockNumber,
			TxHash:      claim.TxHash.String(),
			RollupIndex: claim.RollupIndex,
			MainnetFlag: claim.MainnetFlag,
			GlobalIndex: claim.GlobalIndex,
		})
	}

	return &pb.GetClaimsResponse{
		Claims:   pbClaims,
		TotalCnt: totalCount,
	}, nil
}

// GetProof returns the merkle proof for the given deposit.
// Bridge rest API endpoint
func (s *bridgeService) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	metrics.GetProofCounter()
	start := time.Now()
	defer func() {
		metrics.GetProofLatency(time.Since(start))
	}()
	globalExitRoot, merkleProof, rollupMerkleProof, err := s.GetClaimProof(ctx, req.DepositCnt, req.NetId, nil)
	if err != nil {
		return nil, err
	}
	var (
		proof       []string
		rollupProof []string
	)
	if len(merkleProof) != len(rollupMerkleProof) {
		return nil, fmt.Errorf("proofs have different lengths. MerkleProof: %d. RollupMerkleProof: %d", len(merkleProof), len(rollupMerkleProof))
	}
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
		rollupProof = append(rollupProof, "0x"+hex.EncodeToString(rollupMerkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			RollupMerkleProof: rollupProof,
			MerkleProof:       proof,
			MainExitRoot:      globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot:    globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetBridge returns the bridge  with status whether it is able to send a claim transaction or not.
// Bridge rest API endpoint
func (s *bridgeService) GetBridge(ctx context.Context, req *pb.GetBridgeRequest) (*pb.GetBridgeResponse, error) {
	metrics.GetBridgeCounter()
	start := time.Now()
	defer func() {
		metrics.GetBridgeLatency(time.Since(start))
	}()
	deposit, err := s.storage.GetDeposit(ctx, req.DepositCnt, req.NetId, nil)
	if err != nil {
		return nil, err
	}

	claimTxHash, err := s.GetDepositStatus(ctx, req.DepositCnt, deposit.NetworkID, deposit.DestinationNetwork)
	if err != nil {
		return nil, err
	}
	mainnetFlag := deposit.NetworkID == 0
	var rollupIndex uint32
	if !mainnetFlag {
		rollupIndex = deposit.NetworkID - 1
	}
	localExitRootIndex := deposit.DepositCount

	return &pb.GetBridgeResponse{
		Deposit: &pb.Deposit{
			LeafType:      uint32(deposit.LeafType),
			OrigNet:       deposit.OriginalNetwork,
			OrigAddr:      deposit.OriginalAddress.Hex(),
			Amount:        deposit.Amount.String(),
			DestNet:       deposit.DestinationNetwork,
			DestAddr:      deposit.DestinationAddress.Hex(),
			BlockNum:      deposit.BlockNumber,
			DepositCnt:    deposit.DepositCount,
			NetworkId:     deposit.NetworkID,
			TxHash:        deposit.TxHash.String(),
			ClaimTxHash:   claimTxHash,
			Metadata:      "0x" + hex.EncodeToString(deposit.Metadata),
			ReadyForClaim: deposit.ReadyForClaim,
			GlobalIndex:   etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex).String(),
		},
	}, nil
}

// GetTokenWrapped returns the token wrapped created for a specific network
// Bridge rest API endpoint
func (s *bridgeService) GetTokenWrapped(ctx context.Context, req *pb.GetTokenWrappedRequest) (*pb.GetTokenWrappedResponse, error) {
	metrics.GetTokenWrappedCounter()
	start := time.Now()
	defer func() {
		metrics.GetTokenWrappedLatency(time.Since(start))
	}()
	tokenWrapped, err := s.storage.GetTokenWrapped(ctx, req.OrigNet, common.HexToAddress(req.OrigTokenAddr), nil)
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

func (s *bridgeService) GetProofByGER(ctx context.Context, req *pb.GetProofByGERRequest) (*pb.GetProofResponse, error) {
	metrics.GetProofByGERCounter()
	start := time.Now()
	defer func() {
		metrics.GetProofByGERLatency(time.Since(start))
	}()
	ger := common.HexToHash(req.Ger)
	globalExitRoot, merkleProof, rollupMerkleProof, err := s.GetClaimProofbyGER(ctx, req.DepositCnt, req.NetId, ger, nil)
	if err != nil {
		return nil, err
	}
	var (
		proof       []string
		rollupProof []string
	)
	if len(merkleProof) != len(rollupMerkleProof) {
		return nil, fmt.Errorf("proofs have different lengths. MerkleProof: %d. RollupMerkleProof: %d", len(merkleProof), len(rollupMerkleProof))
	}
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
		rollupProof = append(rollupProof, "0x"+hex.EncodeToString(rollupMerkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			RollupMerkleProof: rollupProof,
			MerkleProof:       proof,
			MainExitRoot:      globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot:    globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetPendingBridgesToClaim returns the pending bridges to claim by destination address, destination network and leaf type in L1 and L2's.
// Bridge rest API endpoint
func (s *bridgeService) GetPendingBridgesToClaim(ctx context.Context, req *pb.GetPendingBridgesRequest) (*pb.GetBridgesResponse, error) {
	metrics.GetPendingBridgesToClaimCounter()
	start := time.Now()
	defer func() {
		metrics.GetPendingBridgesToClaimLatency(time.Since(start))
	}()
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}
	destAddr := common.HexToAddress(req.DestAddr)
	deposits, totalDeposits, err := s.storage.GetPendingDepositsToClaim(ctx, destAddr, req.DestNet, req.LeafType, limit, req.Offset, -1, nil) // -1 means from all networks
	if err != nil {
		return nil, err
	}

	var pbDeposits []*pb.Deposit
	for _, deposit := range deposits {
		mainnetFlag := deposit.NetworkID == 0
		var rollupIndex uint32
		if !mainnetFlag {
			rollupIndex = deposit.NetworkID - 1
		}
		localExitRootIndex := deposit.DepositCount
		pbDeposits = append(
			pbDeposits, &pb.Deposit{
				LeafType:      uint32(deposit.LeafType),
				OrigNet:       deposit.OriginalNetwork,
				OrigAddr:      deposit.OriginalAddress.Hex(),
				Amount:        deposit.Amount.String(),
				DestNet:       deposit.DestinationNetwork,
				DestAddr:      deposit.DestinationAddress.Hex(),
				BlockNum:      deposit.BlockNumber,
				DepositCnt:    deposit.DepositCount,
				NetworkId:     deposit.NetworkID,
				TxHash:        deposit.TxHash.String(),
				ClaimTxHash:   "",
				Metadata:      "0x" + hex.EncodeToString(deposit.Metadata),
				ReadyForClaim: deposit.ReadyForClaim,
				GlobalIndex:   etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex).String(),
			},
		)
	}

	return &pb.GetBridgesResponse{
		Deposits: pbDeposits,
		TotalCnt: totalDeposits,
	}, nil
}

// GetProofV2 returns the merkle proof for the given deposit. It is compatible with Apps Team bridge
// Bridge rest API endpoint
func (s *bridgeService) GetProofV2(ctx context.Context, req *pb.GetProofV2Request) (*pb.GetProofResponse, error) {
	metrics.GetProofCounter()
	start := time.Now()
	defer func() {
		metrics.GetProofLatency(time.Since(start))
	}()
	globalExitRoot, merkleProof, rollupMerkleProof, err := s.GetClaimProof(ctx, req.DepositCount, req.NetworkId, nil)
	if err != nil {
		return nil, err
	}
	var (
		proof       []string
		rollupProof []string
	)
	if len(merkleProof) != len(rollupMerkleProof) {
		return nil, fmt.Errorf("proofs have different lengths. MerkleProof: %d. RollupMerkleProof: %d", len(merkleProof), len(rollupMerkleProof))
	}
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
		rollupProof = append(rollupProof, "0x"+hex.EncodeToString(rollupMerkleProof[i][:]))
	}

	return &pb.GetProofResponse{
		Proof: &pb.Proof{
			RollupMerkleProof: rollupProof,
			MerkleProof:       proof,
			MainExitRoot:      globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot:    globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetSyncStatus returns the sync Status of all networks.
// Bridge rest API endpoint
func (s *bridgeService) GetSyncStatus(ctx context.Context, req *pb.GetSyncStatusRequest) (*pb.GetSyncStatusResponse, error) {
	status, err := s.storage.GetSyncStatus(ctx, nil)
	if err != nil {
		return nil, err
	}
	var returnSyncStatus []*pb.SyncStatus
	for _, syncStatus := range status {
		ss := &pb.SyncStatus{
			NetworkId:      syncStatus.NetworkID,
			Percentage:     syncStatus.Percentage,
			RemainingBlocks: syncStatus.RemainingBlocks,
			Synced:         syncStatus.Synced,
		}
		returnSyncStatus = append(returnSyncStatus, ss)
	}

	return &pb.GetSyncStatusResponse{
		Sync: returnSyncStatus,
	}, nil
}

// GetBackwardLETData returns the data needed to backwardLET in the smc.
// Bridge rest API endpoint
func (s *bridgeService) GetBackwardLETData(ctx context.Context, req *pb.GetBackwardLETDataRequest) (*pb.GetBackwardLETDataResponse, error) {
	indexToRemove := req.DepositCnt
	networkID := req.NetId
	if networkID == 0 {
		return nil, fmt.Errorf("networkID cannot be 0 for backward LET")
	}
	leafHash, err := s.getLeaf(ctx, indexToRemove, networkID)
	if err != nil {
		return nil, err
	}
	root, err := s.storage.GetLastComputedRoot(ctx, networkID, nil)
	if err != nil {
		return nil, err
	}
	rollupMerkleProof, err := s.getProof(ctx, indexToRemove, root, nil)
	if err != nil {
		return nil, err
	}
	frontier, err := s.computeFrontierFromProof(indexToRemove, rollupMerkleProof)
	if err != nil {
		return nil, err
	}
	var rollupProof, frontierString []string
	for i := range rollupMerkleProof {
		rollupProof = append(rollupProof, "0x"+hex.EncodeToString(rollupMerkleProof[i][:]))
	}
	for i := range frontier {
		frontierString = append(frontierString, "0x"+hex.EncodeToString(frontier[i][:]))
	}
	return &pb.GetBackwardLETDataResponse{
		Root: 			   root.String(),
		LeafHash:          leafHash.String(),
		RollupMerkleProof: rollupProof,
		Frontier:          frontierString,
	}, nil
}

// ComputeFrontierFromProof build the frontier for backwardLET.
// Rule: in each level i, if the bit i of n is 1 => frontier[i] = proof[i],
// If it is 0 => frontier[i] = 0x00..00.
func (s *bridgeService) computeFrontierFromProof(n uint32, proof [][bridgectrl.KeyLen]byte) ([][bridgectrl.KeyLen]byte, error) {
	if len(proof) != int(s.height) {
		return [][bridgectrl.KeyLen]byte{}, fmt.Errorf("proof length %d does not match tree height %d", len(proof), s.height)
	}
	var zero [32]byte
	var frontier [][bridgectrl.KeyLen]byte
	for i := uint(0); i < uint(s.height); i++ {
		if ((n >> i) & 1) == 1 {
			frontier = append(frontier, proof[i])
		} else {
			frontier = append(frontier, zero)
		}
	}
	return frontier, nil
}

func (s *bridgeService) getLeaf(ctx context.Context, index uint32, networkID uint32) (common.Hash, error) {
	deposit, err := s.storage.GetDeposit(ctx, index, networkID, nil)
	if err != nil {
		if errors.Is(err, gerror.ErrStorageNotFound) {
			return common.Hash{}, nil
		}
		return common.Hash{}, err
	}
	return bridgectrl.HashDeposit(deposit), nil
}
