package bridgetree

import (
	"context"
)

// merkleTreeStore interface for the Merkle Tree
type merkleTreeStore interface {
	Get(ctx context.Context, key []byte) ([][]byte, error)
	Set(ctx context.Context, key []byte, value [][]byte) error
	GetMTRoot(ctx context.Context, root []byte) (uint, error)
	SetMTRoot(ctx context.Context, index uint, root []byte) error
}

// bridgeTreeStorage interface for the Bridge Tree
type bridgeTreeStorage interface {
	GetLastGlobalExitRoot(ctx context.Context) (uint64, []byte, [][]byte, error)
	SetGlobalExitRoot(ctx context.Context, index uint64, globalRoot []byte, roots [][]byte) error
}
