package main

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	l1NetworkURL                   = "http://localhost:8545"
	polygonZkEVMAddressHex         = "0x8dAF17A20c9DBA35f005b6324F493785D239719d"
	polygonRollupManagerAddressHex = "0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"
	polTokenAddressHex             = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec

	maxSequenceTimestamp = 1
	initSequencedBatch   = 1
)

func main() {
	ctx := context.Background()
	// Eth client
	log.Infof("Connecting to l1")
	client, err := utils.NewClient(ctx, l1NetworkURL, common.Address{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := client.GetSigner(ctx, l1AccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	polygonZkEVMAddress := common.HexToAddress(polygonZkEVMAddressHex)
	polygonZkEVM, err := polygonzkevm.NewPolygonzkevm(polygonZkEVMAddress, client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	polygonRollupManagerAddress := common.HexToAddress(polygonRollupManagerAddressHex)
	polygonRollupManager, err := polygonrollupmanager.NewPolygonrollupmanager(polygonRollupManagerAddress, client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	polAmount, err := polygonRollupManager.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("Error getting collateral amount from smc: ", err)
	}
	err = client.ApproveERC20(ctx, common.HexToAddress(polTokenAddressHex), polygonZkEVMAddress, polAmount, auth)
	if err != nil {
		log.Fatal("Error approving pol: ", err)
	}
	tx, err := polygonZkEVM.SequenceBatches(auth, nil, maxSequenceTimestamp, initSequencedBatch, auth.From)
	if err != nil {
		log.Fatal("Error sending the batch: ", err)
	}

	// Wait eth transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txETHTransferTimeout = 60 * time.Second
	err = utils.WaitTxToBeMined(ctx, client.Client, tx, txETHTransferTimeout)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Batch succefully sent!")
}
