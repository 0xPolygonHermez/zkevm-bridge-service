package redisstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/models/tokenlogo"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	enableCoinPriceCfgKey    = "coinPrice.enabled"
	coinPriceHashKey         = "bridge_coin_prices"
	l1BlockNumKey            = "bridge_l1_block_num"
	l1BlockDepositListKey    = "bridge_block_deposits_%d_%d" // Params: networkID and block num
	l1BlockDepositListExpiry = 24 * time.Hour

	//batch info key
	latestCommitBatchNumKey = "bridge_latest_commit_batch_num"
	maxCommitBlockNumKey    = "bridge_max_commit_block_num"
	avgCommitDurationKey    = "bridge_avg_commit_duration"
	commitBatchTimeListKey  = "bridge_commit_batch_simple_info"
	latestVerifyBatchNumKey = "bridge_latest_verify_batch_num"
	avgVerifyDurationKey    = "bridge_avg_verify_duration"
	verifyBatchTimeListKey  = "bridge_verify_batch_simple_info"
	l2BlockCommitTimeKey    = "bridge_l2_block_commit_time_key_"

	// token logo info key
	tokenLogoInfoKey = "bridge_token_logo_info_"

	// Set a default expiration for locks to prevent a process from keeping the lock for too long
	lockExpire = 1 * time.Minute

	// time for 48 hour
	durationFor48h = 48 * time.Hour
)

// redisStorageImpl implements RedisStorage interface
type redisStorageImpl struct {
	client             RedisClient
	enableCoinPriceCfg apolloconfig.Entry[bool]
	keyPrefix          apolloconfig.Entry[string]
}

func NewRedisStorage(cfg Config) (RedisStorage, error) {
	if len(cfg.Addrs) == 0 {
		return nil, errors.New("redis address is empty")
	}
	var client RedisClient
	if cfg.IsClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.Addrs,
			Username: cfg.Username,
			Password: cfg.Password,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Addrs[0],
			Username: cfg.Username,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}
	res, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to redis server")
	}
	log.Debugf("redis health check done, result: %v", res)
	return &redisStorageImpl{client: client,
		enableCoinPriceCfg: apolloconfig.NewBoolEntry("CoinPrice.Enabled", cfg.EnablePrice),
		keyPrefix:          apolloconfig.NewStringEntry("Redis.KeyPrefix", cfg.KeyPrefix),
	}, nil
}

func (s *redisStorageImpl) addKeyPrefix(key string) string {
	return s.keyPrefix.Get() + key
}

func (s *redisStorageImpl) SetCoinPrice(ctx context.Context, prices []*pb.SymbolPrice) error {
	log.Debugf("SetCoinPrice size[%v]", len(prices))
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}

	var valueList []interface{}
	symbols := convertPricesToSymbols(prices)
	currentPrices, err := s.getCoinPrice(ctx, symbols)
	if err != nil {
		return errors.Wrap(err, "SetCoinPrice get current price error")
	}
	// Assuming size of currentPrices and prices is the same
	for i, price := range prices {
		if price.Time < currentPrices[i].Time {
			// Old price, ignored
			continue
		}

		priceKey := getCoinPriceKey(price.ChainId, price.Address)
		priceVal, err := protojson.Marshal(price)
		if err != nil {
			return errors.Wrap(err, "marshal price error")
		}
		valueList = append(valueList, priceKey, priceVal)
	}
	if len(valueList) < 2 { // nolint:gomnd
		return nil
	}
	err = s.client.HSet(ctx, s.addKeyPrefix(coinPriceHashKey), valueList...).Err()
	if err != nil {
		return errors.Wrap(err, "SetCoinPrice redis HSet error")
	}

	return nil
}

