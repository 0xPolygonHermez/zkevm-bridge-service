//go:build e2e_real_network_erc20
// +build e2e_real_network_erc20

package e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	erc20 "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestERC20TransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	log.Infof("Starting test ERC20 L1 -> L2")
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	balances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	log.Infof("ERC20 [L1->L2]  balances: %s", balances.String())

	log.Infof("ERC20 [L1->L2] Deploy token")
	tokenAddr, err := deployToken(ctx, testData.L1Client, testData.auth[operations.L1], testData.cfg.ConnectionConfig.L1BridgeAddr)
	require.NoError(t, err)
	log.Infof("ERC20 [L1->L2] L1: Token Addr: ", tokenAddr.Hex())

	// Do Asset L1 -> L2
	amount := big.NewInt(143210000000001234) // 0.14321 ETH
	log.Infof("ERC20 [L1->L2] assetERC20L2ToL1")
	txAssetHash := assetERC20L1ToL2(ctx, testData, t, tokenAddr, amount)
	log.Infof("ERC20 [L1->L2] assetERC20L2ToL1 txAssetHash: %s ", txAssetHash.String())
	log.Infof("ERC20 [L1->L2] waitToAutoClaim")
	//waitToAutoClaim(t, ctx, testData, txAssetHash, maxTimeToClaimReady)
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady, testData.auth[operations.L1].From.String())
	require.NoError(t, err)
	err = waitToAutoClaimTx(t, ctx, testData, deposit, 60*time.Second)
	if err != nil {
		log.Errorf("ERC20 [L1->L2] Doing manual claim")
		err = manualClaimDepositL2(ctx, testData, deposit)
		require.NoError(t, err)
	}
}

func TestERC20TransferL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	log.Infof("Starting test ERC20 L2->L1")

	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	balances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	log.Infof("ERC20 [L1->L2]  balances: %s", balances.String())

	tokenAddr, err := deployToken(ctx, testData.L2Client, testData.auth[operations.L2], testData.cfg.ConnectionConfig.L2BridgeAddr)
	require.NoError(t, err)
	log.Infof("ERC20 [L2->L1] L2: Token Addr: ", tokenAddr.Hex())
	log.Infof("ERC20 [L2->L1] assetERC20L2ToL1")
	txAssetHash := assetERC20L2ToL1(ctx, testData, t, tokenAddr, big.NewInt(133330000000001234))
	log.Infof("ERC20 [L2->L1] ERC20 L2->L1 txAssetHash: %s ", txAssetHash.String())
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady, testData.auth[operations.L2].From.String())
	require.NoError(t, err)
	log.Infof("ERC20 [L2->L1] manualClaimDeposit")
	err = manualClaimDepositL1(ctx, testData, deposit)
	require.NoError(t, err)
}

func deployToken(ctx context.Context, client *utils.Client, auth *bind.TransactOpts, bridgeAddr common.Address) (common.Address, error) {
	tokenAddr, _, err := client.DeployERC20(ctx, "A COIN", "ACO", auth)
	if err != nil {
		return tokenAddr, err
	}
	log.Info("Token Addr: ", tokenAddr.Hex())
	amountTokens := new(big.Int).SetUint64(1000000000000000000)
	err = client.ApproveERC20(ctx, tokenAddr, bridgeAddr, amountTokens, auth)
	if err != nil {
		return tokenAddr, err
	}
	err = client.MintERC20(ctx, tokenAddr, amountTokens, auth)
	if err != nil {
		return tokenAddr, err
	}
	erc20Balance, err := getAccountTokenBalance(ctx, auth, client, tokenAddr, nil)
	if err != nil {
		return tokenAddr, err
	}
	log.Info("ERC20 Balance: ", erc20Balance.String())
	return tokenAddr, nil
}

func getAccountTokenBalance(ctx context.Context, auth *bind.TransactOpts, client *utils.Client, tokenAddr common.Address, account *common.Address) (*big.Int, error) {

	if account == nil {
		account = &auth.From
	}
	erc20Token, err := erc20.NewPol(tokenAddr, client)
	if err != nil {
		return big.NewInt(0), nil
	}
	balance, err := erc20Token.BalanceOf(&bind.CallOpts{Pending: false}, *account)
	if err != nil {
		return big.NewInt(0), nil
	}
	return balance, nil
}

func assetERC20L1ToL2(ctx context.Context, testData *bridge2e2TestData, t *testing.T, tokenAddr common.Address, amount *big.Int) common.Hash {
	l2NetworkId, err := testData.L2Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	auth := testData.auth[operations.L1]
	fmt.Printf("L1->L2 Network ID: %d. Asset ERC20(%s) %+v from L1 -> L2 (addr=%s)\n", l2NetworkId, tokenAddr.String(), amount, auth.From.String())
	txHash, err := assetERC20Generic(ctx, testData.L1Client, l2NetworkId, auth, tokenAddr, amount)
	require.NoError(t, err)
	return txHash
}

func assetERC20L2ToL1(ctx context.Context, testData *bridge2e2TestData, t *testing.T, tokenAddr common.Address, amount *big.Int) common.Hash {
	destNetworkId, err := testData.L1Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	auth := testData.auth[operations.L2]
	fmt.Printf("L2->L1 Network ID: %d. Asset ERC20(%s) %+v from L2 -> L1 (addr=%s)\n", destNetworkId, tokenAddr.String(), amount, auth.From.String())
	txHash, err := assetERC20Generic(ctx, testData.L2Client, destNetworkId, auth, tokenAddr, amount)
	require.NoError(t, err)
	return txHash
}

func assetERC20Generic(ctx context.Context, client *utils.Client, destNetwork uint32, auth *bind.TransactOpts, tokenAddr common.Address, amount *big.Int) (common.Hash, error) {
	destAddr := auth.From
	auth.GasLimit = 500000
	tx, err := client.Bridge.BridgeAsset(auth, destNetwork, destAddr, amount, tokenAddr, true, []byte{})
	if err != nil {
		return common.Hash{}, err
	}
	fmt.Println("Tx: ", tx.Hash().Hex())
	err = ops.WaitTxToBeMined(ctx, client.Client, tx, 60*time.Second)
	return tx.Hash(), err
}
