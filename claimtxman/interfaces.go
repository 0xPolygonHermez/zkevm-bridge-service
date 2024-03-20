package claimtxman

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/jackc/pgx/v4"
)

type storageInterface interface {
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	UpdateL1DepositsStatus(ctx context.Context, exitRoot []byte, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	UpdateL2DepositsStatus(ctx context.Context, exitRoot []byte, rollupID, networkID uint, dbTx pgx.Tx) error
	AddClaimTx(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx) error
	UpdateClaimTx(ctx context.Context, mTx types.MonitoredTx, dbTx pgx.Tx) error
	GetClaimTxsByStatus(ctx context.Context, statuses []types.MonitoredTxStatus, dbTx pgx.Tx) ([]types.MonitoredTx, error)
	// atomic
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error

	// XLayer
	UpdateL1DepositsStatusXLayer(ctx context.Context, exitRoot []byte, eventTime time.Time, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	UpdateL2DepositsStatusXLayer(ctx context.Context, exitRoot []byte, eventTime time.Time, rollupID, networkID uint, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	GetL1Deposits(ctx context.Context, exitRoot []byte, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	UpdateL1DepositStatus(ctx context.Context, depositCount uint, eventTime time.Time, dbTx pgx.Tx) error
	GetDeposit(ctx context.Context, depositCounterUser uint, networkID uint, dbTx pgx.Tx) (*etherman.Deposit, error)
	GetClaim(ctx context.Context, depositCount, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error)
}

type bridgeServiceInterface interface {
	GetClaimProof(depositCnt, networkID uint, dbTx pgx.Tx) (*etherman.GlobalExitRoot, [][bridgectrl.KeyLen]byte, [][bridgectrl.KeyLen]byte, error)
	GetDepositStatus(ctx context.Context, depositCount uint, destNetworkID uint) (string, error)
}
