package main

import (
	"os"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	flagURLName        = "url"
	flagBridgeAddrName = "bridgeAddress"
	flagWalletFileName = "walletFile"
	flagPasswordName   = "password"
)

var (
	flagURL = cli.StringFlag{
		Name:     flagURLName,
		Aliases:  []string{"u"},
		Usage:    "Node url",
		Required: true,
	}
	flagBridgeAddr = cli.StringFlag{
		Name:     flagBridgeAddrName,
		Aliases:  []string{"br"},
		Usage:    "Bridge smart contract address",
		Required: true,
	}
	flagWalletFile = cli.StringFlag{
		Name:     flagWalletFileName,
		Aliases:  []string{"f"},
		Usage:    "Wallet file",
		Required: true,
	}
	flagPassword = cli.StringFlag{
		Name:     flagPasswordName,
		Aliases:  []string{"pass"},
		Usage:    "Password",
		Required: true,
	}
)

func main() {
	claimCompDeployer := cli.NewApp()
	claimCompDeployer.Name = "forcedBatchsender"
	claimCompDeployer.Usage = "send forced batch transactions to L1"
	claimCompDeployer.DefaultCommand = "send"
	flags := []cli.Flag{&flagURL, &flagBridgeAddr, &flagWalletFile, &flagPassword}
	claimCompDeployer.Commands = []*cli.Command{
		{
			Name:    "deploy",
			Aliases: []string{},
			Flags:   flags,
			Action:  deploy,
		},
	}

	err := claimCompDeployer.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
func deploy(ctx *cli.Context) error {
	l2NetworkURL := ctx.String(flagURLName)
	l2BridgeAddr := ctx.String(flagBridgeAddrName)
	c, err := utils.NewClient(ctx.Context, l2NetworkURL, common.HexToAddress(l2BridgeAddr))
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	privateKey := types.KeystoreFileConfig{
		Path:     ctx.String(flagWalletFileName),
		Password: ctx.String(flagPasswordName),
	}
	log.Debug(privateKey)
	auth, err := c.GetSignerFromKeystore(ctx.Context, privateKey)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	networkID, err := c.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	log.Debug("auth.From: ", auth.From)
	balance, err := c.Client.BalanceAt(ctx.Context, auth.From, nil)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	log.Debug("balance: ", balance)
	claimCompressorAddress, _, _, err := claimcompressor.DeployClaimcompressor(auth, c.Client, common.HexToAddress(l2BridgeAddr), networkID)
	if err != nil {
		log.Error("Error deploying claimCompressor contract: ", err)
		return err
	}
	log.Info("ClaimCompressosAddress: ", claimCompressorAddress)
	return nil
}
