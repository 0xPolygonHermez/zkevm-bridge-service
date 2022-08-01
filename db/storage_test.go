package db

import (
	"context"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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

func TestTrustedReset(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg, 1)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)
	batch1 := etherman.Batch {
		BatchNumber:    1,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x1d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	batch2 := etherman.Batch {
		BatchNumber:    2,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x2d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	batch3 := etherman.Batch {
		BatchNumber:    3,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x3d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	insertBatch := "INSERT INTO syncv2.batch (batch_num, sequencer, raw_tx_data, global_exit_root, timestamp) VALUES ($1, $2, $3, $4, $5)"
	_, err = tx.Exec(ctx, insertBatch, batch1.BatchNumber, batch1.Coinbase, batch1.BatchL2Data, batch1.GlobalExitRoot, batch1.Timestamp)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, insertBatch, batch2.BatchNumber, batch2.Coinbase, batch2.BatchL2Data, batch2.GlobalExitRoot, batch2.Timestamp)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, insertBatch, batch3.BatchNumber, batch3.Coinbase, batch3.BatchL2Data, batch3.GlobalExitRoot, batch3.Timestamp)
	require.NoError(t, err)

	getCount := "select count(*) from syncv2.batch"
	var result int
	err = tx.QueryRow(ctx, getCount).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 3, result)
	err = pg.ResetTrustedState(ctx, 1, tx)
	require.NoError(t, err)

	err = tx.QueryRow(ctx, getCount).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	require.NoError(t, tx.Commit(ctx))
}