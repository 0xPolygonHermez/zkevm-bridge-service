package operations

import (
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
)

// RunMockServer runs mock server
func RunMockServer(dbType string, height uint8, networks []uint) (*bridgectrl.BridgeController, StorageInterface, error) {
	if dbType != "postgres" {
		return nil, nil, fmt.Errorf("not registered database")
	}

	dbCfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(dbCfg)
	if err != nil {
		return nil, nil, err
	}

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	if err != nil {
		return nil, nil, err
	}

	btCfg := bridgectrl.Config{
		Height: height,
		Store:  "postgres",
	}

	bt, err := bridgectrl.NewBridgeController(btCfg, networks, store)
	if err != nil {
		return nil, nil, err
	}

	cfg := server.Config{
		GRPCPort:         "9090",
		HTTPPort:         "8080",
		CacheSize:        100000, //nolint:gomnd
		DefaultPageLimit: 25,     //nolint:gomnd
		MaxPageLimit:     100,    //nolint:gomnd
		BridgeVersion:    "v1",
	}
	bridgeService := server.NewBridgeService(cfg, btCfg.Height, networks, store)
	return bt, store, server.RunServer(cfg, bridgeService)
}
