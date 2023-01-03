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
	Set(ctx context.Context, key []byte, value [][]byte, dbTx pgx.Tx) error
	ResetMT(ctx context.Context, depositCount uint, network uint8, dbTx pgx.Tx) error
	GetRoot(ctx context.Context, depositCount uint, network uint8, dbTx pgx.Tx) ([]byte, error)
	SetRoot(ctx context.Context, root []byte, depositCount uint, network uint8, dbTx pgx.Tx) error
	GetLastDepositCount(ctx context.Context, network uint8, dbTx pgx.Tx) (uint, error)
}

// bridgeStorage interface for the Bridge Tree
type bridgeStorage interface {
	GetRoot(ctx context.Context, depositCnt uint, network uint8, dbTx pgx.Tx) ([]byte, error)
	GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error)
	GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestTrustedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	AddGlobalExitRoot(ctx context.Context, globalExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error)
}

// BridgeServiceStorage interface for the Bridge Service.
type BridgeServiceStorage interface {
	GetClaims(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Claim, error)
	GetClaim(ctx context.Context, index uint, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error)
	GetClaimCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error)
	GetDeposit(ctx context.Context, depositCnt uint, networkID uint, dbTx pgx.Tx) (*etherman.Deposit, error)
	GetDeposits(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	GetDepositCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error)
}
