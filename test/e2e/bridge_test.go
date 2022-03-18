package e2e

import (
	"context"
	"fmt"
	"testing"
	"strconv"


	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

// E2ETest tests the flow of deposit and withdraw funds using the vector
func E2ETest(t *testing.T) {
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

			opsCfg := &operations.Config{
				
			}
			opsman, err := operations.NewManager(ctx, opsCfg)
			require.NoError(t, err)

			fmt.Println(opsman, testCase.Txs)

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
