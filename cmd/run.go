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
	"github.com/0xPolygonHermez/zkevm-bridge-service/config"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
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
	logVersion()
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

	networkID := l1Etherman.GetNetworkID()
	log.Infof("main network id: %d", networkID)

	var networkIDs = []uint{networkID}
	for _, client := range l2Ethermans {
		networkID := client.GetNetworkID()
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
	bridgeService := server.NewBridgeService(c.BridgeServer, c.BridgeController.Height, networkIDs, apiStorage)
	err = server.RunServer(c.BridgeServer, bridgeService)
	if err != nil {
		log.Error(err)
		return err
	}

	var chsExitRootEvent []chan *etherman.GlobalExitRoot
	var chsSyncedL2 []chan uint
	for i, l2EthermanClient := range l2Ethermans {
		log.Debug("trusted sequencer URL ", c.Etherman.L2URLs[i])
		zkEVMClient := client.NewClient(c.Etherman.L2URLs[i])
		chExitRootEventL2 := make(chan *etherman.GlobalExitRoot)
		chSyncedL2 := make(chan uint)
		chsExitRootEvent = append(chsExitRootEvent, chExitRootEventL2)
		chsSyncedL2 = append(chsSyncedL2, chSyncedL2)
		go runSynchronizer(ctx.Context, 0, bridgeController, l2EthermanClient, c.Synchronizer, storage, zkEVMClient, chExitRootEventL2, nil, chSyncedL2, []uint{})
	}
	chSynced := make(chan uint)
	go runSynchronizer(ctx.Context, c.NetworkConfig.GenBlockNumber, bridgeController, l1Etherman, c.Synchronizer, storage, nil, nil, chsExitRootEvent, chSynced, networkIDs)
	go func() {
		for {
			select {
			case netID := <-chSynced:
				log.Debug("NetworkID synced: ", netID)
			case <-ctx.Done():
				log.Debug("Stopping goroutine that listen new GER updates")
				return
			}
		}
	}()
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
			rollupID := l2Ethermans[i].GetNetworkID() // RollupID == networkID
			claimTxManager, err := claimtxman.NewClaimTxManager(ctx, c.ClaimTxManager, chsExitRootEvent[i], chsSyncedL2[i],
				c.Etherman.L2URLs[i], networkIDs[i+1], c.NetworkConfig.L2PolygonBridgeAddresses[i], bridgeService, storage, rollupID, l2Ethermans[i], nonceCache, auth)
			if err != nil {
				log.Fatalf("error creating claim tx manager for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
			}
			go claimTxManager.Start()
		}
	} else {
		log.Warn("ClaimTxManager not configured")
		for i := range chsExitRootEvent {
			monitorChannel(ctx.Context, chsExitRootEvent[i], chsSyncedL2[i], networkIDs[i+1], storage)
		}
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

func monitorChannel(ctx context.Context, chExitRootEvent chan *etherman.GlobalExitRoot, chSynced chan uint, networkID uint, storage db.Storage) {
	go func() {
		for {
			select {
			case ger := <-chExitRootEvent:
				log.Debug("New GER received")
				s := storage.(claimtxman.StorageInterface)
				dbTx, err := s.BeginDBTransaction(ctx)
				if err != nil {
					log.Error("networkId: %d, error creating dbTx. Error: %v", networkID, err)
					continue
				}
				if ger.BlockID != 0 { // L2 exit root is updated
					if err := s.UpdateL2DepositsStatus(ctx, ger.ExitRoots[1][:], networkID, networkID, dbTx); err != nil {
						log.Errorf("networkId: %d, error updating L2DepositsStatus. Error: %v", networkID, err)
						rollbackErr := s.Rollback(ctx, dbTx)
						if rollbackErr != nil {
							log.Errorf("networkId: %d, error rolling back state. RollbackErr: %s, err: %s", networkID, rollbackErr.Error(), err.Error())
						}
						continue
					}
				} else { // L1 exit root is updated in the trusted state
					_, err := s.UpdateL1DepositsStatus(ctx, ger.ExitRoots[0][:], networkID, dbTx)
					if err != nil {
						log.Errorf("networkId: %d, error getting and updating L1DepositsStatus. Error: %v", networkID, err)
						rollbackErr := s.Rollback(ctx, dbTx)
						if rollbackErr != nil {
							log.Errorf("networkId: %d, error rolling back state. RollbackErr: %s, err: %s", networkID, rollbackErr.Error(), err.Error())
						}
						continue
					}
				}
				err = s.Commit(ctx, dbTx)
				if err != nil {
					log.Errorf("networkId: %d, error committing dbTx. Err: %v", networkID, err)
					rollbackErr := s.Rollback(ctx, dbTx)
					if rollbackErr != nil {
						log.Errorf("networkId: %d, error rolling back state. RollbackErr: %s, err: %s", networkID, rollbackErr.Error(), err.Error())
					}
				}
			case netID := <-chSynced:
				log.Debug("NetworkID synced: ", netID)
			case <-ctx.Done():
				log.Debug("Stopping goroutine that listen new GER updates")
				return
			}
		}
	}()
}
func newEthermans(c *config.Config) (*etherman.Client, []*etherman.Client, error) {
	l1Etherman, err := etherman.NewClient(c.Etherman,
		c.NetworkConfig.PolygonBridgeAddress,
		c.NetworkConfig.PolygonZkEVMGlobalExitRootAddress,
		c.NetworkConfig.PolygonRollupManagerAddress)
	if err != nil {
		log.Error("L1 etherman error: ", err)
		return nil, nil, err
	}
	if len(c.L2PolygonBridgeAddresses) != len(c.Etherman.L2URLs) {
		log.Fatal("environment configuration error. zkevm bridge addresses and zkevm node urls mismatch")
	}
	var l2Ethermans []*etherman.Client
	for i, addr := range c.L2PolygonBridgeAddresses {
		l2Etherman, err := etherman.NewL2Client(c.Etherman.L2URLs[i], addr, c.NetworkConfig.L2ClaimCompressorAddress)
		if err != nil {
			log.Error("L2 etherman ", i, c.Etherman.L2URLs[i], ", error: ", err)
			return l1Etherman, nil, err
		}
		l2Ethermans = append(l2Ethermans, l2Etherman)
	}
	return l1Etherman, l2Ethermans, nil
}

func runSynchronizer(ctx context.Context, genBlockNumber uint64, brdigeCtrl *bridgectrl.BridgeController, etherman *etherman.Client, cfg synchronizer.Config, storage db.Storage, zkEVMClient *client.Client, chExitRootEventL2 chan *etherman.GlobalExitRoot, chsExitRootEvent []chan *etherman.GlobalExitRoot, chSynced chan uint, allNetworkIDs []uint) {
	sy, err := synchronizer.NewSynchronizer(ctx, storage, brdigeCtrl, etherman, zkEVMClient, genBlockNumber, chExitRootEventL2, chsExitRootEvent, chSynced, cfg, allNetworkIDs)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}
