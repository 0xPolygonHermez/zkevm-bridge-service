package bridgectrl

import (
	"context"

	"github.com/jackc/pgx/v4"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error)
	BulkSet(ctx context.Context, rows [][]interface{}, dbTx pgx.Tx) error
	ResetMT(ctx context.Context, depositCount uint, network uint8, dbTx pgx.Tx) error
	GetRoot(ctx context.Context, depositCount uint, network uint8, dbTx pgx.Tx) ([]byte, error)
	SetRoot(ctx context.Context, root []byte, depositCount uint, network uint8, dbTx pgx.Tx) (uint64, error)
	GetLastDepositCount(ctx context.Context, network uint8, dbTx pgx.Tx) (uint, error)
}
