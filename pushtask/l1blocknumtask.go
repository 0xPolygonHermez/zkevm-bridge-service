package pushtask

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
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
	l1BlockNumTaskInterval = 5 * time.Second
	l1BlockNumTaskLockKey  = "bridge_l1_block_num_lock"
)

type L1BlockNumTask struct {
	storage             DBStorage
	redisStorage        redisstorage.RedisStorage
	client              *ethclient.Client
	messagePushProducer messagepush.KafkaProducer
}

func NewL1BlockNumTask(rpcURL string, storage interface{}, redisStorage redisstorage.RedisStorage, producer messagepush.KafkaProducer) (*L1BlockNumTask, error) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "ethclient dial error")
	}

	return &L1BlockNumTask{
		storage:             storage.(DBStorage),
		redisStorage:        redisStorage,
		client:              client,
		messagePushProducer: producer,
	}, nil
}

func (t *L1BlockNumTask) Start(ctx context.Context) {
	log.Debugf("Starting L1BlockNumTask, interval:%v", l1BlockNumTaskInterval)
	ticker := time.NewTicker(l1BlockNumTaskInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.doTask(ctx)
		}
	}
}

func (t *L1BlockNumTask) doTask(ctx context.Context) {
	ok, err := t.redisStorage.TryLock(ctx, l1BlockNumTaskLockKey)
	if err != nil {
		log.Errorf("TryLock key[%v] error: %v", l1BlockNumTaskLockKey, err)
		return
	}

	if !ok {
		return
	}

	// If successfully acquired the lock, need to eventually release it
	defer func() {
		err = t.redisStorage.ReleaseLock(ctx, l1BlockNumTaskLockKey)
		if err != nil {
			log.Errorf("ReleaseLock key[%v] error: %v", l1BlockNumTaskLockKey, err)
		}
	}()

	// Get the latest block num from the chain RPC
	blockNum, err := t.client.BlockNumber(ctx)
	if err != nil {
		log.Errorf("eth_blockNumber error: %v", err)
		return
	}

	// Get the previous block num from Redis cache and check
	oldBlockNum, err := t.redisStorage.GetL1BlockNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("Get L1 block num from Redis error: %v", err)
		return
	}

	// If the block num is not changed, no need to do anything
	if blockNum <= oldBlockNum {
		return
	}

	defer func(blockNum uint64) {
		// Update Redis cached block num
		err = t.redisStorage.SetL1BlockNum(ctx, blockNum)
		if err != nil {
			log.Errorf("SetL1BlockNum error: %v", err)
		}
	}(blockNum)

	// Minus 64 to get the target query block num
	oldBlockNum -= utils.Min(utils.L1TargetBlockConfirmations.Get(), oldBlockNum)
	blockNum -= utils.Min(utils.L1TargetBlockConfirmations.Get(), blockNum)
	if blockNum <= oldBlockNum {
		return
	}

	var (
		totalDeposits = 0
	)

	for block := oldBlockNum + 1; block <= blockNum; block++ {
		// For each block num, get the list of deposit and push the events to FE
		deposits, err := t.redisStorage.GetBlockDepositList(ctx, 0, block)
		if err != nil {
			log.Errorf("L1BlockNumTask query Redis error: %v", err)
			return
		}
		totalDeposits += len(deposits)

		// Notify FE for each transaction
		for _, deposit := range deposits {
			go func(deposit *etherman.Deposit) {
				if t.messagePushProducer == nil {
					return
				}
				if deposit.LeafType != uint8(utils.LeafTypeAsset) {
					log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
					return
				}
				err := t.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
					FromChain: uint32(deposit.NetworkID),
					ToChain:   uint32(deposit.DestinationNetwork),
					TxHash:    deposit.TxHash.String(),
					Index:     uint64(deposit.DepositCount),
					Status:    uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM),
					DestAddr:  deposit.DestinationAddress.Hex(),
				})
				if err != nil {
					log.Errorf("PushTransactionUpdate error: %v", err)
				}
			}(deposit)
		}
	}

	log.Infof("L1BlockNumTask push for %v deposits, block num from %v to %v", totalDeposits, oldBlockNum, blockNum)
}
