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
		BlockID:        1,
		ExitRoots:      []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}

	err = pg.AddGlobalExitRoot(ctx, l1GER, tx)
	require.NoError(t, err)

	ger, err := pg.GetLatestL1SyncedExitRoot(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, ger.BlockID, l1GER.BlockID)
	require.Equal(t, ger.GlobalExitRoot, l1GER.GlobalExitRoot)

	latestGER, err := pg.GetLatestExitRoot(ctx, true, tx)
	require.NoError(t, err)
	require.Equal(t, latestGER.GlobalExitRoot, l1GER.GlobalExitRoot)
	require.Equal(t, latestGER.BlockNumber, l1GER.BlockNumber)
	require.Equal(t, latestGER.ExitRoots[0], l1GER.ExitRoots[0])
	require.Equal(t, latestGER.ExitRoots[1], l1GER.ExitRoots[1])

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
		ExitRoots:      []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1")},
		GlobalExitRoot: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
	}
	isInserted, err := pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.True(t, isInserted)
	require.NoError(t, err)
	getCount := "select count(*) from sync.exit_root where block_id = 0 AND global_exit_root = $1"
	var result int
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	isInserted, err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.False(t, isInserted)
	require.NoError(t, err)
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	require.NoError(t, tx.Commit(ctx))

	tx, err = pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	ger1 := &etherman.GlobalExitRoot{
		ExitRoots:      []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"), common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2")},
		GlobalExitRoot: common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
	}
	isInserted, err = pg.AddTrustedGlobalExitRoot(ctx, ger, tx)
	require.False(t, isInserted)
	require.NoError(t, err)
	err = tx.QueryRow(ctx, getCount, ger.GlobalExitRoot).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
	isInserted, err = pg.AddTrustedGlobalExitRoot(ctx, ger1, tx)
	require.True(t, isInserted)
	require.NoError(t, err)
	getCount2 := "select count(*) from sync.exit_root"
	err = tx.QueryRow(ctx, getCount2).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 2, result)

	tGER, err := pg.GetLatestTrustedExitRoot(ctx, tx)
	require.NoError(t, err)
	require.Equal(t, tGER.GlobalExitRoot, ger1.GlobalExitRoot)

	latestGER, err := pg.GetLatestExitRoot(ctx, false, tx)
	require.NoError(t, err)
	require.Equal(t, latestGER.GlobalExitRoot, ger1.GlobalExitRoot)
	require.Equal(t, latestGER.BlockNumber, ger1.BlockNumber)
	require.Equal(t, latestGER.ExitRoots[0], ger1.ExitRoots[0])
	require.Equal(t, latestGER.ExitRoots[1], ger1.ExitRoots[1])

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

// Test MerkleTree storage
func TestMTStorage(t *testing.T) {
	// Init database instance
	cfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)
	ctx := context.Background()
	pg, err := pgstorage.NewPostgresStorage(cfg)
	require.NoError(t, err)
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	leaf1 := common.FromHex("0xa4bfa0908dc7b06d98da4309f859023d6947561bc19bc00d77f763dea1a0b9f5")
	leaf2 := common.FromHex("0x315fee1aa202bf4a6bd0fde560c89be90b6e6e2aaf92dc5e8d118209abc3410f")
	root := common.FromHex("0x88e652896cb1de5962a0173a222059f51e6b943a2ba6dfc9acbff051ceb1abb5")
	deposit := &etherman.Deposit{
		Metadata: common.Hex2Bytes("0x0"),
	}
	depositID, err := pg.AddDeposit(ctx, deposit, tx)
	require.NoError(t, err)
	err = pg.SetRoot(ctx, root, depositID, 1, 0, tx)
	require.NoError(t, err)

	err = pg.Set(ctx, root, [][]byte{leaf1, leaf2}, depositID, tx)
	require.NoError(t, err)

	vals, err := pg.Get(ctx, root, tx)
	require.NoError(t, err)
	require.Equal(t, vals[0], leaf1)
	require.Equal(t, vals[1], leaf2)

	rRoot, err := pg.GetRoot(ctx, 1, 0, tx)
	require.NoError(t, err)
	require.Equal(t, rRoot, root)

	count, err := pg.GetLastDepositCount(ctx, 0, tx)
	require.NoError(t, err)
	require.Equal(t, count, uint(1))

	dCount, err := pg.GetDepositCountByRoot(ctx, root, 0, tx)
	require.NoError(t, err)
	require.Equal(t, dCount, uint(1))

	require.NoError(t, tx.Commit(ctx))
}

