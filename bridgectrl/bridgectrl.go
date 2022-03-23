package bridgectrl

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/etherman"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeController struct
type BridgeController struct {
	exitTrees  []*MerkleTree
	networkIDs map[uint]uint8
	storage    bridgeStorage
}

var (
	contextKeyNetwork = "merkle-tree-network"
)

// NewBridgeController creates new BridgeController.
func NewBridgeController(cfg Config, networks []uint, storage bridgeStorage, store merkleTreeStore) (*BridgeController, error) {
	var (
		networkIDs = make(map[uint]uint8)
		exitTrees  []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i + 1)
		ctx := context.WithValue(context.TODO(), contextKeyNetwork, uint8(i+1)) //nolint
		mt, err := NewMerkleTree(ctx, store, cfg.Height)
		if err != nil {
			return nil, err
		}
		exitTrees = append(exitTrees, mt)
	}

	return &BridgeController{
		exitTrees:  exitTrees,
		networkIDs: networkIDs,
		storage:    storage,
	}, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeController) AddDeposit(deposit *etherman.Deposit) error {
	var ctx context.Context

	leaf := hashDeposit(deposit)

	tID := bt.networkIDs[deposit.OriginalNetwork]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	return bt.exitTrees[tID-1].addLeaf(ctx, leaf)
}

// GetClaim returns claim information to the user.
func (bt *BridgeController) GetClaim(networkID uint, index uint) ([][KeyLen]byte, *etherman.GlobalExitRoot, error) {
	var (
		ctx   context.Context
		proof [][KeyLen]byte
	)

	tID := bt.networkIDs[networkID]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	tID--
	globalExitRoot, err := bt.storage.GetLatestExitRoot(context.TODO())
	if err != nil {
		return proof, nil, err
	}

	proof, err = bt.exitTrees[tID].getSiblings(ctx, index-1, globalExitRoot.ExitRoots[tID])
	if err != nil {
		return proof, nil, err
	}

	return proof, &etherman.GlobalExitRoot{
		GlobalExitRootNum: globalExitRoot.GlobalExitRootNum,
		ExitRoots:         []common.Hash{bt.exitTrees[0].root, bt.exitTrees[1].root},
	}, err
}

// GetMerkleRoot returns claim information to the user.
func (bt *BridgeController) GetMerkleRoot(networkID uint) ([][KeyLen]byte, error) {
	return nil, nil
}

// ReorgMT reorg the specific merkle tree.
func (bt *BridgeController) ReorgMT(depositCount uint, networkID uint) error {
	var ctx context.Context
	tID := bt.networkIDs[networkID]
	ctx = context.WithValue(context.TODO(), contextKeyNetwork, tID) //nolint
	return bt.exitTrees[tID-1].resetLeaf(ctx, depositCount)
}
