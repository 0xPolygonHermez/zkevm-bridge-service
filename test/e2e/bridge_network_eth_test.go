//go:build e2e_real_network
// +build e2e_real_network

package e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCLaimAlreadyClaimedDepositL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	showCurrentStatus(t, ctx, testData)
	require.NoError(t, err)
	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	txAssetHash := common.HexToHash("0x099252f808424cd7779f8b43038600efd47cdfdde6c077e0c1089ef742aaab36")
	deposit, err := waitDepositByTxHash(ctx, testData, txAssetHash, maxTimeToClaimReady)
	require.NoError(t, err)
	fmt.Println("Deposit: ", deposit)

	err = manualClaimDeposit(ctx, testData, deposit)
	if !isAlreadyClaimedError(err) {
		require.NoError(t, err)
	}
	ethFinalBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Initial Balance ", ethFinalBalances.String())
	fmt.Println("ETH Final   Balance ", ethFinalBalances.String())
}

// ETH L1 -> L2
// Bridge Service (L1): BridgeAsset
//   - Bridge do the autoclaim ( Claim -> L2/Bridge Contract)
//   - Find the autoclaim tx
//   - done
func TestEthTransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)

	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	chainL2, err := testData.L2Client.ChainID(ctx)
	require.NoError(t, err)
	fmt.Println("Chain ID L2: ", chainL2.String())
	auth := testData.auth[operations.L1]

	//amount := big.NewInt(12345678)
	//amount := ethInitialBalances.balanceL1.Div(ethInitialBalances.balanceL1, big.NewInt(4))
	amount := big.NewInt(143210000000001234) // 0.14321 ETH
	txAssetHash := assetEthL1ToL2(ctx, testData, t, auth, amount)
	waitToAutoClaim(t, ctx, testData, txAssetHash, maxTimeToAutoClaim)
	ethFinalBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("AFTER ETH Balance ", ethFinalBalances.String())
	//checkFinalBalanceL1toL2(t, ctx, testData, ethInitialBalances, amount)
}

// This case we need to do manually the claim of the asset
func TestEthTransferL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)

	checkBridgeServiceIsAliveAndExpectedVersion(t, testData)
	showCurrentStatus(t, ctx, testData)
	ethInitialBalances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	fmt.Println("ETH Balance ", ethInitialBalances.String())
	amount := big.NewInt(12344321)
	txAssetHash := assetEthL2ToL1(ctx, testData, t, amount)
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady)
	require.NoError(t, err)
	err = manualClaimDeposit(ctx, testData, deposit)
	require.NoError(t, err)
}
