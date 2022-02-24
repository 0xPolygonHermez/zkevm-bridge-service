package bridgetree

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Set(ctx context.Context, key []byte, value []byte) error
}

// bridgeTreeStorage interface for the Bridge Tree
type bridgeTreeStorage interface {
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
	AddBlock(ctx context.Context, block *etherman.Block) error
}
