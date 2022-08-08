package db

import (
	"context"
	"math/big"
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
	getCount := "select count(*) from syncv2.exit_root where block_id is null and global_exit_root_num = $1 AND global_exit_root = $2"
	var result int
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRootNum.String(), ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.NoError(t, err)
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRootNum.String(), ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	require.NoError(t, tx.Commit(ctx))

	tx, err = pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	ger1 := &etherman.GlobalExitRoot{
		GlobalExitRootNum: big.NewInt(2),
		ExitRoots:         []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot:    common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}
	err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.NoError(t, err)
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRootNum.String(), ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	err = pg.AddTrustedGlobalExitRoot(ctx, ger1, tx)
	require.NoError(t, err)
	getCount2 := "select count(*) from syncv2.exit_root"
	err = tx.QueryRow(ctx, getCount2).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 2, result)
	require.NoError(t, tx.Commit(ctx))
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
	batch1 := etherman.Batch{
		BatchNumber:    1,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x1d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	batch2 := etherman.Batch{
		BatchNumber:    2,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x2d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	batch3 := etherman.Batch{
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

func TestGetLastBlock(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg, 1)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)
	block1 := etherman.Block {
		BlockNumber: 1,
		BlockHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID: 0,
		ReceivedAt: time.Now(),
	}
	block2 := etherman.Block {
		BlockNumber: 2,
		BlockHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f3"),
		ParentHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f4"),
		NetworkID: 0,
		ReceivedAt: time.Now(),
	}
	block3 := etherman.Block {
		BlockNumber: 100,
		BlockHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f5"),
		ParentHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f6"),
		NetworkID: 1,
		ReceivedAt: time.Now(),
	}
	block4 := etherman.Block {
		BlockNumber: 101,
		BlockHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f7"),
		ParentHash: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f8"),
		NetworkID: 1,
		ReceivedAt: time.Now(),
	}
	_, err = pg.AddBlock(ctx, &block1, tx)
	require.NoError(t, err)
	b, err := pg.GetLastBlock(ctx, 0, tx)
	assert.Equal(t, block1.BlockNumber, b.BlockNumber)
	assert.Equal(t, block1.BlockHash, b.BlockHash)
	assert.Equal(t, block1.ParentHash, b.ParentHash)
	assert.Equal(t, block1.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block2, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 0, tx)
	assert.Equal(t, block2.BlockNumber, b.BlockNumber)
	assert.Equal(t, block2.BlockHash, b.BlockHash)
	assert.Equal(t, block2.ParentHash, b.ParentHash)
	assert.Equal(t, block2.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block3, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 1, tx)
	assert.Equal(t, block3.BlockNumber, b.BlockNumber)
	assert.Equal(t, block3.BlockHash, b.BlockHash)
	assert.Equal(t, block3.ParentHash, b.ParentHash)
	assert.Equal(t, block3.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block4, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 1, tx)
	assert.Equal(t, block4.BlockNumber, b.BlockNumber)
	assert.Equal(t, block4.BlockHash, b.BlockHash)
	assert.Equal(t, block4.ParentHash, b.ParentHash)
	assert.Equal(t, block4.NetworkID, b.NetworkID)
	require.NoError(t, tx.Commit(ctx))
}
