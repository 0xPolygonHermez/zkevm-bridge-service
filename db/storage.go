package db

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

// Storage interface
type Storage interface {
	GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error)
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error)
	Reset(ctx context.Context, blockNumber uint64, dbTx pgx.Tx) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) error
	GetDeposit(ctx context.Context, depositCounterUser uint, networkID uint, dbTx pgx.Tx) (*etherman.Deposit, error)
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error
	AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	GetLatestExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestL2SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error
	AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error)
	ConsolidateBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error
	AddBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error
	GetClaim(ctx context.Context, depositCounterUser uint, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*etherman.Batch, error)
	GetNumberDeposits(ctx context.Context, networkID uint, blockNumber uint64, dbTx pgx.Tx) (uint64, error)
	GetLastBatchState(ctx context.Context, dbTx pgx.Tx) (uint64, uint64, bool, error)
	AddBatchNumberInForcedBatch(ctx context.Context, forceBatchNumber, batchNumber uint64, dbTx pgx.Tx) error
	AddForcedBatch(ctx context.Context, forcedBatch *etherman.ForcedBatch, dbTx pgx.Tx) error
	AddTrustedGlobalExitRoot(ctx context.Context, ger common.Hash) error
	AddVerifiedBatch(ctx context.Context, verifiedBatch *etherman.VerifiedBatch, dbTx pgx.Tx) error
	GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	GetNextForcedBatches(ctx context.Context, nextForcedBatches int, dbTx pgx.Tx) ([]etherman.ForcedBatch, error)
	ResetTrustedState(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) error
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
	return pgstorage.RunMigrationsUp(config)
}
