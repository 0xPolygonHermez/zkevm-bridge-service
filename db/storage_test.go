package db

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitRootStore(t *testing.T) {
	ctx := context.Background()
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
	storage, err := NewStorage(storageCfg, networksNumber)
	require.NoError(t, err)
	var networkID uint = 1
	_, err = storage.GetLatestL1SyncedExitRoot(ctx, nil)
	require.Error(t, err)

	var exitRoot etherman.GlobalExitRoot
	exitRoot.BlockNumber = 1
	exitRoot.GlobalExitRootNum = big.NewInt(1)
	exitRoot.ExitRoots = []common.Hash{common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"), common.HexToHash("0x30e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9ed")}

	var block etherman.Block
	block.BlockNumber = 1
	block.BlockHash = common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc")
	block.NetworkID = networkID
	id, err := storage.AddBlock(ctx, &block, nil)
	require.NoError(t, err)
	exitRoot.BlockID = id
	err = storage.AddGlobalExitRoot(ctx, &exitRoot, nil)
	require.NoError(t, err)

	exit, err := storage.GetLatestExitRoot(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, exitRoot.GlobalExitRootNum, exit.GlobalExitRootNum)
	assert.Equal(t, exitRoot.ExitRoots[0], exit.ExitRoots[0])
	assert.Equal(t, exitRoot.ExitRoots[1], exit.ExitRoots[1])

	// Claim
	amount, _ := new(big.Int).SetString("100000000000000000000000000000000000000000000000", 10)
	claim := etherman.Claim{
		Index:              1,
		OriginalNetwork:    1,
		Token:              common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		Amount:             amount,
		DestinationAddress: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		BlockNumber:        1,
		BlockID:            id,
		NetworkID:          2,
		TxHash:             common.HexToHash("0xc67996b03ac2ca401822b5be568e828f432121dd07ddedb306d9203e67675db8"),
	}
	err = storage.AddClaim(ctx, &claim, nil)
	require.NoError(t, err)

	claimStored, err := storage.GetClaim(ctx, claim.Index, claim.NetworkID, nil)
	require.NoError(t, err)
	assert.Equal(t, claim.Amount, claimStored.Amount)
	assert.Equal(t, claim.BlockNumber, claimStored.BlockNumber)
	assert.Equal(t, claim.DestinationAddress, claimStored.DestinationAddress)
	assert.Equal(t, claim.Index, claimStored.Index)
	assert.Equal(t, claim.OriginalNetwork, claimStored.OriginalNetwork)
	assert.Equal(t, claim.Token, claimStored.Token)
	assert.Equal(t, claim.TxHash, claimStored.TxHash)

	// Deposit
	deposit := etherman.Deposit{
		DepositCount:       1,
		OriginalNetwork:    1,
		TokenAddress:       common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		Amount:             amount,
		DestinationAddress: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		DestinationNetwork: 2,
		BlockNumber:        1,
		BlockID:            id,
		TxHash:             common.HexToHash("0xc67996b03ac2ca401822b5be568e828f432121dd07ddedb306d9203e67675db8"),
	}
	err = storage.AddDeposit(ctx, &deposit, nil)
	require.NoError(t, err)

	depositStored, err := storage.GetDeposit(ctx, deposit.DepositCount, deposit.NetworkID, nil)
	require.NoError(t, err)
	assert.Equal(t, deposit.Amount, depositStored.Amount)
	assert.Equal(t, deposit.BlockNumber, depositStored.BlockNumber)
	assert.Equal(t, deposit.DestinationAddress, depositStored.DestinationAddress)
	assert.Equal(t, deposit.DepositCount, depositStored.DepositCount)
	assert.Equal(t, deposit.OriginalNetwork, depositStored.OriginalNetwork)
	assert.Equal(t, deposit.TokenAddress, depositStored.TokenAddress)
	assert.Equal(t, deposit.TxHash, depositStored.TxHash)

	// TokenWrapped
	tokenWrapped := etherman.TokenWrapped{
		OriginalNetwork:      1,
		OriginalTokenAddress: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fe"),
		WrappedTokenAddress:  common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		BlockNumber:          1,
		BlockID:              id,
		NetworkID:            2,
	}
	err = storage.AddTokenWrapped(ctx, &tokenWrapped, nil)
	require.NoError(t, err)

	tokenWrappedStored, err := storage.GetTokenWrapped(ctx, tokenWrapped.OriginalNetwork, tokenWrapped.OriginalTokenAddress, nil)
	require.NoError(t, err)
	assert.Equal(t, tokenWrapped.BlockNumber, tokenWrappedStored.BlockNumber)
	assert.Equal(t, tokenWrapped.OriginalNetwork, tokenWrappedStored.OriginalNetwork)
	assert.Equal(t, tokenWrapped.OriginalTokenAddress, tokenWrappedStored.OriginalTokenAddress)
	assert.Equal(t, tokenWrapped.WrappedTokenAddress, tokenWrappedStored.WrappedTokenAddress)
}
