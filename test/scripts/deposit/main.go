package main

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/log"
)

const (
	l1BridgeAddr = "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"

	l1AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l1NetworkURL       = "http://localhost:8545"

	funds = 90000000000000000
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
	amount := big.NewInt(funds)
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
