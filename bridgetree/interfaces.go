package bridgetree

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte) ([][]byte, uint, error)
	Set(ctx context.Context, key []byte, value [][]byte, depositCount uint) error
	ResetMT(ctx context.Context, depositCount uint) error
	GetRoot(ctx context.Context, depositCount uint) ([]byte, error)
}

// bridgeTreeStorage interface for the Bridge Tree
type bridgeTreeStorage interface {
	GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
}
