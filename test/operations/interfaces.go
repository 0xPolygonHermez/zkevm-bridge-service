package operations

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

// StorageInterface is a storage interface.
type StorageInterface interface {
	GetLastBlock(ctx context.Context, networkID uint, dbTx pgx.Tx) (*etherman.Block, error)
	GetLatestExitRoot(ctx context.Context, isRollup bool, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestL1SyncedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetLatestTrustedExitRoot(ctx context.Context, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error)
	GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error)
	UpdateBlocksForTesting(ctx context.Context, networkID uint, blockNum uint64, dbTx pgx.Tx) error
	GetClaim(ctx context.Context, depositCount, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error)
	UpdateDepositsStatusForTesting(ctx context.Context, dbTx pgx.Tx) error
	// synchronizer
	AddBlock(ctx context.Context, block *etherman.Block, dbTx pgx.Tx) (uint64, error)
	AddGlobalExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) error
	AddTrustedGlobalExitRoot(ctx context.Context, trustedExitRoot *etherman.GlobalExitRoot, dbTx pgx.Tx) (bool, error)
	AddDeposit(ctx context.Context, deposit *etherman.Deposit, dbTx pgx.Tx) (uint64, error)
	AddClaim(ctx context.Context, claim *etherman.Claim, dbTx pgx.Tx) error
	AddTokenWrapped(ctx context.Context, tokenWrapped *etherman.TokenWrapped, dbTx pgx.Tx) error
	// atomic
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error
}

// BridgeServiceInterface is an interface for the bridge service.
type BridgeServiceInterface interface {
	GetBridges(ctx context.Context, req *pb.GetBridgesRequest) (*pb.GetBridgesResponse, error)
	GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error)
}
