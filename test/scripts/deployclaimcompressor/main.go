package main

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	l2BridgeAddr = "0xCca6ECD73932e49633B9307e1aa0fC174525F424"

	l2AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l2AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l2NetworkURL       = "http://localhost:8123"
)

func main() {
	ctx := context.Background()
	c, err := utils.NewClient(ctx, l2NetworkURL, common.HexToAddress(l2BridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := c.GetSigner(ctx, l2AccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	networkID, err := c.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	claimCompressorAddress, _, _, err := claimcompressor.DeployClaimcompressor(auth, c.Client, common.HexToAddress(l2BridgeAddr), networkID)
	if err != nil {
		log.Fatal("Error deploying claimCompressor contract: ", err)
	}
	log.Info("ClaimCompressosAddress: ", claimCompressorAddress)
}
