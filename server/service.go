package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/estimatetime"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/pushtask"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

const (
	defaultErrorCode   = 1
	defaultSuccessCode = 0
	defaultMinDuration = 1
)

type bridgeService struct {
	storage             BridgeServiceStorage
	redisStorage        redisstorage.RedisStorage
	mainCoinsCache      localcache.MainCoinsCache
	networkIDs          map[uint]uint8
	height              uint8
	defaultPageLimit    uint32
	maxPageLimit        uint32
	version             string
	cache               *lru.Cache[string, [][]byte]
	estTimeCalculator   estimatetime.Calculator
	messagePushProducer messagepush.KafkaProducer
	pb.UnimplementedBridgeServiceServer
}

// NewBridgeService creates new bridge service.
func NewBridgeService(cfg Config, height uint8, networks []uint, storage interface{}, redisStorage redisstorage.RedisStorage,
	mainCoinsCache localcache.MainCoinsCache, estTimeCalc estimatetime.Calculator) *bridgeService {
	var networkIDs = make(map[uint]uint8)
	for i, network := range networks {
		networkIDs[network] = uint8(i)
	}
	cache, err := lru.New[string, [][]byte](cfg.CacheSize)
	if err != nil {
		panic(err)
	}
	return &bridgeService{
		storage:           storage.(BridgeServiceStorage),
		redisStorage:      redisStorage,
		mainCoinsCache:    mainCoinsCache,
		estTimeCalculator: estTimeCalc,
		height:            height,
		networkIDs:        networkIDs,
		defaultPageLimit:  cfg.DefaultPageLimit,
		maxPageLimit:      cfg.MaxPageLimit,
		version:           cfg.BridgeVersion,
		cache:             cache,
	}
}

