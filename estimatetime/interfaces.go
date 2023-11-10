package estimatetime

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/jackc/pgx/v4"
)

// Calculator provides methods to calculate the deposit estimate time by sampling recent deposits
type Calculator interface {
	Get(networkID uint) uint32
}

type DBStorage interface {
	GetLatestReadyDeposits(ctx context.Context, networkID uint, limit uint, dbTx pgx.Tx) ([]*etherman.Deposit, error)
}
