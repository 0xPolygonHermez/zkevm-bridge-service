package main

import (
	"os"
	"os/signal"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/coinmiddleware"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/estimatetime"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/metrics"
	"github.com/0xPolygonHermez/zkevm-bridge-service/pushtask"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/sentinel"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server/iprestriction"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/urfave/cli/v2"
)

func runAPI(ctx *cli.Context) error {
	return startServer(ctx, withAPI())
}

func runTask(ctx *cli.Context) error {
	return startServer(ctx, withTasks())
}

func runPushTask(ctx *cli.Context) error {
	return startServer(ctx, withPushTasks())
}

func runAll(ctx *cli.Context) error {
	return startServer(ctx, withAPI(), withTasks(), withPushTasks())
}

type runOption struct {
	runAPI       bool
	runTasks     bool
	runPushTasks bool
}

type runOptionFunc func(opt *runOption)

func withAPI() runOptionFunc {
	return func(opt *runOption) {
		opt.runAPI = true
	}
}

func withTasks() runOptionFunc {
	return func(opt *runOption) {
		opt.runTasks = true
	}
}

func withPushTasks() runOptionFunc {
	return func(opt *runOption) {
		opt.runPushTasks = true
	}
}

var _ = start // This is to ignore the "unused" error when linting

func startServer(ctx *cli.Context, opts ...runOptionFunc) error {
	opt := &runOption{}
	for _, f := range opts {
		f(opt)
	}

	c, err := initCommon(ctx)
	if err != nil {
		return err
	}

	err = db.RunMigrations(c.SyncDB)
	if err != nil {
		log.Error(err)
		return err
	}

	utils.InitUSDCLxLyMapping(c.BusinessConfig.USDCContractAddresses, c.BusinessConfig.USDCTokenAddresses)

	l1ChainId := c.Etherman.L1ChainId
	l2ChainIds := c.Etherman.L2ChainIds
	var chainIDs = []uint{l1ChainId}
	chainIDs = append(chainIDs, l2ChainIds...)
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
	for _, cl := range l2Ethermans {
		networkID, err := cl.GetNetworkID(ctx.Context)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Infof("l2 network id: %d", networkID)
		networkIDs = append(networkIDs, networkID)
		utils.InitRollupNetworkId(networkID)
	}

	l2NodeClients := make([]*utils.Client, len(c.Etherman.L2URLs))
	l2Auths := make([]*bind.TransactOpts, len(c.Etherman.L2URLs))
	for i := range c.Etherman.L2URLs {
		nodeClient, err := utils.NewClient(ctx.Context, c.Etherman.L2URLs[i], c.NetworkConfig.L2PolygonBridgeAddresses[i])
		if err != nil {
			log.Error(err)
			return err
		}
		auth, err := nodeClient.GetSignerFromKeystore(ctx.Context, c.ClaimTxManager.PrivateKey)
		if err != nil {
			log.Error(err)
			return err
		}
		l2NodeClients[i] = nodeClient
		l2Auths[i] = auth
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

	redisStorage, err := redisstorage.NewRedisStorage(c.BridgeServer.Redis)
	if err != nil {
		log.Error(err)
		return err
	}

	err = localcache.InitDefaultCache(apiStorage)
	if err != nil {
		log.Error(err)
		return err
	}

	err = estimatetime.InitDefaultCalculator(apiStorage)
	if err != nil {
		log.Error(err)
		return err
	}

	var messagePushProducer messagepush.KafkaProducer
	if c.MessagePushProducer.Enabled {
		log.Infof("message push producer's switch is open, so init producer!")
		messagePushProducer, err = messagepush.NewKafkaProducer(c.MessagePushProducer)
		if err != nil {
			log.Error(err)
			return err
		}
		defer func() {
			err := messagePushProducer.Close()
			if err != nil {
				log.Errorf("close kafka producer error: %v", err)
			}
		}()
	}

	// Start metrics
	if c.Metrics.Enabled {
		go metrics.StartMetricsHttpServer(c.Metrics)
	}

	// Initialize chainId manager
	utils.InitChainIdManager(networkIDs, chainIDs)

	rollupID := l1Etherman.GetRollupID()
	bridgeService := server.NewBridgeService(c.BridgeServer, c.BridgeController.Height, networkIDs, l2NodeClients, l2Auths, apiStorage, rollupID).
		WithRedisStorage(redisStorage).WithMainCoinsCache(localcache.GetDefaultCache()).WithMessagePushProducer(messagePushProducer)

	// Initialize inner chain id conf
	utils.InnitOkInnerChainIdMapper(c.BusinessConfig)

	// ---------- Run API ----------
	if opt.runAPI {
		// Init sentinel
		if c.Apollo.Enabled {
			err = sentinel.InitApolloDataSource(c.Apollo)
		} else {
			err = sentinel.InitFileDataSource(c.BridgeServer.SentinelConfigFilePath)
		}
		if err != nil {
			log.Infof("init sentinel error[%v]; ignored and proceed with no sentinel config", err)
		}
		server.RegisterNacos(c.NacosConfig)
		iprestriction.InitClient(c.IPRestriction)

		err = server.RunServer(c.BridgeServer, bridgeService)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// ---------- Run push tasks ----------
	if opt.runPushTasks {
		// Initialize the push task for L1 block num change
		l1BlockNumTask, err := pushtask.NewL1BlockNumTask(c.Etherman.L1URL, apiStorage, redisStorage, messagePushProducer, rollupID)
		if err != nil {
			log.Error(err)
			return err
		}
		go l1BlockNumTask.Start(ctx.Context)

		// Initialize the push task for sync l2 commit batch
		syncCommitBatchTask, err := pushtask.NewCommittedBatchHandler(c.Etherman.L2URLs[0], apiStorage, redisStorage, messagePushProducer, rollupID)
		if err != nil {
			log.Error(err)
			return err
		}
		go syncCommitBatchTask.Start(ctx.Context)

		// Initialize the push task for sync verify batch
		syncVerifyBatchTask, err := pushtask.NewVerifiedBatchHandler(c.Etherman.L2URLs[0], redisStorage)
		if err != nil {
			log.Error(err)
			return err
		}
		go syncVerifyBatchTask.Start(ctx.Context)
	}

	// ---------- Run synchronizer tasks ----------
	if opt.runTasks {
		log.Debug("trusted sequencer URL ", c.Etherman.L2URLs[0])
		zkEVMClient := client.NewClient(c.Etherman.L2URLs[0])
		chExitRootEvent := make(chan *etherman.GlobalExitRoot)
		chSynced := make(chan uint)
		go runSynchronizer(ctx.Context, c.NetworkConfig.GenBlockNumber, bridgeController, l1Etherman, c.Synchronizer, storage, zkEVMClient, chExitRootEvent, chSynced, messagePushProducer, redisStorage)
		for _, cl := range l2Ethermans {
			go runSynchronizer(ctx.Context, 0, bridgeController, cl, c.Synchronizer, storage, zkEVMClient, chExitRootEvent, chSynced, messagePushProducer, redisStorage)
		}

		if c.ClaimTxManager.Enabled {
			for i := 0; i < len(c.Etherman.L2URLs); i++ {
				// we should match the orders of L2URLs between etherman and claimtxman
				// since we are using the networkIDs in the same order
				claimTxManager, err := claimtxman.NewClaimTxManager(c.ClaimTxManager, chExitRootEvent, chSynced, c.Etherman.L2URLs[i], networkIDs[i+1], c.NetworkConfig.L2PolygonBridgeAddresses[i], bridgeService, storage, messagePushProducer, redisStorage, rollupID)
				if err != nil {
					log.Fatalf("error creating claim tx manager for L2 %s. Error: %v", c.Etherman.L2URLs[i], err)
				}
				go claimTxManager.StartXLayer()
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

		if len(c.CoinKafkaConsumer.Brokers) > 0 {
			// Start the coin middleware kafka consumer
			log.Debugf("start initializing kafka consumer...")
			coinKafkaConsumer, err := coinmiddleware.NewKafkaConsumer(c.CoinKafkaConsumer, redisStorage)
			if err != nil {
				log.Error(err)
				return err
			}
			log.Debugf("finish initializing kafka consumer")
			go coinKafkaConsumer.Start(ctx.Context)
			defer func() {
				err := coinKafkaConsumer.Close()
				if err != nil {
					log.Errorf("close kafka consumer error: %v", err)
				}
			}()
		}
	}

	// Wait for an in interrupt.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	return nil
}

func initCommon(ctx *cli.Context) (*config.Config, error) {
	configFilePath := ctx.String(flagCfg)
	network := ctx.String(flagNetwork)

	c, err := config.Load(configFilePath, network)
	if err != nil {
		return nil, err
	}
	setupLog(c.Log)
	apolloconfig.SetLogger()
	return c, nil
}