func (s *bridgeService) WithMessagePushProducer(producer messagepush.KafkaProducer) *bridgeService {
	s.messagePushProducer = producer
	return s
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
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
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
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
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

func (s *bridgeService) GetSmtProof(ctx context.Context, req *pb.GetSmtProofRequest) (*pb.CommonProofResponse, error) {
	globalExitRoot, merkleProof, err := s.GetClaimProof(uint(req.Index), uint(req.FromChain), nil)
	if err != nil {
		return &pb.CommonProofResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  err.Error(),
		}, nil
	}
	var proof []string
	for i := 0; i < len(merkleProof); i++ {
		proof = append(proof, "0x"+hex.EncodeToString(merkleProof[i][:]))
	}

	return &pb.CommonProofResponse{
		Code: defaultSuccessCode,
		Data: &pb.ProofDetail{
			SmtProof:        proof,
			MainnetExitRoot: globalExitRoot.ExitRoots[0].Hex(),
			RollupExitRoot:  globalExitRoot.ExitRoots[1].Hex(),
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

// GetCoinPrice returns the price for each coin symbol in the request
// Bridge rest API endpoint
func (s *bridgeService) GetCoinPrice(ctx context.Context, req *pb.GetCoinPriceRequest) (*pb.CommonCoinPricesResponse, error) {
	priceList, err := s.redisStorage.GetCoinPrice(ctx, req.SymbolInfos)
	if err != nil {
		log.Errorf("get coin price from redis failed for symbol: %v, error: %v", req.SymbolInfos, err)
		return &pb.CommonCoinPricesResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  gerror.ErrInternalErrorForRpcCall.Error(),
		}, nil
	}
	return &pb.CommonCoinPricesResponse{
		Code: defaultSuccessCode,
		Data: priceList,
	}, nil
}

// GetMainCoins returns the info of the main coins in a network
// Bridge rest API endpoint
func (s *bridgeService) GetMainCoins(ctx context.Context, req *pb.GetMainCoinsRequest) (*pb.CommonCoinsResponse, error) {
	coins, err := s.mainCoinsCache.GetMainCoinsByNetwork(ctx, req.NetworkId)
	if err != nil {
		log.Errorf("get main coins from cache failed for net: %v, error: %v", req.NetworkId, err)
		return &pb.CommonCoinsResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  gerror.ErrInternalErrorForRpcCall.Error(),
		}, nil
	}
	return &pb.CommonCoinsResponse{
		Code: defaultSuccessCode,
		Data: coins,
	}, nil
}

// GetPendingTransactions returns the pending transactions of an account
// Bridge rest API endpoint
func (s *bridgeService) GetPendingTransactions(ctx context.Context, req *pb.GetPendingTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}

	deposits, err := s.storage.GetPendingTransactions(ctx, req.DestAddr, uint(limit+1), uint(req.Offset), uint(utils.LeafTypeAsset), nil)
	if err != nil {
		log.Errorf("get pending tx failed for address: %v, limit: %v, offset: %v, error: %v", req.DestAddr, limit, req.Offset, err)
		return &pb.CommonTransactionsResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  gerror.ErrInternalErrorForRpcCall.Error(),
		}, nil
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[:limit]
	}

	l1BlockNum, _ := s.redisStorage.GetL1BlockNum(ctx)
	l2CommitBlockNum, _ := s.redisStorage.GetCommitMaxBlockNum(ctx)
	l2AvgCommitDuration := pushtask.GetAvgCommitDuration(ctx, s.redisStorage)
	l2AvgVerifyDuration := pushtask.GetAvgVerifyDuration(ctx, s.redisStorage)
	currTime := time.Now()

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		transaction := &pb.Transaction{
			FromChain:    uint32(deposit.NetworkID),
			ToChain:      uint32(deposit.DestinationNetwork),
			BridgeToken:  deposit.OriginalAddress.Hex(),
			TokenAmount:  deposit.Amount.String(),
			EstimateTime: s.estTimeCalculator.Get(deposit.NetworkID),
			Time:         uint64(deposit.Time.UnixMilli()),
			TxHash:       deposit.TxHash.String(),
			FromChainId:  utils.GetChainIdByNetworkId(deposit.NetworkID),
			ToChainId:    utils.GetChainIdByNetworkId(deposit.DestinationNetwork),
			Id:           deposit.Id,
			Index:        uint64(deposit.DepositCount),
			Metadata:     "0x" + hex.EncodeToString(deposit.Metadata),
			BlockNumber:  deposit.BlockNumber,
		}
		transaction.Status = uint32(pb.TransactionStatus_TX_CREATED)
		if deposit.ReadyForClaim {
			transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM)
			// For L1->L2, if backend is trying to auto-claim, set the status to 0 to block the user from manual-claim
			// When the auto-claim failed, set status to 1 to let the user claim manually through front-end
			if deposit.NetworkID == 0 {
				mTx, err := s.storage.GetClaimTxById(ctx, deposit.DepositCount, nil)
				if err == nil && mTx.Status != ctmtypes.MonitoredTxStatusFailed {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			}
		} else {
			// For L1->L2, when ready_for_claim is false, but there have been more than 64 block confirmations,
			// should also display the status as "L2 executing" (pending auto claim)
			if deposit.NetworkID == 0 {
				if l1BlockNum-deposit.BlockNumber >= utils.L1TargetBlockConfirmations {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			} else {
				if l2CommitBlockNum >= deposit.BlockNumber {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_VERIFICATION)
				}
				s.setDurationForL2Deposit(ctx, l2AvgCommitDuration, l2AvgVerifyDuration, currTime, transaction, deposit.Time)
			}
		}
		pbTransactions = append(pbTransactions, transaction)
	}
	return &pb.CommonTransactionsResponse{
		Code: defaultSuccessCode,
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

func (s *bridgeService) setDurationForL2Deposit(ctx context.Context, l2AvgCommitDuration uint64, l2AvgVerifyDuration uint64, currTime time.Time,
	tx *pb.Transaction, depositCreateTime time.Time) {
	var duration int
	if tx.Status == uint32(pb.TransactionStatus_TX_CREATED) {
		duration = pushtask.GetLeftCommitTime(depositCreateTime, l2AvgCommitDuration, currTime)
	} else {
		duration = pushtask.GetLeftVerifyTime(ctx, s.redisStorage, tx.BlockNumber, depositCreateTime, l2AvgCommitDuration, l2AvgVerifyDuration, currTime)
	}
	if duration <= 0 {
		log.Debugf("count EstimateTime for L2 -> L1 over range, so use min default duration: %v", defaultMinDuration)
		tx.EstimateTime = uint32(defaultMinDuration)
		return
	}
	tx.EstimateTime = uint32(duration)
}

// GetAllTransactions returns all the transactions of an account, similar to GetBridges
// Bridge rest API endpoint
func (s *bridgeService) GetAllTransactions(ctx context.Context, req *pb.GetAllTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}

	deposits, err := s.storage.GetDepositsWithLeafType(ctx, req.DestAddr, uint(limit+1), uint(req.Offset), uint(utils.LeafTypeAsset), nil)
	if err != nil {
		log.Errorf("get deposits from db failed for address: %v, limit: %v, offset: %v, error: %v", req.DestAddr, limit, req.Offset, err)
		return &pb.CommonTransactionsResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  gerror.ErrInternalErrorForRpcCall.Error(),
		}, nil
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[0:limit]
	}

	l1BlockNum, _ := s.redisStorage.GetL1BlockNum(ctx)
	l2CommitBlockNum, _ := s.redisStorage.GetCommitMaxBlockNum(ctx)
	l2AvgCommitDuration := pushtask.GetAvgCommitDuration(ctx, s.redisStorage)
	l2AvgVerifyDuration := pushtask.GetAvgVerifyDuration(ctx, s.redisStorage)
	currTime := time.Now()

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		transaction := &pb.Transaction{
			FromChain:    uint32(deposit.NetworkID),
			ToChain:      uint32(deposit.DestinationNetwork),
			BridgeToken:  deposit.OriginalAddress.Hex(),
			TokenAmount:  deposit.Amount.String(),
			EstimateTime: s.estTimeCalculator.Get(deposit.NetworkID),
			Time:         uint64(deposit.Time.UnixMilli()),
			TxHash:       deposit.TxHash.String(),
			FromChainId:  utils.GetChainIdByNetworkId(deposit.NetworkID),
			ToChainId:    utils.GetChainIdByNetworkId(deposit.DestinationNetwork),
			Id:           deposit.Id,
			Index:        uint64(deposit.DepositCount),
			Metadata:     "0x" + hex.EncodeToString(deposit.Metadata),
			BlockNumber:  deposit.BlockNumber,
		}
		transaction.Status = uint32(pb.TransactionStatus_TX_CREATED) // Not ready for claim
		if deposit.ReadyForClaim {
			// Check whether it has been claimed or not
			claim, err := s.storage.GetClaim(ctx, deposit.DepositCount, deposit.DestinationNetwork, nil)
			transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM) // Ready but not claimed
			if err != nil {
				if !errors.Is(err, gerror.ErrStorageNotFound) {
					return &pb.CommonTransactionsResponse{
						Code: defaultErrorCode,
						Data: nil,
						Msg:  errors.Wrap(err, "load claim error").Error(),
					}, nil
				}
				// For L1->L2, if backend is trying to auto-claim, set the status to 0 to block the user from manual-claim
				// When the auto-claim failed, set status to 1 to let the user claim manually through front-end
				if deposit.NetworkID == 0 {
					mTx, err := s.storage.GetClaimTxById(ctx, deposit.DepositCount, nil)
					if err == nil && mTx.Status != ctmtypes.MonitoredTxStatusFailed {
						transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
					}
				}
			} else {
				transaction.Status = uint32(pb.TransactionStatus_TX_CLAIMED) // Claimed
				transaction.ClaimTxHash = claim.TxHash.String()
				transaction.ClaimTime = uint64(claim.Time.UnixMilli())
			}
		} else {
			// For L1->L2, when ready_for_claim is false, but there have been more than 64 block confirmations,
			// should also display the status as "L2 executing" (pending auto claim)
			if deposit.NetworkID == 0 {
				if l1BlockNum-deposit.BlockNumber >= utils.L1TargetBlockConfirmations {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM)
				}
			} else {
				if l2CommitBlockNum >= deposit.BlockNumber {
					transaction.Status = uint32(pb.TransactionStatus_TX_PENDING_VERIFICATION)
				}
				s.setDurationForL2Deposit(ctx, l2AvgCommitDuration, l2AvgVerifyDuration, currTime, transaction, deposit.Time)
			}
		}
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonTransactionsResponse{
		Code: defaultSuccessCode,
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetNotReadyTransactions returns all deposit transactions with ready_for_claim = false
func (s *bridgeService) GetNotReadyTransactions(ctx context.Context, req *pb.GetNotReadyTransactionsRequest) (*pb.CommonTransactionsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}

	deposits, err := s.storage.GetNotReadyTransactions(ctx, uint(limit+1), uint(req.Offset), nil)
	if err != nil {
		return &pb.CommonTransactionsResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  err.Error(),
		}, nil
	}

	hasNext := len(deposits) > int(limit)
	if hasNext {
		deposits = deposits[0:limit]
	}

	var pbTransactions []*pb.Transaction
	for _, deposit := range deposits {
		transaction := &pb.Transaction{
			FromChain:    uint32(deposit.NetworkID),
			ToChain:      uint32(deposit.DestinationNetwork),
			BridgeToken:  deposit.OriginalAddress.Hex(),
			TokenAmount:  deposit.Amount.String(),
			EstimateTime: s.estTimeCalculator.Get(deposit.NetworkID),
			Status:       0,
			Time:         uint64(deposit.Time.UnixMilli()),
			TxHash:       deposit.TxHash.String(),
			FromChainId:  utils.GetChainIdByNetworkId(deposit.NetworkID),
			ToChainId:    utils.GetChainIdByNetworkId(deposit.DestinationNetwork),
			Id:           deposit.Id,
			Index:        uint64(deposit.DepositCount),
			Metadata:     "0x" + hex.EncodeToString(deposit.Metadata),
			BlockNumber:  deposit.BlockNumber,
		}
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonTransactionsResponse{
		Code: defaultSuccessCode,
		Data: &pb.TransactionDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetMonitoredTxsByStatus returns list of monitored transactions, filtered by status
func (s *bridgeService) GetMonitoredTxsByStatus(ctx context.Context, req *pb.GetMonitoredTxsByStatusRequest) (*pb.CommonMonitoredTxsResponse, error) {
	limit := req.Limit
	if limit == 0 {
		limit = s.defaultPageLimit
	}
	if limit > s.maxPageLimit {
		limit = s.maxPageLimit
	}

	mTxs, err := s.storage.GetClaimTxsByStatusWithLimit(ctx, []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatus(req.Status)}, uint(limit+1), uint(req.Offset), nil)
	if err != nil {
		return &pb.CommonMonitoredTxsResponse{
			Code: defaultErrorCode,
			Data: nil,
			Msg:  err.Error(),
		}, nil
	}

	hasNext := len(mTxs) > int(limit)
	if hasNext {
		mTxs = mTxs[0:limit]
	}

	var pbTransactions []*pb.MonitoredTx
	for _, mTx := range mTxs {
		transaction := &pb.MonitoredTx{
			Id:        uint64(mTx.DepositID),
			From:      "0x" + mTx.From.String(),
			To:        "0x" + mTx.To.String(),
			Nonce:     mTx.Nonce,
			Value:     mTx.Value.String(),
			Data:      "0x" + hex.EncodeToString(mTx.Data),
			Gas:       mTx.Gas,
			GasPrice:  mTx.GasPrice.String(),
			Status:    string(mTx.Status),
			CreatedAt: uint64(mTx.CreatedAt.UnixMilli()),
			UpdatedAt: uint64(mTx.UpdatedAt.UnixMilli()),
		}
		for h := range mTx.History {
			transaction.History = append(transaction.History, h.String())
		}
		pbTransactions = append(pbTransactions, transaction)
	}

	return &pb.CommonMonitoredTxsResponse{
		Code: defaultSuccessCode,
		Data: &pb.MonitoredTxsDetail{HasNext: hasNext, Transactions: pbTransactions},
	}, nil
}

// GetEstimateTime returns the estimated deposit waiting time for L1 and L2
func (s *bridgeService) GetEstimateTime(ctx context.Context, req *pb.GetEstimateTimeRequest) (*pb.CommonEstimateTimeResponse, error) {
	return &pb.CommonEstimateTimeResponse{
		Code: defaultSuccessCode,
		Data: []uint32{s.estTimeCalculator.Get(0), s.estTimeCalculator.Get(1)},
	}, nil
}

func (s *bridgeService) GetFakePushMessages(ctx context.Context, req *pb.GetFakePushMessagesRequest) (*pb.GetFakePushMessagesResponse, error) {
	if s.messagePushProducer == nil {
		return &pb.GetFakePushMessagesResponse{
			Code: defaultErrorCode,
			Msg:  "producer is nil",
		}, nil
	}

	return &pb.GetFakePushMessagesResponse{
		Code: defaultSuccessCode,
		Data: s.messagePushProducer.GetFakeMessages(req.Topic),
	}, nil
}
