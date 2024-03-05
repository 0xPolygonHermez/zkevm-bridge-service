package server

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v4"
)

type bridgeService struct {
	storage          BridgeServiceStorage
	networkIDs       map[uint]uint8
	height           uint8
	defaultPageLimit apolloconfig.Entry[uint32]
	maxPageLimit     apolloconfig.Entry[uint32]
	version          string
	cache            *lru.Cache[string, [][]byte]
	pb.UnimplementedBridgeServiceServer

	// X1
	redisStorage        redisstorage.RedisStorage
	mainCoinsCache      localcache.MainCoinsCache
	nodeClients         map[uint]*utils.Client
	auths               map[uint]*bind.TransactOpts
	messagePushProducer messagepush.KafkaProducer
}

// NewBridgeService creates new bridge service.
func NewBridgeService(cfg Config, height uint8, networks []uint, l2Clients []*utils.Client, l2Auths []*bind.TransactOpts, storage interface{}) *bridgeService {
	var networkIDs = make(map[uint]uint8)
	var nodeClients = make(map[uint]*utils.Client, len(networks))
	var authMap = make(map[uint]*bind.TransactOpts, len(networks))
	for i, network := range networks {
		networkIDs[network] = uint8(i)
		if i > 0 {
			nodeClients[network] = l2Clients[i-1]
			authMap[network] = l2Auths[i-1]
		}
	}
	cache, err := lru.New[string, [][]byte](cfg.CacheSize)
	if err != nil {
		panic(err)
	}
	return &bridgeService{
		storage:          storage.(BridgeServiceStorage),
		height:           height,
		networkIDs:       networkIDs,
		nodeClients:      nodeClients,
		auths:            authMap,
		defaultPageLimit: apolloconfig.NewIntEntry("BridgeServer.DefaultPageLimit", cfg.DefaultPageLimit),
		maxPageLimit:     apolloconfig.NewIntEntry("BridgeServer.MaxPageLimit", cfg.MaxPageLimit),
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
func (s *bridgeService) getNode(ctx context.Context, parentHash [bridgectrl.KeyLen]byte, dbTx pgx.Tx) (left, right [bridgectrl.KeyLen]byte, err error) {
	value, ok := s.cache.Get(string(parentHash[:]))
	if !ok {
		var err error
		value, err = s.storage.Get(ctx, parentHash[:], dbTx)
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
func (s *bridgeService) getProof(index uint, root [bridgectrl.KeyLen]byte, dbTx pgx.Tx) ([][bridgectrl.KeyLen]byte, error) {
	var siblings [][bridgectrl.KeyLen]byte

	cur := root
	ctx := context.Background()
	// It starts in height-1 because 0 is the level of the leafs
	for h := int(s.height - 1); h >= 0; h-- {
		left, right, err := s.getNode(ctx, cur, dbTx)
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

// GetClaimProof returns the merkle proof to claim the given deposit.
func (s *bridgeService) GetClaimProof(depositCnt, networkID uint, dbTx pgx.Tx) (*etherman.GlobalExitRoot, [][bridgectrl.KeyLen]byte, error) {
	ctx := context.Background()

	if dbTx == nil { // if the call comes from the rest API
		deposit, err := s.storage.GetDeposit(ctx, depositCnt, networkID, nil)
		if err != nil {
			log.Errorf("failed to get deposit from db for GetClaimProof, depositCnt: %v, networkID: %v, error: %v", depositCnt, networkID, err)
			return nil, nil, gerror.ErrInternalErrorForRpcCall
		}

		if !deposit.ReadyForClaim {
			return nil, nil, gerror.ErrDepositNotSynced
		}
	}

	tID, err := s.getNetworkID(networkID)
	if err != nil {
		return nil, nil, err
	}

	globalExitRoot, err := s.storage.GetLatestExitRoot(ctx, tID != 0, dbTx)
	if err != nil {
		log.Errorf("get latest exit root failed fot network: %v, error: %v", networkID, err)
		return nil, nil, gerror.ErrInternalErrorForRpcCall
	}

	merkleProof, err := s.getProof(depositCnt, globalExitRoot.ExitRoots[tID], dbTx)
	if err != nil {
		return nil, nil, fmt.Errorf("getting the proof failed, error: %v, network: %d", err, networkID)
	}

	return globalExitRoot, merkleProof, nil
}

// GetDepositStatus returns deposit with ready_for_claim status.
func (s *bridgeService) GetDepositStatus(ctx context.Context, depositCount uint, destNetworkID uint) (string, error) {
	var (
		claimTxHash string
	)
	// Get the claim tx hash
	claim, err := s.storage.GetClaim(ctx, depositCount, destNetworkID, nil)
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
	return &pb.CheckAPIResponse{
		Api: s.version,
	}, nil
}

// GetBridges returns bridges for the destination address both in L1 and L2.
// Bridge rest API endpoint
func (s *bridgeService) GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}
	totalCount, err := s.storage.GetDepositCount(ctx, req.DestAddr, nil)
	if err != nil {
		log.Errorf("get deposit count from db failed for address: %v, err: %v", req.DestAddr, err)
		return nil, gerror.ErrInternalErrorForRpcCall
	}
	deposits, err := s.storage.GetDeposits(ctx, req.DestAddr, uint(limit), uint(req.Offset), nil)
	if err != nil {
		return nil, err
	}

	var pbDeposits []*pb.Deposit
	for _, deposit := range deposits {
		claimTxHash, err := s.GetDepositStatus(ctx, deposit.DepositCount, deposit.DestinationNetwork)
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
				ReadyForClaim: deposit.ReadyForClaim,
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
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit.Get()
	}
	if limit > s.maxPageLimit.Get() {
		limit = s.maxPageLimit.Get()
	}
	totalCount, err := s.storage.GetClaimCount(ctx, req.DestAddr, nil)
	if err != nil {
		log.Errorf("get claim count from db for address: %v, err: %v", req.DestAddr, err)
		return nil, gerror.ErrInternalErrorForRpcCall
	}
	claims, err := s.storage.GetClaims(ctx, req.DestAddr, uint(limit), uint(req.Offset), nil) //nolint:gomnd
	if err != nil {
		log.Errorf("get claim infos from db for address: %v, err: %v", req.DestAddr, err)
		return nil, gerror.ErrInternalErrorForRpcCall
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

// GetProof returns the merkle proof for the given deposit.
// Bridge rest API endpoint
func (s *bridgeService) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	globalExitRoot, merkleProof, err := s.GetClaimProof(uint(req.DepositCnt), uint(req.NetId), nil)
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
			MainExitRoot:   globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot: globalExitRoot.ExitRoots[1].Hex(),
		},
	}, nil
}

// GetBridge returns the bridge  with status whether it is able to send a claim transaction or not.
// Bridge rest API endpoint
func (s *bridgeService) GetBridge(ctx context.Context, req *pb.GetBridgeRequest) (*pb.GetBridgeResponse, error) {
	deposit, err := s.storage.GetDeposit(ctx, uint(req.DepositCnt), uint(req.NetId), nil)
	if err != nil {
		log.Errorf("get deposit info failed for depositCnt: %v, net: %v, error: %v", req.DepositCnt, req.NetId, err)
		return nil, gerror.ErrInternalErrorForRpcCall
	}

	claimTxHash, err := s.GetDepositStatus(ctx, uint(req.DepositCnt), deposit.DestinationNetwork)
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
			ReadyForClaim: deposit.ReadyForClaim,
		},
	}, nil
}

// GetTokenWrapped returns the token wrapped created for a specific network
// Bridge rest API endpoint
func (s *bridgeService) GetTokenWrapped(ctx context.Context, req *pb.GetTokenWrappedRequest) (*pb.GetTokenWrappedResponse, error) {
	tokenWrapped, err := s.storage.GetTokenWrapped(ctx, uint(req.OrigNet), common.HexToAddress(req.OrigTokenAddr), nil)
	if err != nil {
		log.Errorf("get token wrap info failed, origNet: %v, origTokenAddr: %v, error: %v", req.OrigNet, req.OrigTokenAddr, err)
		return nil, gerror.ErrInternalErrorForRpcCall
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
