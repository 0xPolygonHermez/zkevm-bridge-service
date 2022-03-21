package bridgetree

import (
	"context"

	"github.com/hermeznetwork/hermez-bridge/etherman"
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
	GetLatestExitRoot(ctx context.Context) (*etherman.GlobalExitRoot, error)
}

// BridgeServiceStorage interface for the Bridge Service.
type BridgeServiceStorage interface {
	GetClaims(ctx context.Context, destNetwork uint, limit uint, offset uint) ([]*etherman.Claim, error)
	GetDeposits(ctx context.Context, depositCount uint, origNetwork uint, limit uint) ([]*etherman.Deposit, error)
}
