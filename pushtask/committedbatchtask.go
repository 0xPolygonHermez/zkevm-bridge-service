package pushtask

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	committedBatchCacheRefreshInterval = 10 * time.Second
	l2NetWorkId                        = 1
	l1PendingDepositQueryLimit         = 100
	syncL1CommittedBatchLockKey        = "sync_l1_committed_batch_lock"
)

var (
	minCommitDuration     = apolloconfig.NewIntEntry[uint64]("pushtask.minCommitDuration", 2)      //nolint:gomnd
	defaultCommitDuration = apolloconfig.NewIntEntry[uint64]("pushtask.defaultCommitDuration", 10) //nolint:gomnd
	commitDurationListLen = apolloconfig.NewIntEntry[int]("pushtask.commitDurationListLen", 5)     //nolint:gomnd
)

type CommittedBatchHandler struct {
	rpcUrl              string
	client              *ethclient.Client
	storage             DBStorage
	redisStorage        redisstorage.RedisStorage
	messagePushProducer messagepush.KafkaProducer
}

func NewCommittedBatchHandler(rpcUrl string, storage interface{}, redisStorage redisstorage.RedisStorage, producer messagepush.KafkaProducer) (*CommittedBatchHandler, error) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, errors.Wrap(err, "eth-client dial error")
	}
	return &CommittedBatchHandler{
		rpcUrl:              rpcUrl,
		client:              client,
		storage:             storage.(DBStorage),
		redisStorage:        redisStorage,
		messagePushProducer: producer,
	}, nil
}

func (ins *CommittedBatchHandler) Start(ctx context.Context) {
	log.Debugf("Starting processSyncCommitBatchTask, interval:%v", committedBatchCacheRefreshInterval)
	ticker := time.NewTicker(committedBatchCacheRefreshInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ins.processSyncCommitBatchTask(ctx)
		}
	}
}

func (ins *CommittedBatchHandler) processSyncCommitBatchTask(ctx context.Context) {
	lock, err := ins.redisStorage.TryLock(ctx, syncL1CommittedBatchLockKey)
	if err != nil {
		log.Errorf("sync latest commit batch lock error, so kip, error: %v", err)
		return
	}
	if !lock {
		log.Infof("sync latest commit batch lock failed, another is running, so kip, error: %v", err)
		return
	}
	defer func() {
		err = ins.redisStorage.ReleaseLock(ctx, syncL1CommittedBatchLockKey)
		if err != nil {
			log.Errorf("ReleaseLock key[%v] error: %v", syncL1CommittedBatchLockKey, err)
		}
	}()
	log.Infof("start to sync latest commit batch")
	latestBatchNum, err := QueryLatestCommitBatch(ins.rpcUrl)
	if err != nil {
		log.Warnf("query latest commit batch num error, so skip sync latest commit batch!")
		return
	}
	now := time.Now().Unix()
	isBatchLegal, err := ins.checkLatestBatchLegal(ctx, latestBatchNum)
	if err != nil {
		log.Warnf("check latest commit batch num error, so skip sync latest commit batch!")
		return
	}
	if !isBatchLegal {
		log.Infof("latest commit batch num is un-legal, so skip sync latest commit batch!")
		return
	}
	oldMaxBlockNum, maxBlockNum, err := ins.freshRedisByLatestBatch(ctx, latestBatchNum, now)
	if err != nil {
		log.Warnf("fresh redis for latest commit batch num error, so skip sync latest commit batch!")
		return
	}
	err = ins.pushStatusChangedMsg(ctx, latestBatchNum, oldMaxBlockNum, maxBlockNum)
	if err != nil {
		log.Warnf("push msg for latest commit batch num error, so skip sync latest commit batch!")
		return
	}
	log.Infof("success process all thing for sync latest commit batch num %v", latestBatchNum)
}

