//go:build edge
// +build edge

package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func depositFromL1(ctx context.Context, opsman *operations.Manager, t *testing.T) {
	amount := new(big.Int).SetUint64(2500000000000000000)
	tokenAddr := common.Address{} // This means is eth
	destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	var destNetwork uint32 = 1
	// L1 Deposit
	err := opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)

	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	// Get the claim data
	smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].OrigNet), uint(deposits[0].DepositCnt))
	require.NoError(t, err)
	// Claim funds in L2
	err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
	require.NoError(t, err)
}

func depositFromL2(ctx context.Context, opsman *operations.Manager, t *testing.T) {
	// Send L2 Deposit to withdraw the some funds
	var destNetwork uint32 = 0
	amount := new(big.Int).SetUint64(100000000000000000)
	tokenAddr := common.Address{} // This means is eth
	destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	err := opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)

	// Get Bridge Info By DestAddr
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	// Check globalExitRoot
	// Get the claim data
	smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
	require.NoError(t, err)
	// Claim funds in L1
	err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
	require.NoError(t, err)
}

func TestEdgeCase(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	opsCfg := &operations.Config{
		Storage: db.Config{
			Database: "postgres",
			Name:     "test_db",
			User:     "test_user",
			Password: "test_password",
			Host:     "localhost",
			Port:     "5435",
			MaxConns: 10,
		},
		BT: bridgectrl.Config{
			Store:  "postgres",
			Height: uint8(32),
		},
	}

	opsman, err := operations.NewManager(ctx, opsCfg)
	require.NoError(t, err)
	require.NoError(t, opsman.StartBridge())
	const st time.Duration = 20 // wait until the syncing is finished
	time.Sleep(st * time.Second)

	t.Run("Test a case of restart with reorg.", func(t *testing.T) {
		depositFromL1(ctx, opsman, t)
		depositFromL2(ctx, opsman, t)
		// Modify the L1 blocks for L1 reorg
		require.NoError(t, opsman.UpdateBlocksForTesting(ctx, 0, 1))
		// Modify the batch data to check the trusted state reorg
		batchNum, err := opsman.GetLastBatchNumber(ctx)
		require.NoError(t, err)
		require.NoError(t, opsman.UpdateBatchesForTesting(ctx, batchNum))
		// Restart the bridge service.
		require.NoError(t, opsman.StartBridge())
		time.Sleep(st * time.Second)

		depositFromL2(ctx, opsman, t)
		depositFromL1(ctx, opsman, t)
	})
}
