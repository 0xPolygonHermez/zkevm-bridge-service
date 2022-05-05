package main

import (
	"context"
	"strings"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
)
const (
	l1BridgeAddr      = "0x1E0D0574616274aBbcB7D5b37630e22DaEf52e1f"
	l2BridgeAddr      = "0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988"

	l1AccHexAddress    = ""
	l1AccHexPrivateKey = ""
	l1NetworkURL       = "http://localhost:8545/" 
)
var tokenAddr = common.Address{}


func main() {
	ctx := context.Background()
	client, err := ethclient.Dial(l1NetworkURL)
	if err != nil {
		log.Error(err)
		return
	}
	br, err := bridge.NewBridge(common.HexToAddress(l1BridgeAddr), client)
	if err != nil {
		log.Error(err)
		return
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l1AccHexPrivateKey, "0x"))
	if err != nil {
		log.Error(err)
		return
	}
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Error(err)
		return
	}
	amount := big.NewInt(100000000000000000)
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	var destNetwork uint32 = 1
	destAddr := common.HexToAddress(l1AccHexAddress)
	log.Info("Sending bridge tx...")
	tx, err := br.Bridge(auth, tokenAddr, amount, destNetwork, destAddr)
	if err != nil {
		log.Error(err)
		return
	}
	const txTimeout = 30 * time.Second
	err = operations.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("Success! txHash: ", tx.Hash())
}