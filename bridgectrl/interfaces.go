package bridgectrl

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte) ([][]byte, error)
	Set(ctx context.Context, key []byte, value [][]byte) error
	ResetMT(ctx context.Context, depositCount uint) error
	GetRoot(ctx context.Context, depositCount uint) ([]byte, error)
	SetRoot(ctx context.Context, root []byte, depositCount uint) error
	GetLastDepositCount(ctx context.Context) (uint, error)
}

// bridgeStorage interface for the Bridge Tree
type bridgeStorage interface {
	GetDepositCountByRoot(ctx context.Context, root []byte, network uint8) (uint, error)
	GetLatestL1SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	GetLatestL2SyncedExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
	AddExitRoot(ctx context.Context, globalExitRoot *etherman.GlobalExitRoot) error
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address) (*etherman.TokenWrapped, error)
}

// BridgeServiceStorage interface for the Bridge Service.
type BridgeServiceStorage interface {
	GetClaims(ctx context.Context, destAddr string, limit uint, offset uint) ([]*etherman.Claim, error)
	GetClaim(ctx context.Context, index uint, networkID uint) (*etherman.Claim, error)
	GetClaimCount(ctx context.Context, destAddr string) (uint64, error)
	GetDeposit(ctx context.Context, depositCnt uint, networkID uint) (*etherman.Deposit, error)
	GetDeposits(ctx context.Context, destAddr string, limit uint, offset uint) ([]*etherman.Deposit, error)
	GetDepositCount(ctx context.Context, destAddr string) (uint64, error)
}
