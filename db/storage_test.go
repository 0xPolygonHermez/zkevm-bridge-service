package db

import (
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/stretchr/testify/require"
)

func TestExitRootStore(t *testing.T) {
	cfg := pgstorage.NewConfigFromEnv()
	// Init database instance
	err := pgstorage.InitOrReset(cfg)
	require.NoError(t, err)

	storageCfg := Config{
		Database: "postgres",
		Name:     cfg.Name,
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
		MaxConns: 20,
	}
	var networksNumber uint = 2
	_, err = NewStorage(storageCfg, networksNumber)
	require.NoError(t, err)
}
