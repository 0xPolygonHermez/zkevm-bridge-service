package main

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	bridgeAddr = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"

	networkURL = "http://localhost:8123"

	depositCnt      = 585
	originalNetwork = 0
)

func main() {
	ctx := context.Background()
	client, err := utils.NewClient(ctx, networkURL, common.HexToAddress(bridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}

	isClaimed, err := client.Bridge.IsClaimed(&bind.CallOpts{Pending: false}, depositCnt, originalNetwork)
	if err != nil {
		log.Fatal("error sending deposit. Error: ", err)
	}
	log.Info("IsCLaimed: ", isClaimed)
}
