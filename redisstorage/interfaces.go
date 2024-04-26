package redisstorage

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server/tokenlogoinfo"
	"github.com/redis/go-redis/v9"
)

type RedisStorage interface {
	// Coin price storage
	SetCoinPrice(ctx context.Context, prices []*pb.SymbolPrice) error
	GetCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error)

	// Block num storage
	SetL1BlockNum(ctx context.Context, blockNum uint64) error
	GetL1BlockNum(ctx context.Context) (uint64, error)
	AddBlockDeposit(ctx context.Context, deposit *etherman.Deposit) error
	DeleteBlockDeposit(ctx context.Context, deposit *etherman.Deposit) error
	GetBlockDepositList(ctx context.Context, networkID uint, blockNum uint64) ([]*etherman.Deposit, error)

	// General lock
	TryLock(ctx context.Context, lockKey string) (success bool, err error)
	ReleaseLock(ctx context.Context, lockKey string) error

	// commit and verify batch storage
	SetCommitBatchNum(ctx context.Context, batchNum uint64) error
	GetCommitBatchNum(ctx context.Context) (uint64, error)
	SetCommitMaxBlockNum(ctx context.Context, batchNum uint64) error
	GetCommitMaxBlockNum(ctx context.Context) (uint64, error)
	SetAvgCommitDuration(ctx context.Context, commitDuration int64) error
	GetAvgCommitDuration(ctx context.Context) (uint64, error)
	SetL2BlockCommitTime(ctx context.Context, blockNum uint64, commitTimestamp int64) error
	GetL2BlockCommitTime(ctx context.Context, blockNum uint64) (uint64, error)
	LPushCommitTime(ctx context.Context, commitTimeTimestamp int64) error
	LLenCommitTimeList(ctx context.Context) (int64, error)
	RPopCommitTime(ctx context.Context) (int64, error)
	SetVerifyBatchNum(ctx context.Context, batchNum uint64) error
	GetVerifyBatchNum(ctx context.Context) (uint64, error)
	SetAvgVerifyDuration(ctx context.Context, commitDuration int64) error
	GetAvgVerifyDuration(ctx context.Context) (uint64, error)
	LPushVerifyTime(ctx context.Context, commitTimeTimestamp int64) error
	LLenVerifyTimeList(ctx context.Context) (int64, error)
	RPopVerifyTime(ctx context.Context) (int64, error)

	// token logo storage
	SetTokenLogoInfo(ctx context.Context, keySuffix string, logoInfo *tokenlogoinfo.TokenLogoInfo) error
	GetTokenLogoInfo(ctx context.Context, keySuffix string) (*tokenlogoinfo.TokenLogoInfo, error)
}

type RedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
	HVals(ctx context.Context, key string) *redis.StringSliceCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}
