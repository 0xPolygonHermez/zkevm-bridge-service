package redisstorage

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/redis/go-redis/v9"
)

type RedisStorage interface {
	SetCoinPrice(ctx context.Context, prices []*pb.SymbolPrice) error
	GetCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error)
}

type RedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
}
