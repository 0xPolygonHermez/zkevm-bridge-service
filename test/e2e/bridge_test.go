package e2e

import (
	"context"
	"encoding/hex"
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

			//Run environment
			require.NoError(t, opsman.Setup())

			// Check initial globalExitRoot. Must fail because at the beggining, no globalExitRoot event is thrown.
			globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
			require.NoError(t, err)			
			t.Logf("initial globalExitRootSMC.GlobalExitRootNum: %+v,", globalExitRootSMC)

			// Send L1 deposit
			var destNetwork uint32 = 1
			amount := new(big.Int).SetUint64(10000000000000000000)
			// tokenAddr := common.HexToAddress(operations.MaticTokenAddress)
			tokenAddr := common.Address{} // This means is eth
			destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
			err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			require.NoError(t, err)
			// Check globalExitRoot
			globalExitRoot2, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			assert.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			assert.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			assert.Equal(t, common.HexToHash("0x843cb84814162b93794ad9087a037a1948f9aff051838ba3a93db0ac92b9f719"), globalExitRoot2.ExitRoots[0])
			// Wait until a new batch proposal appears
			t.Log("time1: ", time.Now())
			time.Sleep(15 * time.Second)
			t.Log("time2: ", time.Now())
			// Get Bridge Info By DestAddr
			deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
			require.NoError(t, err)
			// Check L2 funds
			balance, err := opsman.CheckAccountBalance(ctx, "l2", &destAddr)
			require.NoError(t, err)
			initL2Balance := big.NewInt(0)
			assert.Equal(t, 0, balance.Cmp(initL2Balance))
			t.Log(deposits)
			t.Log("Before getClaimData: ", deposits[0].OrigNet, deposits[0].DepositCnt)
			// Get the claim data
			smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].OrigNet), uint(deposits[0].DepositCnt))
			require.NoError(t, err)
			proof := testCase.Txs[0].Params[5].([]interface{})
			assert.Equal(t, len(proof), len(smtProof))
			for i,s := range smtProof {
				assert.Equal(t, proof[i].(string), "0x" + hex.EncodeToString(s[:]))
			}
			// Force to propose a new batch
			err = opsman.ForceBatchProposal(ctx)
			require.NoError(t, err)
			// Claim funds in L1
			err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
			t.Log("Error claiming l2 funds: ", err)
			require.NoError(t, err)
			// Check L2 funds to see if the amount has been increased
			balance2, err := opsman.CheckAccountBalance(ctx, "l2", &destAddr)
			require.NoError(t, err)
			fmt.Println("Balance: ", balance, balance2)
			assert.NotEqual(t, balance, balance2)







			// Check globalExitRoot
			globalExitRoot3, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			// Send L2 Deposit to withdraw the some funds
			destNetwork = 0
			amount = new(big.Int).SetUint64(1000000000000000000)
			err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			require.NoError(t, err)
			panic(1)
			// Get Bridge Info By DestAddr
			deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, nil)
			require.NoError(t, err)
			t.Log("Deposits 2: ", deposits)
			// Check globalExitRoot
			globalExitRoot4, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			t.Logf("Global3 %+v: ", globalExitRoot3)
			t.Logf("Global4 %+v: ", globalExitRoot4)
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
			t.Log("smt2: ", smtProof)
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
