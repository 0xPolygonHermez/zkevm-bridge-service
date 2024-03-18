package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	zkevmbridgeservice "github.com/0xPolygonHermez/zkevm-bridge-service"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/txcompressor"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/generated_binding/ClaimCompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/synchronizer"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/urfave/cli/v2"
)

func logVersion() {
	log.Infow("Starting application",
		// node version is already logged by default
		"gitRevision", zkevmbridgeservice.GitRev,
		"gitBranch", zkevmbridgeservice.GitBranch,
		"goVersion", runtime.Version(),
		"built", zkevmbridgeservice.BuildDate,
		"os/arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	)
}

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

	networkID, err := l1Etherman.GetNetworkID(ctx.Context)
	log.Infof("main network id: %d", networkID)
	if err != nil {
		log.Error(err)
		return err
	}

	var networkIDs = []uint{networkID}
	for _, client := range l2Ethermans {
		networkID, err := client.GetNetworkID(ctx.Context)
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
	pgstorage := storage.(*pgstorage.PostgresStorage)

	var bridgeController *bridgectrl.BridgeController

	if c.BridgeController.Store == "postgres" {
		bridgeController, err = bridgectrl.NewBridgeController(ctx.Context, c.BridgeController, networkIDs, storage)
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
	rollupID := l1Etherman.GetRollupID()
	bridgeService := server.NewBridgeService(c.BridgeServer, c.BridgeController.Height, networkIDs, apiStorage, rollupID)
	err = server.RunServer(c.BridgeServer, bridgeService)
	if err != nil {
		log.Error(err)
		return err
	}

	log.Debug("trusted sequencer URL ", c.Etherman.L2URLs[0])
	zkEVMClient := client.NewClient(c.Etherman.L2URLs[0])
	chExitRootEvent := make(chan *etherman.GlobalExitRoot)
	chSynced := make(chan uint)
	logVersion()
	go runSynchronizer(ctx.Context, c.NetworkConfig.GenBlockNumber, bridgeController, l1Etherman, c.Synchronizer, storage, zkEVMClient, chExitRootEvent, chSynced)
	for _, client := range l2Ethermans {
		go runSynchronizer(ctx.Context, 0, bridgeController, client, c.Synchronizer, storage, zkEVMClient, chExitRootEvent, chSynced)
	}

	if c.ClaimTxManager.Enabled {
		for i := 0; i < len(c.Etherman.L2URLs); i++ {
			// we should match the orders of L2URLs between etherman and claimtxman
			// since we are using the networkIDs in the same order
			ctx := context.Background()
			client, err := utils.NewClient(ctx, c.Etherman.L2URLs[i], c.NetworkConfig.L2PolygonBridgeAddresses[i])
			if err != nil {
				log.Fatalf("error creating client for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
			}
			nonceCache, err := claimtxman.NewNonceCache(ctx, client)
			if err != nil {
				log.Fatalf("error creating nonceCache for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
			}
			auth, err := client.GetSignerFromKeystore(ctx, c.ClaimTxManager.PrivateKey)
			if err != nil {
				log.Fatalf("error creating signer for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
			}
			var monitorTx ctmtypes.TxMonitorer
			if c.ClaimTxManager.GroupingClaims.Enabled {
				log.Info("ClaimTxManager grouping claims enabled, claimCompressor=%s", c.ClaimCompressorAddress.String())
				claimCompressor, err := ClaimCompressor.NewClaimCompressor(c.ClaimCompressorAddress, client)
				if err != nil {
					log.Fatalf("error creating claim compressor for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
				}
				monitorTx = txcompressor.NewMonitorTxs(ctx, pgstorage, client, c.ClaimTxManager, nonceCache, auth, claimCompressor, utils.NewTimeProviderSystemLocalTime())
			} else {
				monitorTx = claimtxman.NewMonitorTxs(ctx, storage, client, c.ClaimTxManager, nonceCache, auth)
			}

			claimTxManager, err := claimtxman.NewClaimTxManager(ctx, c.ClaimTxManager, chExitRootEvent, chSynced,
				c.Etherman.L2URLs[i], networkIDs[i+1], c.NetworkConfig.L2PolygonBridgeAddresses[i], bridgeService, storage, monitorTx, rollupID, nonceCache, auth)
			if err != nil {
				log.Fatalf("error creating claim tx manager for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
			}
			go claimTxManager.Start()
		}
	} else {
		log.Warn("ClaimTxManager not configured")
		go func() {
			for {
				select {
				case <-chExitRootEvent:
					log.Debug("New GER received")
				case netID := <-chSynced:
					log.Debug("NetworkID synced: ", netID)
				case <-ctx.Context.Done():
					log.Debug("Stopping goroutine that listen new GER updates")
					return
				}
			}
		}()
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
	l1Etherman, err := etherman.NewClient(c.Etherman,
		c.NetworkConfig.PolygonBridgeAddress,
		c.NetworkConfig.PolygonZkEVMGlobalExitRootAddress,
		c.NetworkConfig.PolygonRollupManagerAddress,
		c.NetworkConfig.PolygonZkEvmAddress,
		c.NetworkConfig.ClaimCompressorAddress)
	if err != nil {
		log.Error("L1 etherman error: ", err)
		return nil, nil, err
	}
	if len(c.L2PolygonBridgeAddresses) != len(c.Etherman.L2URLs) {
		log.Fatal("environment configuration error. zkevm bridge addresses and zkevm node urls mismatch")
	}
	var l2Ethermans []*etherman.Client
	for i, addr := range c.L2PolygonBridgeAddresses {
		l2Etherman, err := etherman.NewL2Client(c.Etherman.L2URLs[i], addr)
		if err != nil {
			log.Error("L2 etherman ", i, c.Etherman.L2URLs[i], ", error: ", err)
			return l1Etherman, nil, err
		}
		l2Ethermans = append(l2Ethermans, l2Etherman)
	}
	return l1Etherman, l2Ethermans, nil
}

func runSynchronizer(ctx context.Context, genBlockNumber uint64, brdigeCtrl *bridgectrl.BridgeController, etherman *etherman.Client, cfg synchronizer.Config, storage db.Storage, zkEVMClient *client.Client, chExitRootEvent chan *etherman.GlobalExitRoot, chSynced chan uint) {
	sy, err := synchronizer.NewSynchronizer(ctx, storage, brdigeCtrl, etherman, zkEVMClient, genBlockNumber, chExitRootEvent, chSynced, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
