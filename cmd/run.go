package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/hermeznetwork/hermez-bridge/bridgectrl"
	"github.com/hermeznetwork/hermez-bridge/config"
	"github.com/hermeznetwork/hermez-bridge/db"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/gerror"
	"github.com/hermeznetwork/hermez-bridge/server"
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

	etherman, l2Ethermans, err := newEthermans(*c)
	if err != nil {
		log.Error(err)
		return err
	}

	networkID, err := etherman.GetNetworkID(context.Background())
	if err != nil {
		log.Error(err)
		return err
	}

	var networkIDs = []uint{networkID}

	for _, client := range l2Ethermans {
		networkID, err := client.GetNetworkID(context.Background())
		if err != nil {
			log.Error(err)
			return err
		}

		networkIDs = append(networkIDs, networkID)
	}

	storage, err := db.NewStorage(c.Database)
	if err != nil {
		log.Error(err)
		return err
	}

	var bridgeController *bridgectrl.BridgeController

	if c.BridgeController.Store == "postgres" {
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

		bridgeController, err = bridgectrl.NewBridgeController(c.BridgeController, networkIDs, pgStorage, pgStorage)
		if err != nil {
			log.Error(err)
			return err
		}

		err = server.RunServer(pgStorage, bridgeController, c.BridgeServer)
		if err != nil {
			log.Error(err)
			return err
		}
	} else {
		log.Error(gerror.ErrStorageNotRegister)
		return gerror.ErrStorageNotRegister
	}

	go runSynchronizer(c.NetworkConfig.GenBlockNumber, bridgeController, etherman, c.Synchronizer, storage)
	for _, client := range l2Ethermans {
		go runSynchronizer(0, bridgeController, client, c.Synchronizer, storage)
	}

	// Wait for an in interrupt.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	return nil
}

func setupLog(c log.Config) {
	log.Init(c)
}

func newEthermans(c config.Config) (*etherman.ClientEtherMan, []*etherman.ClientEtherMan, error) {
	l1Etherman, err := etherman.NewEtherman(c.Etherman, c.NetworkConfig.PoEAddr, c.NetworkConfig.BridgeAddr, c.NetworkConfig.GlobalExitRootManAddr)
	if err != nil {
		return nil, nil, err
	}
	if len(c.L2BridgeAddrs) != len(c.Etherman.L2URLs) {
		log.Fatal("Environment configuration error. L2 bridge addresses and l2 hermezCore urls mismatch")
	}
	var l2Ethermans []*etherman.ClientEtherMan
	for i, addr := range c.L2BridgeAddrs {
		l2Etherman, err := etherman.NewL2Etherman(c.Etherman.L2URLs[i], addr)
		if err != nil {
			return l1Etherman, nil, err
		}
		l2Ethermans = append(l2Ethermans, l2Etherman)
	}
	return l1Etherman, l2Ethermans, nil
}

func runSynchronizer(genBlockNumber uint64, brdigeCtrl *bridgectrl.BridgeController, etherman *etherman.ClientEtherMan, cfg synchronizer.Config, storage db.Storage) {
	sy, err := synchronizer.NewSynchronizer(storage, brdigeCtrl, etherman, genBlockNumber, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
