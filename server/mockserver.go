package server

import (
	"github.com/hermeznetwork/hermez-bridge/bridgectrl"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
)

// RunMockServer runs mock server
func RunMockServer() error {
	dbCfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(dbCfg)
	if err != nil {
		return err
	}

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	if err != nil {
		return err
	}

	bt, err := bridgectrl.MockBridgeCtrl(store)
	if err != nil {
		return err
	}

	cfg := Config{
		GRPCPort: "9090",
		HTTPPort: "8080",
	}

	return RunServer(store, bt, cfg)
}
