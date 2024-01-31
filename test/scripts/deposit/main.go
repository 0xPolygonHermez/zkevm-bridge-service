package main

import (
	"context"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

const (
	l1BridgeAddr = "0x10B65c586f795aF3eCCEe594fE4E38E1F059F780"

	l1AccHexAddress    = "0x2ecf31ece36ccac2d3222a303b1409233ecbb225"
	l1AccHexPrivateKey = "0xde3ca643a52f5543e84ba984c4419ff40dbabd0e483c31c1d09fee8168d68e38"
	l1NetworkURL       = "http://localhost:8545"

	funds              = 90000000000000000 // nolint
	destNetwork uint32 = 1
)

var tokenAddr = common.Address{}

func main() {
	ctx := context.Background()
	client, err := utils.NewClient(ctx, l1NetworkURL, common.HexToAddress(l1BridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := client.GetSigner(ctx, l1AccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	amount := big.NewInt(funds)
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	destAddr := common.HexToAddress(l1AccHexAddress)
	log.Info("Sending bridge tx...")
	err = client.SendBridgeAsset(ctx, tokenAddr, amount, destNetwork, &destAddr, []byte{}, auth)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Success!")
}
