package redisstorage

import (
	"context"
	"strconv"
	"strings"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	coinPriceHashKey = "bridge_coin_prices"
)

// redisStorageImpl implements RedisStorage interface
type redisStorageImpl struct {
	client    RedisClient
	mockPrice bool
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
	return &redisStorageImpl{client: client, mockPrice: cfg.MockPrice}, nil
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
	err = s.client.HSet(ctx, coinPriceHashKey, valueList...).Err()
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

	redisResult, err := s.client.HMGet(ctx, coinPriceHashKey, keyList...).Result()
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
	priceList, err := s.getCoinPrice(ctx, symbols)
	if err != nil {
		return nil, err
	}

	// If enable mock price, always return no price to the front end
	if s.mockPrice {
		for _, price := range priceList {
			price.Price = 0
			price.Time = 0
		}
	}
	return priceList, nil
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