func (ins *CommittedBatchHandler) freshRedisByLatestBatch(ctx context.Context, latestBatchNum uint64, currTimestamp int64) (uint64, uint64, error) {
	err := ins.freshRedisForMaxCommitBatchNum(ctx, latestBatchNum)
	if err != nil {
		log.Errorf("fresh redis for max commit batch num err, num: %v, err: %v", latestBatchNum, err)
		return 0, 0, err
	}
	err = ins.freshRedisForAvgCommitDuration(ctx, latestBatchNum, currTimestamp)
	if err != nil {
		log.Errorf("fresh redis for avg commit duration err, num: %v, err: %v", latestBatchNum, err)
		return 0, 0, err
	}
	oldMaxBlockNum, maxBlockNum, err := ins.freshRedisForMaxCommitBlockNum(ctx, latestBatchNum)
	if err != nil {
		log.Errorf("fresh redis for max commit block num err, num: %v, err: %v", latestBatchNum, err)
		return 0, 0, err
	}
	if maxBlockNum == 0 {
		log.Infof("batch has none transaction and block, so return, batch: %v", latestBatchNum)
		return 0, 0, nil
	}
	err = ins.cacheEveryLockCommitTimeForBatch(ctx, oldMaxBlockNum, maxBlockNum, currTimestamp)
	if err != nil {
		log.Errorf("cache every block commit time for batch error, num: %v, err: %v", latestBatchNum, err)
		return oldMaxBlockNum, maxBlockNum, err
	}
	log.Infof("success fresh redis cache of latest committed batch by batch %v", latestBatchNum)
	return oldMaxBlockNum, maxBlockNum, nil
}

func (ins *CommittedBatchHandler) getMaxBlockNumByBatchNum(ctx context.Context, batchNum uint64) (uint64, error) {
	maxBlockHash, err := QueryMaxBlockHashByBatchNum(ins.rpcUrl, batchNum)
	if err != nil {
		return 0, err
	}
	if maxBlockHash == "" {
		return 0, err
	}
	maxBlockNum, err := QueryBlockNumByBlockHash(ctx, ins.client, maxBlockHash)
	if err != nil {
		return 0, err
	}
	return maxBlockNum, nil
}

func (ins *CommittedBatchHandler) checkLatestBatchLegal(ctx context.Context, latestBatchNum uint64) (bool, error) {
	oldBatchNum, err := ins.redisStorage.GetCommitBatchNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("failed to get batch num from redis, so skip, error: %v", err)
		return false, errors.Wrap(err, "failed to get batch num from redis")
	}
	if oldBatchNum >= latestBatchNum {
		log.Infof("redis committed batch number: %v gt latest num: %v, so skip", oldBatchNum, latestBatchNum)
		return false, nil
	}
	log.Infof("latest committed batch num check pass, num: %v", latestBatchNum)
	return true, nil
}

func (ins *CommittedBatchHandler) freshRedisForMaxCommitBatchNum(ctx context.Context, latestBatchNum uint64) error {
	return ins.redisStorage.SetCommitBatchNum(ctx, latestBatchNum)
}

func (ins *CommittedBatchHandler) freshRedisForMaxCommitBlockNum(ctx context.Context, latestBatchNum uint64) (uint64, uint64, error) {
	oldMaxBlockNum, err := ins.redisStorage.GetCommitMaxBlockNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, 0, err
	}
	maxBlockNum, err := ins.getMaxBlockNumByBatchNum(ctx, latestBatchNum)
	if err != nil {
		return 0, 0, err
	}
	if maxBlockNum == 0 {
		return 0, 0, nil
	}
	err = ins.redisStorage.SetCommitMaxBlockNum(ctx, maxBlockNum)
	if err != nil {
		return 0, 0, err
	}
	log.Infof("success to set max commit block num: %v", maxBlockNum)
	return oldMaxBlockNum, maxBlockNum, nil
}

func (ins *CommittedBatchHandler) cacheEveryLockCommitTimeForBatch(ctx context.Context, oldMaxBlockNum uint64, maxBlockNum uint64, currTimestamp int64) error {
	for i := oldMaxBlockNum + 1; i <= maxBlockNum; i++ {
		err := ins.redisStorage.SetL2BlockCommitTime(ctx, i, currTimestamp)
		if err != nil {
			log.Errorf("failed to set commit time for block: %v", i)
		}
	}
	return nil
}

