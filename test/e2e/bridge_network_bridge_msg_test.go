//go:build e2e_real_network_msg
// +build e2e_real_network_msg

package e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/BridgeMessageReceiver"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/PingReceiver"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func newMetadata(starting byte) []byte {
	var res []byte
	for i := 0; i < 32; i++ {
		res = append(res, starting+byte(i))
	}
	return res
}
func TestMessageTransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	log.Infof("Starting test bridge_message L1 -> L2")
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	balances, err := getcurrentBalance(ctx, testData)
	require.NoError(t, err)
	log.Infof("MSG [L1->L2]  balances: %s", balances.String())
	log.Infof("MSG [L1->L2]  deploying contract to L2 to recieve message")
	contractDeployedAdrr, pingContract, blockDeployed, err := deployBridgeMessagePingReceiver(ctx, testData.auth[operations.L2], testData.L2Client, testData.cfg.ConnectionConfig.L2BridgeAddr)
	//contractDeployedAdrr, pingContract, blockDeployed, err := deployBridgeMessagePingReceiver2(ctx, testData.cfg.TestL2AddrPrivate, testData.L2Client, testData.cfg.ConnectionConfig.L2BridgeAddr)

	require.NoError(t, err)
	log.Infof("MSG [L1->L2] Setting to PingReceiver Contract the sender of the message, if not is refused")
	log.Infof("MSG [L1->L2]  deployed contract to recieve message: %s", contractDeployedAdrr.String())
	tx, err := pingContract.SetSender(testData.auth[operations.L2], testData.auth[operations.L1].From)
	require.NoError(t, err)
	log.Infof("MSG [L1->L2] SetSender(%s) tx: %s", testData.cfg.ConnectionConfig.L2BridgeAddr, tx.Hash().String())
	log.Infof("MSG [L1->L2] send BridgeMessage to L1 bridge metadata:")
	amount := new(big.Int).SetUint64(1000000000000004444)

	txAssetHash := assetMsgL1ToL2(ctx, testData, t, contractDeployedAdrr, amount, newMetadata(1))
	log.Infof("MSG [L1->L2] txAssetHash: %s", txAssetHash.String())
	log.Infof("MSG [L1->L2] waitDepositToBeReadyToClaim")
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady, contractDeployedAdrr.String())
	require.NoError(t, err)
	log.Infof("MSG [L2->L1] manualClaimDeposit")
	err = manualClaimDepositL2(ctx, testData, deposit)
	require.NoError(t, err)
	checkThatPingReceivedIsEmitted(ctx, t, blockDeployed, pingContract)
}

func TestMessageTransferL2toL1(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	log.Infof("Starting test bridge_message L2 -> L1")
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	log.Infof("MSG [L2->L1]  deploying contract to L1 to recieve message")
	v, err := testData.L1Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	log.Infof("MSG [L2->L1] L1 Network ID: %d", v)
	//contractDeployedAdrr, _, err := deployBridgeMessageReceiver(ctx, testData.auth[operations.L2], testData.L2Client)
	contractDeployedAdrr, pingContract, blockDeployed, err := deployBridgeMessagePingReceiver(ctx, testData.auth[operations.L1], testData.L1Client, testData.cfg.ConnectionConfig.L1BridgeAddr)
	require.NoError(t, err)
	log.Infof("MSG [L2->L1]  deployed contract to recieve message: %s", contractDeployedAdrr.String())
	log.Infof("MSG [L2->L1] Setting to PingReceiver Contract the sender of the message, if not is refused")
	tx, err := pingContract.SetSender(testData.auth[operations.L1], testData.auth[operations.L2].From)
	require.NoError(t, err)
	log.Infof("MSG [L2->L1] SetSender(%s) tx: %s", testData.auth[operations.L2].From, tx.Hash().String())

	log.Infof("MSG [L2->L1] send BridgeMessage to L1 bridge metadata: ")
	amount := new(big.Int).SetUint64(1000000000000005555)
	txAssetHash := assetMsgL2ToL1(ctx, testData, t, contractDeployedAdrr, amount, newMetadata(4))
	log.Infof("MSG [L2->L1] txAssetHash: %s", txAssetHash.String())
	log.Infof("MSG [L2->L1] waitDepositToBeReadyToClaim")
	deposit, err := waitDepositToBeReadyToClaim(ctx, testData, txAssetHash, maxTimeToClaimReady, contractDeployedAdrr.String())
	require.NoError(t, err)
	log.Infof("MSG [L2->L1] manualClaimDeposit")
	err = manualClaimDepositL1(ctx, testData, deposit)
	require.NoError(t, err)
	checkThatPingReceivedIsEmitted(ctx, t, blockDeployed, pingContract)
}

