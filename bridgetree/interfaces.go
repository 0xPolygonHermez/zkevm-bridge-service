package bridgetree

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

// store interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Set(ctx context.Context, key []byte, value []byte) error
}

// Storage interface for the Bridge Tree
type Storage interface {
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
}
