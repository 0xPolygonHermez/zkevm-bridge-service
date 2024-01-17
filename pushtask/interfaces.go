package pushtask

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/jackc/pgx/v4"
)

type DBStorage interface {
	GetNotReadyTransactionsWithBlockRange(ctx context.Context, networkID uint, minBlockNum, maxBlockNum uint64, limit, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error)
}
