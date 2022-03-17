package bridgetree

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/etherman"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeTree struct
type BridgeTree struct {
	exitRootTrees []*MerkleTree
	networkIDs    map[uint64]uint8
	storage       bridgeTreeStorage
}

var (
	contextKeyNetwork = "merkle-tree-network"
)

// NewBridgeTree creates new BridgeTree.
func NewBridgeTree(cfg Config, networks []uint64, storage bridgeTreeStorage, store merkleTreeStore) (*BridgeTree, error) {
	var (
		networkIDs    = make(map[uint64]uint8)
		exitRootTrees []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i + 1)
		ctx := context.WithValue(context.TODO(), contextKeyNetwork, uint8(i+1)) //nolint
		mt, err := NewMerkleTree(ctx, store, cfg.Height)
		if err != nil {
			return nil, err
		}
		exitRootTrees = append(exitRootTrees, mt)
	}

	return &BridgeTree{
		exitRootTrees: exitRootTrees,
		networkIDs:    networkIDs,
		storage:       storage,
	}, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeTree) AddDeposit(deposit *etherman.Deposit) error {
	var ctx context.Context

	leaf := hashDeposit(deposit)

	tID := bt.networkIDs[uint64(deposit.OriginalNetwork)]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	return bt.exitRootTrees[tID-1].addLeaf(ctx, leaf)
}

// GetClaim returns claim information to the user.
func (bt *BridgeTree) GetClaim(networkID uint, index uint) ([][KeyLen]byte, *etherman.GlobalExitRoot, error) {
	var (
		ctx   context.Context
		proof [][KeyLen]byte
	)

	tID := bt.networkIDs[uint64(networkID)]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	globalExitRoot, err := bt.storage.GetLatestExitRoot(context.TODO())
	if err != nil {
		return proof, nil, err
	}

	proof, err = bt.exitRootTrees[tID].getSiblings(ctx, index-1, globalExitRoot.ExitRoots[tID])
	if err != nil {
		return proof, nil, err
	}

	return proof, &etherman.GlobalExitRoot{
		GlobalExitRootNum: globalExitRoot.GlobalExitRootNum,
		ExitRoots:         []common.Hash{bt.exitRootTrees[0].root, bt.exitRootTrees[1].root},
	}, err
}
