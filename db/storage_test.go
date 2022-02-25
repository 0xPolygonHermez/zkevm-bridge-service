package db

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitRootStore(t *testing.T) {
	ctx := context.Background()
	cfg := pgstorage.NewConfigFromEnv()
	// Init database instance
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)

 storageCfg := Config {
	Database: "postgres",
	Name: cfg.Name,
	User: cfg.User,
	Password: cfg.Password,
	Host: cfg.Host,
	Port: cfg.Port,
}
	storage, err := NewStorage(storageCfg)
	require.NoError(t, err)

	_, err = storage.GetLatestExitRoot(ctx)
	require.Error(t, err)

	var exitRoot etherman.GlobalExitRoot
	exitRoot.BlockNumber = 1
	exitRoot.GlobalExitRootNum = big.NewInt(1)
	exitRoot.MainnetExitRoot = common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc")
	exitRoot.RollupExitRoot = common.HexToHash("0x30e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9ed")

	var block etherman.Block
	block.BlockNumber = 1
	err = storage.AddBlock(ctx, &block)
	require.NoError(t, err)
	err = storage.AddL2Block(ctx, &block)
	require.NoError(t, err)
	err = storage.AddExitRoot(ctx, &exitRoot)
	require.NoError(t, err)

	exit, err := storage.GetLatestExitRoot(ctx)
	require.NoError(t, err)
	assert.Equal(t, exitRoot.GlobalExitRootNum, exit.GlobalExitRootNum)
	assert.Equal(t, exitRoot.MainnetExitRoot, exit.MainnetExitRoot)
	assert.Equal(t, exitRoot.RollupExitRoot, exit.RollupExitRoot)

	// Claim
	amount, _ := new(big.Int).SetString("100000000000000000000000000000000000000000000000", 10)
	claim := etherman.Claim {
		Index: 1, 
		OriginalNetwork: 1,
		Token: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		Amount: amount,
		DestinationAddress: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		BlockNumber: 1,
	}
	err = storage.AddClaim(ctx, &claim)
	require.NoError(t, err)

	claimStored, err := storage.GetClaim(ctx, claim.Index, claim.OriginalNetwork)
	require.NoError(t, err)
	assert.Equal(t, claim.Amount, claimStored.Amount)
	assert.Equal(t, claim.BlockNumber, claimStored.BlockNumber)
	assert.Equal(t, claim.DestinationAddress, claimStored.DestinationAddress)
	assert.Equal(t, claim.Index, claimStored.Index)
	assert.Equal(t, claim.OriginalNetwork, claimStored.OriginalNetwork)
	assert.Equal(t, claim.Token, claimStored.Token)

	// L2Claim
	l2Claim := etherman.Claim {
		Index: 1, 
		OriginalNetwork: 1,
		Token: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		Amount: amount,
		DestinationAddress: common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		BlockNumber: 1,
	}
	err = storage.AddL2Claim(ctx, &l2Claim)
	require.NoError(t, err)

	l2ClaimStored, err := storage.GetClaim(ctx, l2Claim.Index, l2Claim.OriginalNetwork)
	require.NoError(t, err)
	assert.Equal(t, l2Claim.Amount, l2ClaimStored.Amount)
	assert.Equal(t, l2Claim.BlockNumber, l2ClaimStored.BlockNumber)
	assert.Equal(t, l2Claim.DestinationAddress, l2ClaimStored.DestinationAddress)
	assert.Equal(t, l2Claim.Index, l2ClaimStored.Index)
	assert.Equal(t, l2Claim.OriginalNetwork, l2ClaimStored.OriginalNetwork)
	assert.Equal(t, l2Claim.Token, l2ClaimStored.Token)

}