func (s *redisStorageImpl) getCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error) {
	if len(symbols) == 0 {
		return nil, nil
	}
	if s == nil || s.client == nil {
		return nil, errors.New("redis client is nil")
	}

	var keyList []string
	for _, symbol := range symbols {
		if symbol == nil {
			// This means there can be a chance that request size and response size are different
			// But by right the symbols array should not have nil values
			continue
		}
		priceKey := getCoinPriceKey(symbol.ChainId, symbol.Address)
		keyList = append(keyList, priceKey)
	}

	redisResult, err := s.client.HMGet(ctx, s.addKeyPrefix(coinPriceHashKey), keyList...).Result()
	if err != nil {
		return nil, errors.Wrap(err, "getCoinPrice redis HMGet error")
	}

	var priceList []*pb.SymbolPrice
	for i, res := range redisResult {
		if res == nil {
			log.Infof("getCoinPrice price not found chainId[%v] address[%v]", symbols[i].ChainId, symbols[i].Address)
			priceList = append(priceList, &pb.SymbolPrice{ChainId: symbols[i].ChainId, Address: symbols[i].Address})
			continue
		}
		price := &pb.SymbolPrice{}
		err := protojson.Unmarshal([]byte(res.(string)), price)
		if err != nil {
			log.Infof("cannot unmarshal price object[%v] error[%v]", res, err)
			priceList = append(priceList, &pb.SymbolPrice{ChainId: symbols[i].ChainId, Address: symbols[i].Address})
		} else {
			priceList = append(priceList, price)
		}
	}

	return priceList, nil
}

func (s *redisStorageImpl) GetCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error) {
	log.Debugf("GetCoinPrice size[%v]", len(symbols))
	var priceList []*pb.SymbolPrice
	var err error
	if s.enableCoinPriceCfg.Get() {
		priceList, err = s.getCoinPrice(ctx, symbols)
		if err != nil {
			return nil, err
		}
	} else {
		// If disable coin price, just return a list of empty prices
		for _, si := range symbols {
			priceList = append(priceList, &pb.SymbolPrice{ChainId: si.ChainId, Address: si.Address})
		}
	}

	return priceList, nil
}

func (s *redisStorageImpl) SetL1BlockNum(ctx context.Context, blockNum uint64) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	err := s.client.Set(ctx, s.addKeyPrefix(l1BlockNumKey), blockNum, 0).Err()
	if err != nil {
		return errors.Wrap(err, "SetL1BlockNum error")
	}
	return nil
}

func (s *redisStorageImpl) GetL1BlockNum(ctx context.Context) (uint64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("redis client is nil")
	}
	res, err := s.client.Get(ctx, s.addKeyPrefix(l1BlockNumKey)).Result()
	if err != nil {
		return 0, errors.Wrap(err, "GetL1BlockNum error")
	}
	num, err := strconv.ParseInt(res, 10, 64) //nolint:gomnd
	return uint64(num), errors.Wrap(err, "GetL1BlockNum convert error")
}

func (s *redisStorageImpl) AddBlockDeposit(ctx context.Context, deposit *etherman.Deposit) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	if deposit == nil {
		return nil
	}
	// Encode to json
	val, err := json.Marshal(deposit)
	if err != nil {
		return errors.Wrap(err, "json encode error")
	}
	key := s.addKeyPrefix(getBlockDepositListKey(deposit.NetworkID, deposit.BlockNumber))
	err = s.client.HSet(ctx, key, deposit.TxHash.String(), string(val)).Err()
	if err != nil {
		return errors.Wrap(err, "AddBlockDeposit HSet error")
	}

	// If add successfully, expire the list in 24 hours, no need to check for result
	s.client.Expire(ctx, key, l1BlockDepositListExpiry)
	return nil
}

func (s *redisStorageImpl) DeleteBlockDeposit(ctx context.Context, deposit *etherman.Deposit) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	if deposit == nil {
		return nil
	}
	key := s.addKeyPrefix(getBlockDepositListKey(deposit.NetworkID, deposit.BlockNumber))
	err := s.client.HDel(ctx, key, deposit.TxHash.String()).Err()
	return errors.Wrap(err, "DeleteBlockDeposit HDel error")
}

func (s *redisStorageImpl) GetBlockDepositList(ctx context.Context, networkID uint, blockNum uint64) ([]*etherman.Deposit, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("redis client is nil")
	}

	key := s.addKeyPrefix(getBlockDepositListKey(networkID, blockNum))
	res, err := s.client.HVals(ctx, key).Result()

	if err != nil {
		return nil, errors.Wrap(err, "GetBlockDeposits HVals error")
	}

	// Decode the deposit list
	depositList := make([]*etherman.Deposit, len(res))
	for i, s := range res {
		deposit := &etherman.Deposit{}
		err = json.Unmarshal([]byte(s), deposit)
		if err != nil {
			return nil, errors.Wrap(err, "GetBlockDeposits json decode error")
		}
		depositList[i] = deposit
	}

	return depositList, nil
}