func (ins *CommittedBatchHandler) freshRedisForAvgCommitDuration(ctx context.Context, latestBatchNum uint64, currTimestamp int64) error {
	err := ins.redisStorage.LPushCommitTime(ctx, currTimestamp)
	if err != nil {
		return err
	}
	listLen, err := ins.redisStorage.LLenCommitTimeList(ctx)
	if err != nil {
		return err
	}
	if listLen <= int64(commitDurationListLen.Get()) {
		log.Infof("redis duration list is not enough, so skip count the avg duration!")
		return nil
	}
	fistTimestamp, err := ins.redisStorage.RPopCommitTime(ctx)
	if err != nil {
		return err
	}
	log.Debugf("count for avg commit duration, currTime: %v, oldest time: %v, list len: %v", currTimestamp, fistTimestamp, listLen)
	timestampDiff := currTimestamp - fistTimestamp
	newAvgDuration := (timestampDiff) / (listLen - 1) / secondsPreMinute
	remainder := timestampDiff / (listLen - 1) % secondsPreMinute
	if remainder > 0 {
		newAvgDuration++
	}
	if !ins.checkAvgDurationLegal(newAvgDuration) {
		log.Errorf("new avg commit is un-legal, so drop it. new duration: %v", newAvgDuration)
		return nil
	}
	err = ins.redisStorage.SetAvgCommitDuration(ctx, newAvgDuration)
	if err != nil {
		return err
	}
	log.Infof("success fresh the avg commit duration: %v", newAvgDuration)
	return nil
}

func (ins *CommittedBatchHandler) pushStatusChangedMsg(ctx context.Context, latestBatchNum uint64, oldMaxBlockNum uint64, maxBlockNum uint64) error {
	// Scan the DB and push events to FE
	if oldMaxBlockNum >= maxBlockNum {
		log.Infof("batch has no block and transaction, so skip push msg, batch: %v", latestBatchNum)
		return nil
	}
	var offset = uint(0)
	l2AvgVerifyDuration := GetAvgVerifyDuration(ctx, ins.redisStorage)
	for {
		deposits, err := ins.storage.GetNotReadyTransactionsWithBlockRange(ctx, l2NetWorkId, oldMaxBlockNum+1,
			maxBlockNum, l1PendingDepositQueryLimit, offset, nil)
		if err != nil {
			log.Errorf("query l2 pending deposits error: %v", err)
			return nil
		}
		log.Debugf("success get deposit for batch: %v, size: %v", latestBatchNum, len(deposits))
		// Notify FE for each transaction
		for _, deposit := range deposits {
			ins.pushMsgForDeposit(deposit, l2AvgVerifyDuration)
		}
		if len(deposits) < l1PendingDepositQueryLimit {
			break
		}
		offset += l1PendingDepositQueryLimit
	}
	return nil
}

func (ins *CommittedBatchHandler) pushMsgForDeposit(deposit *etherman.Deposit, l2AvgVerifyDuration uint64) {
	go func(deposit *etherman.Deposit) {
		if ins.messagePushProducer == nil {
			log.Errorf("kafka push producer is nil, so can't push tx status change msg!")
			return
		}
		if deposit.LeafType != uint8(utils.LeafTypeAsset) {
			log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
			return
		}
		err := ins.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
			FromChain:    uint32(deposit.NetworkID),
			ToChain:      uint32(deposit.DestinationNetwork),
			TxHash:       deposit.TxHash.String(),
			Index:        uint64(deposit.DepositCount),
			Status:       uint32(pb.TransactionStatus_TX_PENDING_VERIFICATION),
			DestAddr:     deposit.DestinationAddress.Hex(),
			EstimateTime: uint32(l2AvgVerifyDuration),
		})
		if err != nil {
			log.Errorf("PushTransactionUpdate for pending-verify error: %v", err)
		}
	}(deposit)
}

// checkAvgDurationLegal duration has a default range, 2-10 minutes, if over range, maybe dirty data, drop the data
func (ins *CommittedBatchHandler) checkAvgDurationLegal(avgDuration int64) bool {
	return avgDuration > int64(minCommitDuration.Get()) && avgDuration < int64(defaultCommitDuration.Get())
}

func GetAvgCommitDuration(ctx context.Context, redisStorage redisstorage.RedisStorage) uint64 {
	avgDuration, err := redisStorage.GetAvgCommitDuration(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("get avg commit duration from redis failed, error: %v", err)
		return defaultCommitDuration.Get()
	}
	if avgDuration == 0 {
		log.Infof("get avg commit duration from redis is 0, so use default")
		return defaultCommitDuration.Get()
	}
	return avgDuration
}

func GetLeftCommitTime(depositCreateTime time.Time, l2AvgCommitDuration uint64, currentTime time.Time) int {
	return int(l2AvgCommitDuration - uint64(currentTime.Sub(depositCreateTime).Minutes()))
}
