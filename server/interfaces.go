package server

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

type bridgeServiceStorage interface {
	Get(ctx context.Context, key []byte, dbTx pgx.Tx) ([][]byte, error)
	GetRoot(ctx context.Context, depositCnt uint, network uint, dbTx pgx.Tx) ([]byte, error)
	GetDepositCountByRoot(ctx context.Context, root []byte, network uint8, dbTx pgx.Tx) (uint, error)
	GetLatestExitRoot(ctx context.Context, isRollup bool, dbTx pgx.Tx) (*etherman.GlobalExitRoot, error)
	GetClaim(ctx context.Context, index uint, networkID uint, dbTx pgx.Tx) (*etherman.Claim, error)
	GetClaims(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Claim, error)
	GetClaimCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error)
	GetDeposit(ctx context.Context, depositCnt uint, networkID uint, dbTx pgx.Tx) (*etherman.Deposit, error)
	GetDeposits(ctx context.Context, destAddr string, limit uint, offset uint, dbTx pgx.Tx) ([]*etherman.Deposit, error)
	GetDepositCount(ctx context.Context, destAddr string, dbTx pgx.Tx) (uint64, error)
	GetTokenWrapped(ctx context.Context, originalNetwork uint, originalTokenAddress common.Address, dbTx pgx.Tx) (*etherman.TokenWrapped, error)
}
