package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/synchronizer"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/0xPolygonHermez/zkevm-node/log"
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
	err = db.RunMigrations(c.SyncDB)
	if err != nil {
		log.Error(err)
		return err
	}

	l1Etherman, l2Ethermans, err := newEthermans(c)
	if err != nil {
		log.Error(err)
		return err
	}

	networkID, err := l1Etherman.GetNetworkID(context.Background())
	log.Infof("main network id: %d", networkID)
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
		log.Infof("l2 network id: %d", networkID)
		networkIDs = append(networkIDs, networkID)
	}

	storage, err := db.NewStorage(c.SyncDB)
	if err != nil {
		log.Error(err)
		return err
	}

	var bridgeController *bridgectrl.BridgeController

	if c.BridgeController.Store == "postgres" {
		bridgeController, err = bridgectrl.NewBridgeController(c.BridgeController, networkIDs, storage)
		if err != nil {
			log.Error(err)
			return err
		}
	} else {
		log.Error(gerror.ErrStorageNotRegister)
		return gerror.ErrStorageNotRegister
	}

	apiStorage, err := db.NewStorage(c.BridgeServer.DB)
	if err != nil {
		log.Error(err)
		return err
	}
	bridgeService := server.NewBridgeService(c.BridgeServer, c.BridgeController.Height, networkIDs, apiStorage)
	err = server.RunServer(c.BridgeServer, bridgeService)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debug("trusted sequencer URL ", c.Etherman.L2URLs[0])
	zkEVMClient := client.NewClient(c.Etherman.L2URLs[0])
	chExitRootEvent := make(chan *etherman.GlobalExitRoot)
	go runSynchronizer(c.NetworkConfig.GenBlockNumber, bridgeController, l1Etherman, c.Synchronizer, storage, zkEVMClient, chExitRootEvent)
	for _, client := range l2Ethermans {
		go runSynchronizer(0, bridgeController, client, c.Synchronizer, storage, zkEVMClient, chExitRootEvent)
	}

	for i := 0; i < len(c.Etherman.L2URLs); i++ {
		// we should match the orders of L2URLs between etherman and claimtxman
		// since we are using the networkIDs in the same order
		claimTxManager, err := claimtxman.NewClaimTxManager(c.ClaimTxManager, chExitRootEvent, c.Etherman.L2URLs[i], networkIDs[i+1], c.NetworkConfig.L2BridgeAddrs[i], bridgeService, storage)
		if err != nil {
			log.Fatalf("error creating claim tx manager for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
		}
		go claimTxManager.Start()
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

func newEthermans(c *config.Config) (*etherman.Client, []*etherman.Client, error) {
	l1Etherman, err := etherman.NewClient(c.Etherman, c.NetworkConfig.PoEAddr, c.NetworkConfig.BridgeAddr, c.NetworkConfig.GlobalExitRootManAddr)
	if err != nil {
		return nil, nil, err
	}
	if c.Etherman.L2URLs[0] == "" {
		log.Debug("getting trusted sequencer URL from smc")
		c.Etherman.L2URLs[0], err = l1Etherman.GetTrustedSequencerURL()
		if err != nil {
			log.Fatal("error getting trusted sequencer URI. Error: %v", err)
		}
	}
	if len(c.L2BridgeAddrs) != len(c.Etherman.L2URLs) {
		log.Fatal("environment configuration error. zkevm bridge addresses and zkevm node urls mismatch")
	}
	var l2Ethermans []*etherman.Client
	for i, addr := range c.L2BridgeAddrs {
		l2Etherman, err := etherman.NewL2Client(c.Etherman.L2URLs[i], addr)
		if err != nil {
			return l1Etherman, nil, err
		}
		l2Ethermans = append(l2Ethermans, l2Etherman)
	}
	return l1Etherman, l2Ethermans, nil
}

func runSynchronizer(genBlockNumber uint64, brdigeCtrl *bridgectrl.BridgeController, etherman *etherman.Client, cfg synchronizer.Config, storage db.Storage, zkEVMClient *client.Client, chExitRootEvent chan *etherman.GlobalExitRoot) {
	sy, err := synchronizer.NewSynchronizer(storage, brdigeCtrl, etherman, zkEVMClient, genBlockNumber, chExitRootEvent, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
