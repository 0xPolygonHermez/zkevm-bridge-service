package e2e

import (
	"context"
	"fmt"
	"testing"
	"strconv"


	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/bridgetree"
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

	testCases, err := vectors.LoadE2ETestVectors("./../vectors/src/test-vector-data/e2e-test.json")
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
				BT: bridgetree.Config {
					Store: "postgres",
					Height: uint8(32),
				},
			}
			opsman, err := operations.NewManager(ctx, opsCfg)
			require.NoError(t, err)

			fmt.Println(testCase.Txs)

			//Run environment
			require.NoError(t, opsman.Setup())

			// TODO Check initial globalExitRoot
			// TODO Send L1 deposit
			// TODO Check globalExitRoot
			// TODO Wait until a new batch proposal appears
			// TODO Check L2 funds
			// TODO Claim funds in L2
			// TODO Check L2 funds to see if the amount has been increased

			// TODO Check initial globalExitRoot
			// TODO Send L2 Deposit to withdraw the some funds
			// TODO Wait until the batch that includes the tx is consolidated
			// TODO Check GlobalExitRoot
			// TODO Check L1 funds
			// TODO Claim funds
			// TODO Check L1 funds to see if the amount has been increased

			assert.Equal(t, 1, 1)

			require.NoError(t, operations.Teardown())
		})
	}
}
