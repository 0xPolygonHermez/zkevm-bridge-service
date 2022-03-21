package db

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
)

// Storage interface
type Storage interface {
	GetLastBlock(ctx context.Context, networkID uint) (*etherman.Block, error)
	AddBlock(ctx context.Context, block *etherman.Block) (uint64, error)
	GetPreviousBlock(ctx context.Context, networkID uint, offset uint64) (*etherman.Block, error)
	Reset(ctx context.Context, block *etherman.Block, networkID uint) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
	GetDeposit(ctx context.Context, depositCounterUser uint, origNetwork uint) (*etherman.Deposit, error)
	Rollback(ctx context.Context) error
	BeginDBTransaction(ctx context.Context) error
	Commit(ctx context.Context) error
	AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error
	GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	AddClaim(ctx context.Context, claim *etherman.Claim) error
	AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error)
	ConsolidateBatch(ctx context.Context, batch *etherman.Batch) error
	AddBatch(ctx context.Context, batch *etherman.Batch) error
	GetClaim(ctx context.Context, depositCounterUser uint, originalNetwork uint) (*etherman.Claim, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, networkID uint) (*etherman.Batch, error)
	GetNumberDeposits(ctx context.Context, origNetworkID uint) (uint64, error)
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
