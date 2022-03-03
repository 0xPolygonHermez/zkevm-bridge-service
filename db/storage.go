package db

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
)

// Storage interface
type Storage interface {
	GetLastBlock(ctx context.Context) (*etherman.Block, error)
	GetLastL2Block(ctx context.Context) (*etherman.Block, error)
	AddBlock(ctx context.Context, block *etherman.Block) error
	AddL2Block(ctx context.Context, block *etherman.Block) error
	GetPreviousBlock(ctx context.Context, offset uint64) (*etherman.Block, error)
	GetPreviousL2Block(ctx context.Context, offset uint64) (*etherman.Block, error)
	Reset(ctx context.Context, blockNumber uint64) error
	ResetL2(ctx context.Context, blockNumber uint64) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
	GetDeposit(ctx context.Context, depositCounterUser uint, destNetwork uint) (*etherman.Deposit, error)
	AddL2Deposit(ctx context.Context, deposit *etherman.Deposit) error
	GetL2Deposit(ctx context.Context, depositCounterUser uint64, destNetwork uint) (*etherman.Deposit, error)
	Rollback(ctx context.Context) error
	BeginDBTransaction(ctx context.Context) error
	AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error
	GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	AddClaim(ctx context.Context, claim *etherman.Claim) error
	AddL2Claim(ctx context.Context, claim *etherman.Claim) error
	AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error)
	AddL2TokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error
	GetL2TokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error)
	ConsolidateBatch(ctx context.Context, batchNumber uint64, consolidatedTxHash common.Hash, consolidatedAt time.Time, aggregator common.Address) error
	AddBatch(ctx context.Context, batch *etherman.Batch) error
	GetClaim(ctx context.Context, depositCounterUser uint, originalNetwork uint) (*etherman.Claim, error)
	GetL2Claim(ctx context.Context, depositCounterUser uint, originalNetwork uint) (*etherman.Claim, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64) (*etherman.Batch, error)
	GetNumberL1Deposits(ctx context.Context) (uint64, error)
	GetNumberL2Deposits(ctx context.Context, networkID uint) (uint64, error)
}

// NewStorage creates a new Storage
func NewStorage(cfg Config) (Storage, error) {
	if cfg.Database == "postgres" {
		return pgstorage.NewPostgresStorage(pgstorage.Config{
			Name:     cfg.Name,
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     cfg.Port,
		})
	}
	return nil, gerror.ErrStorageNotRegister
}

// RunMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes
func RunMigrations(cfg Config) error {
	config := pgstorage.Config{
		Name:     cfg.Name,
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
	}
	return pgstorage.RunMigrations(config)
}
