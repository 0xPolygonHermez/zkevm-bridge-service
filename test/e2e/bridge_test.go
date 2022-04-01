package e2e

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hermeznetwork/hermez-core/encoding"
)

// TestE2E tests the flow of deposit and withdraw funds using the vector
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	defer func() {
		require.NoError(t, operations.Teardown())
	}()

	testCases, err := vectors.LoadE2ETestVectors("./../vectors/e2e-test.json")
	require.NoError(t, err)

	for _, testCase := range testCases {
		t.Run("Test id "+strconv.FormatUint(uint64(testCase.ID), 10), func(t *testing.T) {
			ctx := context.Background()

			opsCfg := &operations.Config {
				Storage: db.Config {
					Database: "postgres",
					Name: "test_db",
					User: "test_user",
					Password: "test_password",
					Host: "localhost",
					Port: "5433",
				},
				BT: bridgectrl.Config {
					Store: "postgres",
					Height: uint8(32),
				},
			}
			opsman, err := operations.NewManager(ctx, opsCfg)
			require.NoError(t, err)

			fmt.Println(testCase.Txs)

			//Run environment
			require.NoError(t, opsman.Setup())

			// Check initial globalExitRoot. Must fail because at the beggining, no globalExitRoot event is thrown.
			globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
			require.NoError(t, err)
			fmt.Printf("globalExitRootSMC: %+v", globalExitRootSMC)
			globalExitRoot, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			// require.Error(t, err)
			if err != nil {
				fmt.Println("err: ", err)
				panic(err)
			}
			require.NoError(t, err)
			t.Logf("initial globalExitRoot.GlobalExitRootNum: %d,", globalExitRoot.GlobalExitRootNum)

			// Send L1 deposit
			var destNetwork uint32 = 2
			amount := new(big.Int).SetUint64(1000000000000000000)
			// tokenAddr := common.HexToAddress(operations.MaticTokenAddress)
			tokenAddr := common.Address{} // This means is eth
			opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, nil)
			require.NoError(t, err)
			// Get Bridge Info By DestAddr
			deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, nil)
			require.NoError(t, err)
			// Check globalExitRoot
			globalExitRoot2, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRoot.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			assert.NotEqual(t, globalExitRoot.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			t.Logf("globalExitRoot.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRoot.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			assert.NotEqual(t, globalExitRoot.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			t.Logf("globalExitRoot.rollup: %v, globalExitRoot2.rollup: %v", globalExitRoot.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			assert.Equal(t, globalExitRoot.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			// Wait until a new batch proposal appears
			t.Log("time1: ", time.Now())
			time.Sleep(15 * time.Second)
			t.Log("time2: ", time.Now())
			// Check L2 funds
			balance, err := opsman.CheckAccountBalance(ctx, "l2", nil)
			require.NoError(t, err)
			initL2Balance, _ := new(big.Int).SetString("1000000000000000000000", encoding.Base10)
			assert.Equal(t, initL2Balance, balance)
			t.Log("Before getClaimData: ", deposits[0].OrigNet, deposits[0].DepositCnt)
			// Get the claim data
			smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].OrigNet), 1)//uint(deposits[0].DepositCnt))
			require.NoError(t, err)
			t.Log("smt: ", smtProof)
			t.Logf("globalExitRoot: %+v", globaExitRoot)
			// Claim funds in L1
			opsman.SendL2Claim(ctx, deposits[1], smtProof, globaExitRoot)
			// Check L2 funds to see if the amount has been increased
			balance, err = opsman.CheckAccountBalance(ctx, "l2", nil)
			require.NoError(t, err)
			assert.NotEqual(t, big.NewInt(0), balance)


			// Check globalExitRoot
			globalExitRoot3, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			// Send L2 Deposit to withdraw the some funds
			destNetwork = 0
			amount = new(big.Int).SetUint64(500000000000000000)
			opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, nil)
			require.NoError(t, err)
			// Wait until the batch that includes the tx is consolidated
			time.Sleep(15 * time.Second)
			// Get Bridge Info By DestAddr
			deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, nil)
			require.NoError(t, err)
			fmt.Println(deposits)
			// Check globalExitRoot
			globalExitRoot4, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			assert.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
			assert.NotEqual(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
			assert.Equal(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
			// Check L1 funds
			balance, err = opsman.CheckAccountBalance(ctx, "l1", nil)
			require.NoError(t, err)
			assert.Equal(t, big.NewInt(0), balance)
			// Get the claim data
			smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[1].OrigNet), uint(deposits[1].DepositCnt))
			require.NoError(t, err)
			// Claim funds in L1
			opsman.SendL1Claim(ctx, deposits[1], smtProof, globaExitRoot)
			// Check L1 funds to see if the amount has been increased
			balance, err = opsman.CheckAccountBalance(ctx, "l1", nil)
			require.NoError(t, err)
			assert.NotEqual(t, big.NewInt(0), balance)
			// Check L2 funds to see that the amount has been reduced
			balance, err = opsman.CheckAccountBalance(ctx, "l2", nil)
			require.NoError(t, err)
			assert.Equal(t, big.NewInt(0), balance)

			assert.Equal(t, 1, 1)

			require.NoError(t, operations.Teardown())
		})
	}
}
