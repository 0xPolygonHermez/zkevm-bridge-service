package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	clientUtils "github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

const (
	bridgeAddr = "0x10B65c586f795aF3eCCEe594fE4E38E1F059F780"
	okbAddress = "0x82109a709138A2953C720D3d775168717b668ba6"
	ethAddress = "0x82109a709138A2953C720D3d775168717b668ba6"

	accHexAddress    = "0x2ECF31eCe36ccaC2d3222A303b1409233ECBB225"
	accHexPrivateKey = "0xde3ca643a52f5543e84ba984c4419ff40dbabd0e483c31c1d09fee8168d68e38"

	l1NetworkURL = "http://localhost:8545"
	l2NetworkURL = "http://localhost:8123"

	l1Network uint32 = 0
	l2Network uint32 = 1

	funds     = 100000000
	bridgeURL = "http://localhost:8080"
	mtHeight  = 32

	usage = "Usage: ./bridge <type> [0: L1->L2 OKB; 1: L1->L2 ETH; 2:L2->L1 OKB; 3: L2->L1 ETH]"
)

func main() {
	args := os.Args
	if len(args) != 2 { //nolint
		fmt.Println(usage)
		return
	}
	bridgeType := args[1]
	if bridgeType == "0" { //nolint
		bridgeL1ToL2OKB()
	} else if bridgeType == "1" { //nolint
		bridgeL1ToL2ETH()
	} else if bridgeType == "2" { //nolint
		bridgeL2ToL1OKB()
	} else if bridgeType == "3" { //nolint
		bridgeL2ToL1ETH()
	} else {
		fmt.Println(usage)
	}
}

func bridgeL1ToL2OKB() {
	log.Info("Start L1->L2 OKB ...")
	ctx := context.Background()
	bridgeAddress := common.HexToAddress(bridgeAddr)
	okbAddress := common.HexToAddress(okbAddress)
	userAddress := common.HexToAddress(accHexAddress)
	amount := big.NewInt(funds)

	client, err := utils.NewClient(ctx, l1NetworkURL, bridgeAddress) //nolint
	userAuth, err := client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// approve OKB
	log.Info("Approve OKB to bridge ...")
	err = client.ApproveERC20(ctx, okbAddress, bridgeAddress, amount, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// deposit OKB
	log.Info("Deposit OKB...")
	err = client.SendBridgeAsset(ctx, okbAddress, amount, l2Network, &userAddress, []byte{}, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	time.Sleep(10 * time.Second) //nolint

	log.Info("Success! L1->L2 OKB")
}

func bridgeL1ToL2ETH() {
	// deposit ETH
	log.Info("Start L1->L2 ETH ...")
	ctx := context.Background()
	bridgeAddress := common.HexToAddress(bridgeAddr)
	userAddress := common.HexToAddress(accHexAddress)
	amount := big.NewInt(funds)

	client, err := utils.NewClient(ctx, l1NetworkURL, bridgeAddress) //nolint
	userAuth, err := client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// deposit ETH
	log.Info("Deposit ETH...")
	err = client.SendBridgeAsset(ctx, common.Address{}, amount, l2Network, &userAddress, []byte{}, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	time.Sleep(10 * time.Second) //nolint

	log.Info("Success! L1->L2 ETH")
}

func bridgeL2ToL1OKB() {
	// deposit OKB
	log.Info("Start L2->L1 OKB ...")
	ctx := context.Background()
	bridgeAddress := common.HexToAddress(bridgeAddr)
	userAddress := common.HexToAddress(accHexAddress)
	amount := big.NewInt(funds)

	client, err := utils.NewClient(ctx, l2NetworkURL, bridgeAddress) //nolint
	userAuth, err := client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// deposit ETH
	log.Info("Deposit OKB...")
	err = client.SendBridgeAsset(ctx, common.Address{}, amount, l1Network, &userAddress, []byte{}, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	time.Sleep(30 * time.Second) //nolint
	claimL1()

	log.Info("Success! L1->L2 ETH")
}

func bridgeL2ToL1ETH() {
	// deposit ETH
	ctx := context.Background()
	bridgeAddress := common.HexToAddress(bridgeAddr)
	ethAddress := common.HexToAddress(ethAddress)
	userAddress := common.HexToAddress(accHexAddress)
	amount := big.NewInt(funds)

	client, err := utils.NewClient(ctx, l2NetworkURL, bridgeAddress) //nolint
	userAuth, err := client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// approve ETH
	log.Info("Approve ETH to bridge ...")
	err = client.ApproveERC20(ctx, ethAddress, bridgeAddress, amount, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// deposit OKB
	log.Info("Deposit OKB...")
	err = client.SendBridgeAsset(ctx, ethAddress, amount, l1Network, &userAddress, []byte{}, userAuth)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	time.Sleep(30 * time.Second) //nolint
	claimL1()

	log.Info("Success! L1->L2 OKB")
}

func claimL1() {
	ctx := context.Background()
	c, err := utils.NewClient(ctx, l1NetworkURL, common.HexToAddress(bridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := c.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

	// Get Claim data
	cfg := clientUtils.Config{
		L1NodeURL:    l1NetworkURL,
		L2NodeURL:    l1NetworkURL,
		BridgeURL:    bridgeURL,
		L2BridgeAddr: common.HexToAddress(bridgeAddr),
		L1BridgeAddr: common.HexToAddress(bridgeAddr),
	}
	client, err := clientUtils.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	deposits, _, err := client.GetBridges(accHexAddress, 0, 10) //nolint
	if err != nil {
		log.Fatal("Error: ", err)
	}
	bridgeData := deposits[0]
	proof, err := client.GetMerkleProof(deposits[0].NetworkId, deposits[0].DepositCnt)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("bridge: ", bridgeData)
	log.Info("mainnetExitRoot: ", proof.MainExitRoot)
	log.Info("rollupExitRoot: ", proof.RollupExitRoot)

	var smt [mtHeight][32]byte
	for i := 0; i < len(proof.MerkleProof); i++ {
		log.Info("smt: ", proof.MerkleProof[i])
		smt[i] = common.HexToHash(proof.MerkleProof[i])
	}
	globalExitRoot := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}
	log.Info("Sending claim tx...")
	_, err = c.SendClaim(ctx, utils.PbToEthermanDeposit(bridgeData), smt, globalExitRoot, auth)
	if err != nil {
		log.Fatal("error: ", err)
	}
	log.Info("Claim Success!")
}
