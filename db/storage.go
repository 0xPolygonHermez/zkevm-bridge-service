package db

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

type Storage interface {
	GetLastBlock(ctx context.Context) (*etherman.Block, error)
}
