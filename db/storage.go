package db

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
)

// Storage interface
type Storage struct {
	*pgstorage.PostgresStorage
}

// NewStorage creates a new Storage
func NewStorage(cfg Config, networksNumber uint) (*Storage, error) {
	if cfg.Database == "postgres" {
		pg, err := pgstorage.NewPostgresStorage(pgstorage.Config{
			Name:     cfg.Name,
			User:     cfg.User,
			Password: cfg.Password,
			Host:     cfg.Host,
			Port:     cfg.Port,
			MaxConns: cfg.MaxConns,
		}, networksNumber)
		return &Storage{pg}, err
	}
	return nil, gerror.ErrStorageNotRegister
}

// RunMigrations will execute pending migrations if needed to keep
// the database updated with the latest changes
func RunMigrations(cfg Config) error {
	config := pgstorage.Config{
		Name:     cfg.Name,
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}
	return pgstorage.RunMigrationsUp(config)
}
