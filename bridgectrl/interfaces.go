package bridgectrl

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error)
	BulkSet(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error
	GetRoot(ctx context.Context, depositCount uint, network uint, dbTx pgx.Tx) ([]byte, error)
	SetRoot(ctx context.Context, root []byte, depositID uint64, network uint, dbTx pgx.Tx) error
	GetLastDepositCount(ctx context.Context, network uint, dbTx pgx.Tx) (uint, error)
	AddRollupExitLeaves(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error
	GetRollupExitLeavesByRoot(ctx context.Context, root common.Hash, dbTx pgx.Tx) ([]etherman.RollupExitLeaf, error)
	GetLatestRollupExitLeaves(ctx context.Context, dbTx pgx.Tx) ([]etherman.RollupExitLeaf, error)
}