func (s *redisStorageImpl) TryLock(ctx context.Context, lockKey string) (bool, error) {
	if s == nil || s.client == nil {
		return false, errors.New("redis client is nil")
	}
	success, err := s.client.SetNX(ctx, s.addKeyPrefix(lockKey), true, lockExpire).Result()
	return success, errors.Wrap(err, "TryLock error")
}

func (s *redisStorageImpl) ReleaseLock(ctx context.Context, lockKey string) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	err := s.client.Del(ctx, s.addKeyPrefix(lockKey)).Err()
	return errors.Wrap(err, "ReleaseLock error")
}

func (s *redisStorageImpl) SetCommitBatchNum(ctx context.Context, batchNum uint64) error {
	return s.setFoundation(ctx, s.addKeyPrefix(latestCommitBatchNumKey), batchNum, 0)
}

func (s *redisStorageImpl) GetCommitBatchNum(ctx context.Context) (uint64, error) {
	return s.getIntCacheFoundation(ctx, s.addKeyPrefix(latestCommitBatchNumKey))
}

func (s *redisStorageImpl) SetCommitMaxBlockNum(ctx context.Context, blockNum uint64) error {
	return s.setFoundation(ctx, s.addKeyPrefix(maxCommitBlockNumKey), blockNum, 0)
}

func (s *redisStorageImpl) GetCommitMaxBlockNum(ctx context.Context) (uint64, error) {
	return s.getIntCacheFoundation(ctx, s.addKeyPrefix(maxCommitBlockNumKey))
}

func (s *redisStorageImpl) SetAvgCommitDuration(ctx context.Context, duration int64) error {
	return s.setFoundation(ctx, s.addKeyPrefix(avgCommitDurationKey), duration, 0)
}

func (s *redisStorageImpl) SetL2BlockCommitTime(ctx context.Context, blockNum uint64, commitTimestamp int64) error {
	key := s.addKeyPrefix(s.buildL2BlockCommitTimeCacheKey(blockNum))
	return s.setFoundation(ctx, key, commitTimestamp, durationFor48h)
}
func (s *redisStorageImpl) GetL2BlockCommitTime(ctx context.Context, blockNum uint64) (uint64, error) {
	key := s.addKeyPrefix(s.buildL2BlockCommitTimeCacheKey(blockNum))
	return s.getIntCacheFoundation(ctx, key)
}

func (s *redisStorageImpl) buildL2BlockCommitTimeCacheKey(blockNum uint64) string {
	return l2BlockCommitTimeKey + strconv.FormatUint(blockNum, 10)
}

func (s *redisStorageImpl) GetAvgCommitDuration(ctx context.Context) (uint64, error) {
	return s.getIntCacheFoundation(ctx, s.addKeyPrefix(avgCommitDurationKey))
}

func (s *redisStorageImpl) LPushCommitTime(ctx context.Context, commitTimeTimestamp int64) error {
	return s.lPushFoundation(ctx, s.addKeyPrefix(commitBatchTimeListKey), commitTimeTimestamp)
}

func (s *redisStorageImpl) LLenCommitTimeList(ctx context.Context) (int64, error) {
	return s.lLenFoundation(ctx, s.addKeyPrefix(commitBatchTimeListKey))
}

func (s *redisStorageImpl) RPopCommitTime(ctx context.Context) (int64, error) {
	return s.rPopIntCacheFoundation(ctx, s.addKeyPrefix(commitBatchTimeListKey))
}

func (s *redisStorageImpl) SetVerifyBatchNum(ctx context.Context, batchNum uint64) error {
	return s.setFoundation(ctx, s.addKeyPrefix(latestVerifyBatchNumKey), batchNum, 0)
}

func (s *redisStorageImpl) GetVerifyBatchNum(ctx context.Context) (uint64, error) {
	return s.getIntCacheFoundation(ctx, s.addKeyPrefix(latestVerifyBatchNumKey))
}

func (s *redisStorageImpl) SetAvgVerifyDuration(ctx context.Context, duration int64) error {
	return s.setFoundation(ctx, s.addKeyPrefix(avgVerifyDurationKey), duration, 0)
}

