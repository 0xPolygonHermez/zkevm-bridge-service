package bridgectrl

import (
	"context"

	"github.com/jackc/pgx/v4"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error)
	BulkSet(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error
	GetRoot(ctx context.Context, depositCount uint, network uint, dbTx pgx.Tx) ([]byte, error)
	SetRoot(ctx context.Context, root []byte, depositID uint64, depositCount uint, network uint, dbTx pgx.Tx) error
	GetLastDepositCount(ctx context.Context, network uint, dbTx pgx.Tx) (uint, error)
}
