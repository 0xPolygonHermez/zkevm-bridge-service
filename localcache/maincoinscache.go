package localcache

import (
	"context"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

const (
	cacheRefreshInterval = 5 * time.Minute
	queryLimit           = 100
	maxRetries           = 5
)

type MainCoinsCache interface {
	GetMainCoinsByNetwork(ctx context.Context, networkID uint32) ([]*pb.CoinInfo, error)
}

type MainCoinsDBStorage interface {
	GetAllMainCoins(ctx context.Context, limit uint, offset uint, dbTx pgx.Tx) ([]*pb.CoinInfo, error)
}

// mainCoinsCacheImpl implements the MainCoinsCache interface
type mainCoinsCacheImpl struct {
	lock    sync.RWMutex
	data    map[uint32][]*pb.CoinInfo // networkID -> list of coins
	storage MainCoinsDBStorage
}

func NewMainCoinsCache(storage interface{}) (MainCoinsCache, error) {
	if storage == nil {
		return nil, errors.New("NewMainCoinsCache storage is nil")
	}
	cache := &mainCoinsCacheImpl{
		data:    make(map[uint32][]*pb.CoinInfo),
		storage: storage.(MainCoinsDBStorage),
	}
	err := cache.doRefresh(context.Background())
	if err != nil {
		log.Errorf("init main coins cache err[%v]", err)
		return nil, err
	}
	go cache.Refresh(context.Background())
	return cache, nil
}

// Refresh loops indefinitely and refresh the cache data every 5 minutes
func (c *mainCoinsCacheImpl) Refresh(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(cacheRefreshInterval)
	for range ticker.C {
		log.Info("start refreshing main coins cache")
		err := c.doRefresh(ctx)
		if err != nil {
			log.Errorf("refresh main coins cache error[%v]", err)
		}
		log.Infof("finish refreshing main coins cache")
	}
}

// doRefresh reads all the main coins data from DB and populate the local cache
// If fail to read the DB, will retry for up to 5 times
func (c *mainCoinsCacheImpl) doRefresh(ctx context.Context) error {
	newData := make(map[uint32][]*pb.CoinInfo)

	offset := uint(0)
	retryCnt := 0
	for {
		coins, err := c.storage.GetAllMainCoins(ctx, queryLimit, offset, nil)
		if err != nil {
			// If exceeds max number of retries, returns without updating the cache
			if retryCnt >= maxRetries {
				return err
			}
			retryCnt++
			continue
		}
		retryCnt = 0
		for _, coin := range coins {
			newData[coin.NetworkId] = append(newData[coin.NetworkId], coin)
		}

		// Reaching the last batch of data
		if len(coins) < queryLimit {
			break
		}
	}

	// Need to lock the map before updating
	c.lock.Lock()
	c.data = newData
	c.lock.Unlock()
	return nil
}

func (c *mainCoinsCacheImpl) GetMainCoinsByNetwork(ctx context.Context, networkID uint32) ([]*pb.CoinInfo, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.data[networkID], nil
}
