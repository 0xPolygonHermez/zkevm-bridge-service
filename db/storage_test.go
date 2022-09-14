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

func TestL1GlobalExitRoot(t *testing.T) {
	cfg := pgstorage.NewConfigFromEnv()
	// Init database instance
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	block := &etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}

	blockID, err := pg.AddBlock(ctx, block, tx)
	require.NoError(t, err)
	require.Equal(t, blockID, uint64(1))

	l1GER := &etherman.GlobalExitRoot{
		BlockID:           1,
		GlobalExitRootNum: big.NewInt(1),
		ExitRoots:         []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot:    common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}

	err = pg.AddGlobalExitRoot(ctx, l1GER, tx)
	require.NoError(t, err)

	ger, err := pg.GetLatestL1SyncedExitRoot(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, ger.BlockID, l1GER.BlockID)
	require.Equal(t, ger.GlobalExitRootNum, l1GER.GlobalExitRootNum)
	require.Equal(t, ger.GlobalExitRoot, l1GER.GlobalExitRoot)

	latestGER, err := pg.GetLatestExitRoot(ctx, true, tx)
	require.NoError(t, err)
	require.Equal(t, latestGER.GlobalExitRootNum, l1GER.GlobalExitRootNum)

	require.NoError(t, tx.Commit(ctx))
}

func TestAddTrustedGERDuplicated(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg)
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
	getCount := "select count(*) from syncv2.exit_root where block_id = 0 and global_exit_root_num = $1 AND global_exit_root = $2"
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

	tGER, err := pg.GetLatestTrustedExitRoot(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, tGER.GlobalExitRoot, ger1.GlobalExitRoot)

	latestGER, err := pg.GetLatestExitRoot(ctx, false, tx)
	require.NoError(t, err)
	require.Equal(t, latestGER.GlobalExitRootNum, ger1.GlobalExitRootNum)

	require.NoError(t, tx.Commit(ctx))
}

func TestTrustedReset(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg)
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
	err = pg.AddBatch(ctx, &batch1, tx)
	require.NoError(t, err)
	err = pg.AddBatch(ctx, &batch2, tx)
	require.NoError(t, err)
	batchNum, err := pg.GetLastBatchNumber(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, batchNum, uint64(2))
	err = pg.AddBatch(ctx, &batch3, tx)
	require.NoError(t, err)
	lastBatch, err := pg.GetBatchByNumber(ctx, batch3.BatchNumber, tx)
	require.NoError(t, err)
	require.Equal(t, lastBatch.BatchNumber, batch3.BatchNumber)
	require.Equal(t, lastBatch.GlobalExitRoot, batch3.GlobalExitRoot)

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
	pg, err := pgstorage.NewPostgresStorage(cfg)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)
	block1 := etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}
	block2 := etherman.Block{
		BlockNumber: 2,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f3"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f4"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}
	block3 := etherman.Block{
		BlockNumber: 100,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f5"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f6"),
		NetworkID:   1,
		ReceivedAt:  time.Now(),
	}
	block4 := etherman.Block{
		BlockNumber: 101,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f7"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f8"),
		NetworkID:   1,
		ReceivedAt:  time.Now(),
	}
	_, err = pg.AddBlock(ctx, &block1, tx)
	require.NoError(t, err)
	b, err := pg.GetLastBlock(ctx, 0, tx)
	require.NoError(t, err)
	assert.Equal(t, block1.BlockNumber, b.BlockNumber)
	assert.Equal(t, block1.BlockHash, b.BlockHash)
	assert.Equal(t, block1.ParentHash, b.ParentHash)
	assert.Equal(t, block1.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block2, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 0, tx)
	require.NoError(t, err)
	assert.Equal(t, block2.BlockNumber, b.BlockNumber)
	assert.Equal(t, block2.BlockHash, b.BlockHash)
	assert.Equal(t, block2.ParentHash, b.ParentHash)
	assert.Equal(t, block2.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block3, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 1, tx)
	require.NoError(t, err)
	assert.Equal(t, block3.BlockNumber, b.BlockNumber)
	assert.Equal(t, block3.BlockHash, b.BlockHash)
	assert.Equal(t, block3.ParentHash, b.ParentHash)
	assert.Equal(t, block3.NetworkID, b.NetworkID)

	_, err = pg.AddBlock(ctx, &block4, tx)
	require.NoError(t, err)
	b, err = pg.GetLastBlock(ctx, 1, tx)
	require.NoError(t, err)
	assert.Equal(t, block4.BlockNumber, b.BlockNumber)
	assert.Equal(t, block4.BlockHash, b.BlockHash)
	assert.Equal(t, block4.ParentHash, b.ParentHash)
	assert.Equal(t, block4.NetworkID, b.NetworkID)

	prevBlock, err := pg.GetPreviousBlock(ctx, 1, 1, tx)
	require.NoError(t, err)
	require.Equal(t, prevBlock.BlockNumber, block3.BlockNumber)
	require.Equal(t, prevBlock.BlockHash, block3.BlockHash)

	require.NoError(t, tx.Commit(ctx))
}

func TestForcedAndVerifiedBatch(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	block := &etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}
	_, err = pg.AddBlock(ctx, block, tx)
	require.NoError(t, err)

	batch := &etherman.Batch{
		BatchNumber:    1,
		Coinbase:       common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BatchL2Data:    []byte{},
		Timestamp:      time.Now(),
		GlobalExitRoot: common.HexToHash("0x1d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
	}
	err = pg.AddBatch(ctx, batch, tx)
	require.NoError(t, err)

	fb := &etherman.ForcedBatch{
		BlockID:           1,
		BlockNumber:       1,
		ForcedBatchNumber: 1,
		Sequencer:         common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		GlobalExitRoot:    common.HexToHash("0x3d02f31780d083b996faee908120beef6366b5a6cab3f9efbe5a1f7e9ad47ba8"),
		RawTxsData:        []byte{},
		ForcedAt:          time.Now(),
	}
	err = pg.AddForcedBatch(ctx, fb, tx)
	require.NoError(t, err)

	vb := &etherman.VerifiedBatch{
		BatchNumber: 1,
		BlockID:     1,
		BlockNumber: 1,
		Aggregator:  common.HexToAddress("0x0165878A594ca255338adfa4d48449f69242Eb8F"),
		TxHash:      common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
	}
	err = pg.AddVerifiedBatch(ctx, vb, tx)
	require.NoError(t, err)

	fbs, err := pg.GetNextForcedBatches(ctx, 1, tx)
	require.NoError(t, err)
	require.Equal(t, len(fbs), 1)
	require.Equal(t, fbs[0].BlockID, fb.BlockID)
	require.Equal(t, fbs[0].Sequencer, fb.Sequencer)
	require.Equal(t, fbs[0].GlobalExitRoot, fb.GlobalExitRoot)
	require.Equal(t, fbs[0].BatchNumber, fb.BatchNumber)
	require.Equal(t, fbs[0].ForcedBatchNumber, fb.ForcedBatchNumber)

	err = pg.AddBatchNumberInForcedBatch(ctx, 1, 2, tx)
	require.NoError(t, err)
	fbs, err = pg.GetNextForcedBatches(ctx, 1, tx)
	require.NoError(t, err)
	require.Equal(t, len(fbs), 0)

	require.NoError(t, tx.Commit(ctx))
}
