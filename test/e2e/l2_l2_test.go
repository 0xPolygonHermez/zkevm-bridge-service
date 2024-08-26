//go:build l2l2
// +build l2l2

package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestL2L2 tests the flow of deposit and withdraw funds using the vector
func TestL2L2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	opsman1, err := operations.GetOpsman(ctx, "http://localhost:8123", "test_db", "8080", "9090", "5435", 1)
	require.NoError(t, err)
	opsman2, err := operations.GetOpsman(ctx, "http://localhost:8124", "test_db", "8080", "9090", "5435", 2)
	require.NoError(t, err)

	t.Run("L2-L2 eth bridge", func(t *testing.T) {
		// Check initial globalExitRoot. Must fail because at the beginning, no globalExitRoot event is thrown.
		globalExitRootSMC, err := opsman1.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L2 deposit
		var destNetwork uint32 = 2
		amount := new(big.Int).SetUint64(10000000000000000001)
		tokenAddr := common.Address{} // This means is eth
		address := common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266")

		l2Balance, err := opsman1.CheckAccountBalance(ctx, operations.L2, &address)
		require.NoError(t, err)
		t.Logf("Initial L2 Bridge Balance in origin network 1: %v", l2Balance)
		err = opsman1.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &address, operations.L22)
		require.NoError(t, err)
		l2Balance, err = opsman1.CheckAccountBalance(ctx, operations.L2, &address)
		require.NoError(t, err)
		t.Logf("Final L2 Bridge Balance in origin network 1: %v", l2Balance)


		// Get Bridge Info By DestAddr
		deposits, err := opsman1.GetBridgeInfoByDestAddr(ctx, &address)
		require.NoError(t, err)
		t.Log("Deposit: ", deposits[0])
		// Check globalExitRoot
		globalExitRoot, err := opsman1.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		t.Logf("GlobalExitRoot %+v: ", globalExitRoot)
		require.NotEqual(t, globalExitRoot.ExitRoots[1], globalExitRootSMC.ExitRoots[1])
		require.Equal(t, globalExitRoot.ExitRoots[0], globalExitRootSMC.ExitRoots[0])
		// Check L2 destination funds
		balance, err := opsman2.CheckAccountBalance(ctx, operations.L2, &address)
		require.NoError(t, err)
		v, _ := big.NewInt(0).SetString("100000000000000000000000", 10)
		require.Equal(t, 0,  v.Cmp(balance))
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman1.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		time.Sleep(5 * time.Second)
		// Claim funds in destination L2
		err = opsman2.SendL2Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot, operations.L22)
		require.NoError(t, err)

		// Check destination L2 funds to see if the amount has been increased
		balance, err = opsman2.CheckAccountBalance(ctx, operations.L2, &address)
		require.NoError(t, err)
		require.Equal(t, -1, v.Cmp(balance))
		// Check origin L2 funds to see that the amount has been reduced
		balance, err = opsman1.CheckAccountBalance(ctx, operations.L2, &address)
		require.NoError(t, err)
		require.Equal(t, 1, v.Cmp(balance))
	})
}
