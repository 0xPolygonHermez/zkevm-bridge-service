package main

import (
	"os"
	"os/signal"

	"github.com/hermeznetwork/hermez-bridge/bridgetree"
	"github.com/hermeznetwork/hermez-bridge/config"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
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
	err = db.RunMigrations(c.Database)
	if err != nil {
		log.Error(err)
		return err
	}

	etherman, l2Etherman, err := newEtherman(*c)
	if err != nil {
		log.Error(err)
		return err
	}
	storage, err := db.NewStorage(c.Database)
	if err != nil {
		log.Error(err)
		return err
	}

	var brdigeTree *bridgetree.BridgeTree

	if c.BridgeTree.Store == "postgres" {
		pgStorage, err := pgstorage.NewPostgresStorage(pgstorage.Config{
			User:     c.Database.User,
			Password: c.Database.Password,
			Name:     c.Database.Name,
			Host:     c.Database.Host,
			Port:     c.Database.Port,
		})
		if err != nil {
			log.Error(err)
			return err
		}

		brdigeTree, err = bridgetree.NewBridgeTree(c.BridgeTree, []uint{0, 1000}, pgStorage, pgStorage) // need to update the config to init networkIDs
		if err != nil {
			log.Error(err)
			return err
		}
	}

	go runL1Synchronizer(c.NetworkConfig, brdigeTree, etherman, c.Synchronizer, storage)
	go runL2Synchronizer(c.NetworkConfig, brdigeTree, l2Etherman, c.Synchronizer, storage)

	// Wait for an in interrupt.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

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

func runL1Synchronizer(networkConfig config.NetworkConfig, brdigeTree *bridgetree.BridgeTree, etherman *etherman.ClientEtherMan, cfg synchronizer.Config, storage db.Storage) {
	sy, err := synchronizer.NewSynchronizer(storage, brdigeTree, etherman, networkConfig.GenBlockNumber, cfg, false)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}

func runL2Synchronizer(networkConfig config.NetworkConfig, brdigeTree *bridgetree.BridgeTree, etherman *etherman.ClientEtherMan, cfg synchronizer.Config, storage db.Storage) {
	sy, err := synchronizer.NewSynchronizer(storage, brdigeTree, etherman, 0, cfg, true)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
