//go:build e2e_real_network_msg
// +build e2e_real_network_msg

package e2e

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestMessageTransferL1toL2(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	log.Infof("Starting test bridge_message L1 -> L2")
	ctx := context.TODO()
	testData, err := NewBridge2e2TestData(ctx, nil)
	require.NoError(t, err)
	log.Infof("[L1->L2]  bridge_message")
	destAdrr, err := deployBridgeMessageReceiver(ctx, testData.auth[operations.L1, testData.L1Client])
	require.NoError(t, err)
	tokenAddr, err := deployToken(ctx, testData.L1Client, testData.auth[operations.L1], testData.cfg.ConnectionConfig.L1BridgeAddr)
	log.Infof("[L1->L2] L1: Token Addr: ", tokenAddr.Hex())
	require.NoError(t, err)
	// Do Asset L1 -> L2
	amount := big.NewInt(143210000000001234) // 0.14321 ETH
	log.Infof("[L1->L2] assetERC20L2ToL1")
	txAssetHash := assetERC20L2ToL1(ctx, testData, t, tokenAddr, amount)
	log.Infof("ERC20 L1->L2 txAssetHash: %s \n", txAssetHash.String())
	log.Infof("[L1->L2] waitToAutoClaim")
	waitToAutoClaim(t, ctx, testData, txAssetHash, maxTimeToClaimReady)
}

func deployBridgeMessageReceiver(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *BridgeMessageReceiver, error) {
	parsed, err := BridgeMessageReceiverMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(BridgeMessageReceiverBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &BridgeMessageReceiver{BridgeMessageReceiverCaller: BridgeMessageReceiverCaller{contract: contract}, BridgeMessageReceiverTransactor: BridgeMessageReceiverTransactor{contract: contract}, BridgeMessageReceiverFilterer: BridgeMessageReceiverFilterer{contract: contract}}, nil
}
