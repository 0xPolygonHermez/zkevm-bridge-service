package db

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
)

type Storage interface {
	GetLastBlock(ctx context.Context) (*etherman.Block, error)
	AddBlock(ctx context.Context, block *etherman.Block) error
}

func NewStorage(cfg Config) (Storage, error) {
	if cfg.Database == "postgres" {
		return pgstorage.NewPostgresStorage(cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)
	}
	return nil, nil
}