// Test BridgeService storage
func TestBSStorage(t *testing.T) {
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

	deposit := &etherman.Deposit{
		NetworkID:          0,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		BlockNumber:        1,
		BlockID:            1,
		DepositCount:       1,
		Metadata:           common.FromHex("0x000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000005436f696e410000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003434f410000000000000000000000000000000000000000000000000000000000"),
	}
	_, err = pg.AddDeposit(ctx, deposit, tx)
	require.NoError(t, err)

	claim := &etherman.Claim{
		Index:              1,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationAddress: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		BlockID:            1,
		BlockNumber:        2,
		NetworkID:          0,
		TxHash:             common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
	}
	err = pg.AddClaim(ctx, claim, tx)
	require.NoError(t, err)

	count, err := pg.GetDepositCount(ctx, deposit.DestinationAddress.String(), tx)
	require.NoError(t, err)
	require.Equal(t, count, uint64(1))

	rDeposit, err := pg.GetDeposit(ctx, 1, 0, tx)
	require.NoError(t, err)
	require.Equal(t, rDeposit.DestinationAddress, deposit.DestinationAddress)
	require.Equal(t, rDeposit.DepositCount, deposit.DepositCount)

	rDeposits, err := pg.GetDeposits(ctx, deposit.DestinationAddress.String(), 10, 0, tx)
	require.NoError(t, err)
	require.Equal(t, len(rDeposits), 1)

	count, err = pg.GetNumberDeposits(ctx, 0, 0, tx)
	require.NoError(t, err)
	require.Equal(t, count, uint64(0))
	count, err = pg.GetNumberDeposits(ctx, 0, 1, tx)
	require.NoError(t, err)
	require.Equal(t, count, uint64(2))

	count, err = pg.GetClaimCount(ctx, claim.DestinationAddress.String(), tx)
	require.NoError(t, err)
	require.Equal(t, count, uint64(1))

	rClaim, err := pg.GetClaim(ctx, 1, 0, tx)
	require.NoError(t, err)
	require.Equal(t, rClaim.DestinationAddress, claim.DestinationAddress)
	require.Equal(t, rClaim.NetworkID, claim.NetworkID)
	require.Equal(t, rClaim.Index, claim.Index)

	rClaims, err := pg.GetClaims(ctx, claim.DestinationAddress.String(), 10, 0, tx)
	require.NoError(t, err)
	require.Equal(t, len(rClaims), 1)

	wrappedToken := &etherman.TokenWrapped{
		OriginalNetwork:      0,
		OriginalTokenAddress: deposit.OriginalAddress,
		WrappedTokenAddress:  common.HexToAddress("0x187Bd40226A7073b49163b1f6c2b73d8F2aa8478"),
		BlockID:              1,
		BlockNumber:          1,
		NetworkID:            1,
	}
	metadata, err := pg.GetTokenMetadata(ctx, wrappedToken.OriginalNetwork, wrappedToken.NetworkID, wrappedToken.OriginalTokenAddress, tx)
	require.NoError(t, err)
	require.Equal(t, metadata, deposit.Metadata)

	err = pg.AddTokenWrapped(ctx, wrappedToken, tx)
	require.NoError(t, err)

	wt, err := pg.GetTokenWrapped(ctx, wrappedToken.OriginalNetwork, wrappedToken.OriginalTokenAddress, tx)
	require.NoError(t, err)
	require.Equal(t, wt.WrappedTokenAddress, wrappedToken.WrappedTokenAddress)
	require.Equal(t, wt.TokenMetadata.Name, "CoinA")
	require.Equal(t, wt.TokenMetadata.Symbol, "COA")
	require.Equal(t, wt.TokenMetadata.Decimals, uint8(12))

	require.NoError(t, tx.Commit(ctx))
}
