package db

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestExitRootStore(t *testing.T) {
	cfg := pgstorage.NewConfigFromEnv()
	// Init database instance
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)

	storageCfg := Config{
		Database: "postgres",
		Name:     cfg.Name,
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
		MaxConns: 20,
	}
	var networksNumber uint = 2
	_, err = NewStorage(storageCfg, networksNumber)
	require.NoError(t, err)
}

func TestAddTrustedGERDuplicated(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg, 1)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	ger := &etherman.GlobalExitRoot{
		GlobalExitRootNum: big.NewInt(1),
		ExitRoots:         []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot:    common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}
	err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.NoError(t, err)
	err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.Error(t, err)
	require.Error(t, tx.Commit(ctx))

	tx, err = pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	ger1 := &etherman.GlobalExitRoot{
		GlobalExitRootNum: big.NewInt(2),
		ExitRoots:         []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot:    common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}
	err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.NoError(t, err)
	err = pg.AddTrustedGlobalExitRoot(ctx, ger1, tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
}