func checkThatPingReceivedIsEmitted(ctx context.Context, t *testing.T, blockDeployed uint64, pingContract *PingReceiver.PingReceiver) {
	query := bind.FilterOpts{Start: blockDeployed, End: nil, Context: ctx}
	eventIterator, err := pingContract.FilterPingReceived(&query)
	if err != nil {
		log.Errorf("Error filtering events: %s", err.Error())
	}
	if !eventIterator.Next() {
		log.Errorf("Error getting event, something fails")
		require.Fail(t, "Fail to retrieve the event PingReceived, something fails")
	}
}

func assetMsgL1ToL2(ctx context.Context, testData *bridge2e2TestData, t *testing.T, destAddr common.Address, amount *big.Int, metadata []byte) common.Hash {
	destNetworkId, err := testData.L2Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	auth := testData.auth[operations.L1]
	log.Infof("MSG [L1->L2] L2 Network ID: %d. BridgeMessage(destContract: %s) metadata:%+v from L1 -> L2 (from addr=%s)\n", destNetworkId, destAddr.String(), metadata, auth.From.String())
	txHash, err := assetMsgGeneric(ctx, testData.L1Client, destNetworkId, auth, destAddr, amount, metadata)
	require.NoError(t, err)
	return txHash
}

func assetMsgL2ToL1(ctx context.Context, testData *bridge2e2TestData, t *testing.T, destAddr common.Address, amount *big.Int, metadata []byte) common.Hash {
	destNetworkId, err := testData.L1Client.Bridge.NetworkID(nil)
	require.NoError(t, err)
	auth := testData.auth[operations.L2]
	log.Infof("MSG [L2->L1] L1 Network ID: %d. L2->BridgeMessage(destContract: %s) metadata:%+v from L2 -> L1 (from addr=%s)\n", destNetworkId, destAddr.String(), metadata, auth.From.String())
	txHash, err := assetMsgGeneric(ctx, testData.L2Client, destNetworkId, auth, destAddr, amount, metadata)
	require.NoError(t, err)
	return txHash
}

func assetMsgGeneric(ctx context.Context, client *utils.Client, destNetwork uint32, auth *bind.TransactOpts, destAddr common.Address, amount *big.Int, metadata []byte) (common.Hash, error) {
	auth.GasLimit = 500000
	auth.Value = amount
	tx, err := client.Bridge.BridgeMessage(auth, destNetwork, destAddr, true, metadata)
	if err != nil {
		return common.Hash{}, err
	}
	fmt.Println("Tx: ", tx.Hash().Hex())
	err = ops.WaitTxToBeMined(ctx, client.Client, tx, 60*time.Second)
	return tx.Hash(), err
}
func checkEventOfBridgeMessageReceiver(t *testing.T, pingContract *PingReceiver.PingReceiver, blockDeployed uint64, timeNow uint32) {

}

func deployBridgeMessageReceiver(ctx context.Context, auth *bind.TransactOpts, backend *utils.Client) (common.Address, *BridgeMessageReceiver.BridgeMessageReceiver, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, contractBinded, err := BridgeMessageReceiver.DeployBridgeMessageReceiver(auth, backend)
	if err != nil {
		return common.Address{}, nil, err
	}
	err = ops.WaitTxToBeMined(ctx, backend, tx, txMinedTimeoutLimit)

	return addr, contractBinded, err
}
func deployBridgeMessagePingReceiver(ctx context.Context, auth *bind.TransactOpts, backend *utils.Client, bridgeAddr common.Address) (common.Address, *PingReceiver.PingReceiver, uint64, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	log.Infof("Deploying PingReceiver contract using ath.From: %s", auth.From.String())
	addr, tx, contractBinded, err := PingReceiver.DeployPingReceiver(auth, backend, bridgeAddr)
	if err != nil {
		log.Errorf("Error deploying PingReceiver contract: %s", err.Error())
		return common.Address{}, nil, 0, err
	}
	err = ops.WaitTxToBeMined(ctx, backend, tx, txMinedTimeoutLimit)
	if err != nil {
		log.Errorf("Error deploying PingReceiver contract:WaitTxToBeMined: %s", err.Error())
		return common.Address{}, nil, 0, err
	}
	receipt, err := backend.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		log.Errorf("Error deploying PingReceiver contract:TransactionReceipt: %s", err.Error())
		return common.Address{}, nil, 0, err
	}
	return addr, contractBinded, receipt.BlockNumber.Uint64(), nil
}
