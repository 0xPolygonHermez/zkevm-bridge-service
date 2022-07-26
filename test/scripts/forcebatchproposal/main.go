package main

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/proofofefficiency"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	l1NetworkURL   = "http://localhost:8545"
	poeAddress     = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	maticTokenAddr = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
)

func main() {
	ctx := context.Background()
	// Eth client
	log.Infof("Connecting to l1")
	client, err := utils.NewClient(ctx, l1NetworkURL)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := client.GetSigner(ctx, l1AccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	poeAddr := common.HexToAddress(poeAddress)
	poe, err := proofofefficiency.NewProofofefficiency(poeAddr, client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	maticAmount, err := poe.CalculateForceProverFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("Error getting collateral amount from smc: ", err)
	}
	err = client.ApproveERC20(ctx, common.HexToAddress(maticTokenAddr), poeAddr, maticAmount, auth)
	if err != nil {
		log.Fatal("Error approving matics: ", err)
	}
	tx, err := poe.SequenceBatches(auth, nil)
	if err != nil {
		log.Fatal("Error sending the batch: ", err)
	}

	// Wait eth transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txETHTransferTimeout = 60 * time.Second
	err = utils.WaitTxToBeMined(ctx, client.Client, tx.Hash(), txETHTransferTimeout)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Batch succefully sent!")
}