func (s *redisStorageImpl) GetAvgVerifyDuration(ctx context.Context) (uint64, error) {
	return s.getIntCacheFoundation(ctx, s.addKeyPrefix(avgVerifyDurationKey))
}

func (s *redisStorageImpl) LPushVerifyTime(ctx context.Context, commitTimeTimestamp int64) error {
	return s.lPushFoundation(ctx, s.addKeyPrefix(verifyBatchTimeListKey), commitTimeTimestamp)
}

func (s *redisStorageImpl) LLenVerifyTimeList(ctx context.Context) (int64, error) {
	return s.lLenFoundation(ctx, s.addKeyPrefix(verifyBatchTimeListKey))
}

func (s *redisStorageImpl) RPopVerifyTime(ctx context.Context) (int64, error) {
	return s.rPopIntCacheFoundation(ctx, s.addKeyPrefix(verifyBatchTimeListKey))
}

func (s *redisStorageImpl) setFoundation(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	err := s.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("set for key: %v error", key))
	}
	return nil
}

// todo: optimize adapt to all type
func (s *redisStorageImpl) getIntCacheFoundation(ctx context.Context, key string) (uint64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("redis client is nil")
	}
	res, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("get redis cache for key: %v failed", key))
	}
	num, err := strconv.ParseInt(res, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("convert value for key: %v failed, value: %v", key, res))
	}
	return uint64(num), nil
}

func (s *redisStorageImpl) lPushFoundation(ctx context.Context, key string, values ...interface{}) error {
	if s == nil || s.client == nil {
		return errors.New("redis client is nil")
	}
	err := s.client.LPush(ctx, key, values).Err()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("lPush redis cache for key: %v failed", key))
	}
	return nil
}

func (s *redisStorageImpl) lLenFoundation(ctx context.Context, key string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("redis client is nil")
	}
	res, err := s.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("get redis list len for key: %v failed", key))
	}
	return res, nil
}

// todo: optimize adapt to all type
func (s *redisStorageImpl) rPopIntCacheFoundation(ctx context.Context, key string) (int64, error) {
	if s == nil || s.client == nil {
		return 0, errors.New("redis client is nil")
	}
	res, err := s.client.RPop(ctx, key).Result()
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("r-pop redis list item for key: %v, failed", key))
	}
	num, err := strconv.ParseInt(res, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("convert redis list item value for key: %v failed, value: %v", key, res))
	}
	return num, nil
}

func (s *redisStorageImpl) SetTokenLogoInfo(ctx context.Context, keySuffix string, logoInfo tokenlogo.LogoInfo) error {
	value, err := json.Marshal(logoInfo)
	if err != nil {
		return errors.Wrap(err, "failed to convert logoInfo to string")
	}
	return s.setFoundation(ctx, s.addKeyPrefix(s.getTokenInfoCacheKey(keySuffix)), value, durationFor48h)
}

func (s *redisStorageImpl) GetTokenLogoInfo(ctx context.Context, keySuffix string) (*tokenlogo.LogoInfo, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("redis client is nil")
	}
	fullKey := s.addKeyPrefix(s.getTokenInfoCacheKey(keySuffix))
	res, err := s.client.Get(ctx, fullKey).Result()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("get redis cache for key: %v failed", fullKey))
	}
	logoStruct := &tokenlogo.LogoInfo{}
	err = json.Unmarshal([]byte(res), logoStruct)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed convert logo info cache to struct, cache: %v", res))
	}
	return logoStruct, nil
}

func (s *redisStorageImpl) getTokenInfoCacheKey(keySuffix string) string {
	return tokenLogoInfoKey + keySuffix
}

func getCoinPriceKey(chainID uint64, tokenAddr string) string {
	if tokenAddr == "" {
		tokenAddr = "null"
	}
	return strings.ToLower(strconv.FormatUint(chainID, 10) + "_" + tokenAddr)
}

func convertPricesToSymbols(prices []*pb.SymbolPrice) []*pb.SymbolInfo {
	var result = make([]*pb.SymbolInfo, len(prices))
	for i, price := range prices {
		result[i] = &pb.SymbolInfo{
			ChainId: price.ChainId,
			Address: price.Address,
		}
	}
	return result
}

func getBlockDepositListKey(networkID uint, blockNum uint64) string {
	return fmt.Sprintf(l1BlockDepositListKey, networkID, blockNum)
}
