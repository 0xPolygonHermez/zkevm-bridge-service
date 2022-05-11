package main

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/matic"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/proofofefficiency"
	"github.com/hermeznetwork/hermez-core/log"
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
	client, auth, err := initClientConnection(ctx)
	if err != nil {
		log.Fatal("error connectin to the the L1 node. Error: ", err)
	}
	poeAddr := common.HexToAddress(poeAddress)
	poe, err := proofofefficiency.NewProofofefficiency(poeAddr, client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	maticAmount, err := poe.CalculateSequencerCollateral(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("Error getting collateral amount from smc: ", err)
	}
	matic, err := matic.NewMatic(common.HexToAddress(maticTokenAddr), client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	txApprove, err := matic.Approve(auth, poeAddr, maticAmount)
	if err != nil {
		log.Fatal("Error approving matics: ", err)
	}
	//wait to process approve
	const txETHTransferTimeout = 20 * time.Second
	err = operations.WaitTxToBeMined(ctx, client, txApprove.Hash(), txETHTransferTimeout)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	tx, err := poe.SendBatch(auth, []byte{}, maticAmount)
	if err != nil {
		log.Fatal("Error sending the batch: ", err)
	}

	// Wait eth transfer to be mined
	log.Infof("Waiting tx to be mined")
	err = operations.WaitTxToBeMined(ctx, client, tx.Hash(), txETHTransferTimeout)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Batch succefully sent!")
}

func initClientConnection(ctx context.Context) (*ethclient.Client, *bind.TransactOpts, error) {
	// Eth client
	client, err := ethclient.Dial(l1NetworkURL)
	if err != nil {
		return nil, nil, err
	}
	// Get network chain id
	log.Infof("Getting chainID")
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err
	}
	// Preparing l1 acc info
	log.Infof("Preparing authorization")
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l1AccHexPrivateKey, "0x"))
	if err != nil {
		return nil, nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, err
	}
	return client, auth, nil
}
