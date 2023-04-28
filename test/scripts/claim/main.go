package main

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	clientUtils "github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

const (
	l2BridgeAddr = "0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0"

	l2AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l2AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l2NetworkURL       = "http://localhost:8123"
	bridgeURL          = "http://localhost:8080"

	mtHeight = 32
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

	// Get Claim data
	cfg := clientUtils.Config{
		L1NodeURL:    l2NetworkURL,
		L2NodeURL:    l2NetworkURL,
		BridgeURL:    bridgeURL,
		L2BridgeAddr: common.HexToAddress(l2BridgeAddr),
	}
	client, err := clientUtils.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	deposits, _, err := client.GetBridges(l2AccHexAddress, 0, 10) //nolint
	if err != nil {
		log.Fatal("Error: ", err)
	}
	bridgeData := deposits[0]
	proof, err := client.GetMerkleProof(deposits[0].NetworkId, deposits[0].DepositCnt)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Debug("bridge: ", bridgeData)
	log.Debug("mainnetExitRoot: ", proof.MainExitRoot)
	log.Debug("rollupExitRoot: ", proof.RollupExitRoot)

	var smt [mtHeight][32]byte
	for i := 0; i < len(proof.MerkleProof); i++ {
		log.Debug("smt: ", proof.MerkleProof[i])
		smt[i] = common.HexToHash(proof.MerkleProof[i])
	}
	globalExitRoot := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}
	log.Info("Sending claim tx...")
	err = c.SendClaim(ctx, bridgeData, smt, globalExitRoot, auth)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("Success!")
	balance, err := c.Client.BalanceAt(ctx, common.HexToAddress(l2AccHexAddress), nil)
	if err != nil {
		log.Fatal("error getting balance: ", err)
	}
	log.Info("L2 balance: ", balance)
}
