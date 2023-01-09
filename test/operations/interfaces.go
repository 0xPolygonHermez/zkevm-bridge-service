package operations

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type storageInterface interface {
	GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error)
	GetLatestExitRoot(ctx context.Context, isRollup bool, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestTrustedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error)
	GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error)
	UpdateBlocksForTesting(ctx context.Context, networkID uint, blockNum uint64, dbTx pgx.Tx) error
	GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	UpdateBatchesForTesting(ctx context.Context, batchNum uint64, dbTx pgx.Tx) error
	// synchronizer
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	AddBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error
	AddVerifiedBatch(ctx context.Context, verifiedBatch *etherman.VerifiedBatch, dbTx pgx.Tx) error
	AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) error
	AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error
	AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error
}