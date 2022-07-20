package main

import (
	"context"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/synchronizer"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
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

	storage, err := db.NewStorage(c.Database, uint(len(networkIDs)))
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
			MaxConns: c.Database.MaxConns,
		}, uint(len(networkIDs)))
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

func newEthermans(c config.Config) (*etherman.Client, []*etherman.Client, error) {
	auth, err := newAuthFromKeystore(c.Etherman.PrivateKeyPath, c.Etherman.PrivateKeyPassword, c.NetworkConfig.L1ChainID)
	if err != nil {
		return nil, nil, err
	}
	l1Etherman, err := etherman.NewClient(c.Etherman, auth, c.NetworkConfig.PoEAddr, c.NetworkConfig.BridgeAddr, c.NetworkConfig.GlobalExitRootManAddr)
	if err != nil {
		return nil, nil, err
	}
	if len(c.L2BridgeAddrs) != len(c.Etherman.L2URLs) {
		log.Fatal("Environment configuration error. L2 bridge addresses and l2 hermezCore urls mismatch")
	}
	var l2Ethermans []*etherman.Client
	// for i, addr := range c.L2BridgeAddrs {
	// 	l2Etherman, err := etherman.NewClient(c.Etherman.L2URLs[i], addr)
	// 	if err != nil {
	// 		return l1Etherman, nil, err
	// 	}
	// 	l2Ethermans = append(l2Ethermans, l2Etherman)
	// }
	return l1Etherman, l2Ethermans, nil
}

func runSynchronizer(genBlockNumber uint64, brdigeCtrl *bridgectrl.BridgeController, etherman *etherman.Client, cfg synchronizer.Config, storage interface{}) {
	sy, err := synchronizer.NewSynchronizer(storage, brdigeCtrl, etherman, genBlockNumber, cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := sy.Sync(); err != nil {
		log.Fatal(err)
	}
}

func newKeyFromKeystore(path, password string) (*keystore.Key, error) {
	if path == "" && password == "" {
		return nil, nil
	}
	keystoreEncrypted, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keystoreEncrypted, password)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func newAuthFromKeystore(path, password string, chainID uint64) (*bind.TransactOpts, error) {
	key, err := newKeyFromKeystore(path, password)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("addr: ", key.Address.Hex())
	auth, err := bind.NewKeyedTransactorWithChainID(key.PrivateKey, new(big.Int).SetUint64(chainID))
	if err != nil {
		log.Fatal(err)
	}
	return auth, nil
}
