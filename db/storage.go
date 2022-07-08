package db

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
)

// Storage interface
type Storage interface {
	GetLastBlock(ctx context.Context, networkID uint) (*etherman.Block, error)
	AddBlock(ctx context.Context, block *etherman.Block) (uint64, error)
	GetPreviousBlock(ctx context.Context, networkID uint, offset uint64) (*etherman.Block, error)
	Reset(ctx context.Context, block *etherman.Block, networkID uint) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
	GetDeposit(ctx context.Context, depositCounterUser uint, networkID uint) (*etherman.Deposit, error)
	Rollback(ctx context.Context, index uint) error
	BeginDBTransaction(ctx context.Context, index uint) error
	Commit(ctx context.Context, index uint) error
	AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error
	GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	GetLatestL1SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	GetLatestL2SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	AddClaim(ctx context.Context, claim *etherman.Claim) error
	AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error)
	ConsolidateBatch(ctx context.Context, batch *etherman.Batch) error
	AddBatch(ctx context.Context, batch *etherman.Batch) error
	GetClaim(ctx context.Context, depositCounterUser uint, networkID uint) (*etherman.Claim, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, networkID uint) (*etherman.Batch, error)
	GetNumberDeposits(ctx context.Context, networkID uint, blockNumber uint64) (uint64, error)
	GetLastBatchState(ctx context.Context) (uint64, uint64, bool, error)
}

// NewStorage creates a new Storage
func NewStorage(cfg Config, networksNumber uint) (Storage, error) {
	if cfg.Database == "postgres" {
		return pgstorage.NewPostgresStorage(pgstorage.Config{
			Name:     cfg.Name,
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     cfg.Port,
			MaxConns: cfg.MaxConns,
		}, networksNumber)
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
		Port:     cfg.Port,
	}
	return pgstorage.RunMigrations(config)
}
