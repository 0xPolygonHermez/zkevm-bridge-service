package claimtxman

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/jackc/pgx/v4"
)

type storageInterface interface {
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	UpdateL1DepositsStatus(ctx context.Context, exitRoot []byte) ([]*pb.Deposit, error)
	UpdateL2DepositsStatus(ctx context.Context, exitRoot []byte) error
	AddClaimTx(ctx context.Context, mTx MonitoredTx, dbTx pgx.Tx) error
	UpdateClaimTx(ctx context.Context, mTx MonitoredTx, dbTx pgx.Tx) error
	GetByStatus(ctx context.Context, owner *string, statuses []MonitoredTxStatus, dbTx pgx.Tx) ([]MonitoredTx, error)
	// atomic
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error
}
