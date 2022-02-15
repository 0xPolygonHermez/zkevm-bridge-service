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

	etherman, err := newEtherman(*c)
	if err != nil {
		log.Fatal(err)
		return err
	}

	go runSynchronizer(c.NetworkConfig, etherman, c.Synchronizer)

	return nil
}

func setupLog(c log.Config) {
	log.Init(c)
}

func newEtherman(c config.Config) (*etherman.ClientEtherMan, error) {
	etherman, err := etherman.NewEtherman(c.Etherman, c.NetworkConfig.BridgeAddr, c.NetworkConfig.GlobalExitRootManAddr)
	if err != nil {
		return nil, err
	}
	return etherman, nil
}

func runSynchronizer(networkConfig config.NetworkConfig, etherman *etherman.ClientEtherMan, cfg synchronizer.Config) {
	sy, err := synchronizer.NewSynchronizer(etherman, networkConfig.GenBlockNumber, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
