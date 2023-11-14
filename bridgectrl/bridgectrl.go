package bridgectrl

import (
	"context"
	"math"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/jackc/pgx/v4"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeController struct
type BridgeController struct {
	exitTrees   []*MerkleTree
	rollupsTree *MerkleTree
	networkIDs  map[uint]uint8
}

// NewBridgeController creates new BridgeController.
func NewBridgeController(ctx context.Context, cfg Config, networks []uint, mtStore interface{}) (*BridgeController, error) {
	var (
		networkIDs = make(map[uint]uint8)
		exitTrees  []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i)
		mt, err := NewMerkleTree(ctx, mtStore.(merkleTreeStore), cfg.Height, network)
		if err != nil {
			return nil, err
		}
		exitTrees = append(exitTrees, mt)
	}
	rollupsTree, err := NewMerkleTree(ctx, mtStore.(merkleTreeStore), cfg.Height, math.MaxInt32)
	if err != nil {
		log.Error("error creating rollupsTree. Error: ", err)
		return nil, err
	}

	return &BridgeController{
		exitTrees:   exitTrees,
		rollupsTree: rollupsTree,
		networkIDs:  networkIDs,
	}, nil
}

func (bt *BridgeController) GetNetworkID(networkID uint) (uint8, error) {
	tID, found := bt.networkIDs[networkID]
	if !found {
		return 0, gerror.ErrNetworkNotRegister
	}
	return tID, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeController) AddDeposit(ctx context.Context, deposit *etherman.Deposit, depositID uint64, dbTx pgx.Tx) error {
	leaf := hashDeposit(deposit)
	tID, err := bt.GetNetworkID(deposit.NetworkID)
	if err != nil {
		return err
	}
	return bt.exitTrees[tID].addLeaf(ctx, depositID, leaf, deposit.DepositCount, dbTx)
}

// ReorgMT reorg the specific merkle tree.
func (bt *BridgeController) ReorgMT(ctx context.Context, depositCount uint, networkID uint, dbTx pgx.Tx) error {
	tID, err := bt.GetNetworkID(networkID)
	if err != nil {
		return err
	}
	return bt.exitTrees[tID].resetLeaf(ctx, depositCount, dbTx)
}

// GetExitRoot returns the dedicated merkle tree's root.
// only use for the test purpose
func (bt *BridgeController) GetExitRoot(ctx context.Context, networkID int, dbTx pgx.Tx) ([]byte, error) {
	return bt.exitTrees[networkID].getRoot(ctx, dbTx)
}

func (bt *BridgeController) AddRollupExitLeaf(ctx context.Context, rollupLeaf etherman.RollupExitLeaf, dbTx pgx.Tx) error {
	err := bt.rollupsTree.addRollupExitLeaf(ctx, rollupLeaf, dbTx)
	if err != nil {
		log.Error("error adding rollupleaf. Error: ", err)
		return err
	}
	return nil
}
