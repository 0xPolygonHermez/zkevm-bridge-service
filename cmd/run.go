package main

import (
	"github.com/hermeznetwork/hermez-bridge/config"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/synchronizer"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/urfave/cli/v2"
)

func start(ctx *cli.Context) error {
	configFilePath := ctx.String(flagCfg)
	network := ctx.String(flagNetwork)
	c, err := config.Load(configFilePath, network)
	if err != nil {
		return err
	}
	setupLog(c.Log)

	etherman, l2Etherman, err := newEtherman(*c)
	if err != nil {
		log.Fatal(err)
		return err
	}

	go runL1Synchronizer(c.NetworkConfig, etherman, c.Synchronizer)
	go runL2Synchronizer(c.NetworkConfig, l2Etherman, c.Synchronizer)

	return nil
}

func setupLog(c log.Config) {
	log.Init(c)
}

func newEtherman(c config.Config) (*etherman.ClientEtherMan, *etherman.ClientEtherMan, error) {
	l1Etherman, err := etherman.NewEtherman(c.Etherman, c.NetworkConfig.PoEAddr, c.NetworkConfig.BridgeAddr, c.NetworkConfig.GlobalExitRootManAddr)
	if err != nil {
		return nil, nil, err
	}
	l2Etherman, err := etherman.NewL2Etherman(c.Etherman, c.NetworkConfig.L2BridgeAddr)
	if err != nil {
		return l1Etherman, nil, err
	}
	return l1Etherman, l2Etherman, nil
}

func runL1Synchronizer(networkConfig config.NetworkConfig, etherman *etherman.ClientEtherMan, cfg synchronizer.Config) {
	sy, err := synchronizer.NewSynchronizer(etherman, networkConfig.GenBlockNumber, cfg, false)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}

func runL2Synchronizer(networkConfig config.NetworkConfig, etherman *etherman.ClientEtherMan, cfg synchronizer.Config) {
	sy, err := synchronizer.NewSynchronizer(etherman, 0, cfg, true)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
