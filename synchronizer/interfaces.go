package synchronizer

import (
	"context"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
)

// ethermanInterface contains the methods required to interact with ethereum.
type ethermanInterface interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	GetRollupInfoByBlockRange(ctx context.Context, fromBlock uint64, toBlock *uint64) ([]etherman.Block, map[common.Hash][]etherman.Order, error)
	EthBlockByNumber(ctx context.Context, blockNumber uint64) (*types.Block, error)
	GetLatestBatchNumber() (uint64, error)
	GetNetworkID(ctx context.Context) (uint, error)
}

type storageInterface interface {
	GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error)
	GetLastBatchNumber(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	GetBatchByNumber(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) (*etherman.Batch, error)
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	AddBatch(ctx context.Context, batch *etherman.Batch, dbTx pgx.Tx) error
	AddVerifiedBatch(ctx context.Context, verifiedBatch *etherman.VerifiedBatch, dbTx pgx.Tx) error
	AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) error
	AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error
	AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error
	Reset(ctx context.Context, blockNumber uint64, networkID uint, dbTx pgx.Tx) error
	ResetTrustedState(ctx context.Context, batchNumber uint64, dbTx pgx.Tx) error
	GetPreviousBlock(ctx context.Context, networkID uint, offset uint64, dbTx pgx.Tx) (*etherman.Block, error)
	GetNumberDeposits(ctx context.Context, origNetworkID uint, blockNumber uint64, dbTx pgx.Tx) (uint64, error)
	// GetNextForcedBatches returns the next forcedBatches in FIFO order
	GetNextForcedBatches(ctx context.Context, nextForcedBatches int, dbTx pgx.Tx) ([]etherman.ForcedBatch, error)
	AddBatchNumberInForcedBatch(ctx context.Context, forceBatchNumber, batchNumber uint64, dbTx pgx.Tx) error
	AddForcedBatch(ctx context.Context, forcedBatch *etherman.ForcedBatch, dbTx pgx.Tx) error
	AddTrustedGlobalExitRoot(ctx context.Context, trustedExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	GetLastVerifiedBatch(ctx context.Context, dbTx pgx.Tx) (*etherman.VerifiedBatch, error)
}

type bridgectrlInterface interface {
	AddDeposit(deposit *etherman.Deposit) error
	ReorgMT(depositCount, networkID uint) error
}
