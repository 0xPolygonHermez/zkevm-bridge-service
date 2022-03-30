package e2e

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl"
	"github.com/hermeznetwork/hermez-bridge/db"

	// "github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
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

			// // TODO Check initial globalExitRoot
			// opsman.GetCurrentGlobalExitRootFromSmc()
			// TODO Send L1 deposit
			var destNetwork uint32 = 2
			amount := new(big.Int).SetUint64(1000000000000000000)
			opsman.SendL1Deposit(ctx, common.HexToAddress(operations.MaticTokenAddress), amount, destNetwork, nil)
			require.NoError(t, err)
			// // TODO Check globalExitRoot
			// opsman.GetCurrentGlobalExitRootFromSmc()
			// // TODO Wait until a new batch proposal appears
			// opsman.WaitTxToBeMined()
			// // TODO Check L2 funds
			// opsman.CheckAccountBalance()
			// // TODO Claim funds in L2
			// opsman.SendL2Claim()
			// // TODO Check L2 funds to see if the amount has been increased
			// opsman.CheckAccountBalance()


			// // TODO Check initial globalExitRoot
			// opsman.GetCurrentGlobalExitRootFromSmc()
			// // TODO Send L2 Deposit to withdraw the some funds
			// opsman.SendL2Deposit()
			// // TODO Wait until the batch that includes the tx is consolidated
			// opsman.WaitTxToBeMined()
			// // TODO Check GlobalExitRoot
			// opsman.GetCurrentGlobalExitRootFromSmc()
			// // TODO Check L1 funds
			// opsman.CheckAccountBalance()
			// // TODO Claim funds
			// opsman.SendL2Claim()
			// // TODO Check L2 funds to see if the amount has been increased
			// opsman.CheckAccountBalance()

			assert.Equal(t, 1, 1)

			require.NoError(t, operations.Teardown())
		})
	}
}
