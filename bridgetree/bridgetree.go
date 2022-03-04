package bridgetree

import (
	"context"
	"math/big"

	"github.com/hermeznetwork/hermez-bridge/etherman"
	"golang.org/x/crypto/sha3"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeTree struct
type BridgeTree struct {
	exitRootTrees     []*MerkleTree
	globalExitRoot    [KeyLen]byte
	globalExitRootNum uint64
	networkIDs        map[uint64]uint8
	storage           bridgeTreeStorage
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
		exitRootTrees:     exitRootTrees,
		globalExitRoot:    zeroHashes[cfg.Height+1],
		globalExitRootNum: 0,
		networkIDs:        networkIDs,
		storage:           storage,
	}, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeTree) AddDeposit(deposit *etherman.Deposit) error {
	var (
		ctx   context.Context
		roots [][]byte
	)

	leaf := hashDeposit(deposit)

	tID := bt.networkIDs[uint64(deposit.OriginNetwork)]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	err := bt.exitRootTrees[tID-1].addLeaf(ctx, leaf)
	if err != nil {
		return err
	}

	for _, mt := range bt.exitRootTrees {
		roots = append(roots, mt.root[:])
	}
	bt.globalExitRootNum++
	hash := sha3.NewLegacyKeccak256()
	for _, d := range roots {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(bt.globalExitRoot[:], hash.Sum(nil))
	err = bt.storage.SetGlobalExitRoot(context.TODO(), bt.globalExitRootNum, bt.globalExitRoot[:], roots)
	if err != nil {
		return err
	}

	return bt.storage.AddDeposit(ctx, deposit)
}

// GetClaim returns claim information to the user.
func (bt *BridgeTree) GetClaim(networkID uint, index uint, mtProoves [][KeyLen]byte) (*etherman.Deposit, *etherman.GlobalExitRoot, error) {
	deposit, err := bt.storage.GetDeposit(context.TODO(), index, networkID)
	if err != nil {
		return nil, nil, err
	}

	var (
		ctx         context.Context
		prooves     [][KeyLen]byte
		networkRoot [KeyLen]byte
	)

	tID := bt.networkIDs[uint64(networkID)]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	globalExitRootNum, _, roots, err := bt.storage.GetLastGlobalExitRoot(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	copy(networkRoot[:], roots[tID])
	prooves, err = bt.exitRootTrees[tID].getSiblings(ctx, index-1, networkRoot)
	if err != nil {
		return nil, nil, err
	}
	copy(mtProoves, prooves)

	return deposit, &etherman.GlobalExitRoot{
		GlobalExitRootNum: big.NewInt(int64(globalExitRootNum)),
		MainnetExitRoot:   bt.exitRootTrees[0].root,
		RollupExitRoot:    bt.exitRootTrees[1].root}, err
}
