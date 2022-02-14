package bridgetree

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

// Store interface for the Merkle Tree
type Store interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Set(ctx context.Context, key []byte, value []byte) error
}

// Storage interface for the Bridge Tree
type Storage interface {
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
}
