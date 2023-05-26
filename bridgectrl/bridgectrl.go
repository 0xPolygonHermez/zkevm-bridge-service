package bridgectrl

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/jackc/pgx/v4"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// BridgeController struct
type BridgeController struct {
	exitTrees  []*MerkleTree
	networkIDs map[uint]uint8
}

// NewBridgeController creates new BridgeController.
func NewBridgeController(cfg Config, networks []uint, mtStore interface{}) (*BridgeController, error) {
	var (
		networkIDs = make(map[uint]uint8)
		exitTrees  []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i)
		mt, err := NewMerkleTree(context.TODO(), mtStore.(merkleTreeStore), cfg.Height, network)
		if err != nil {
			return nil, err
		}
		exitTrees = append(exitTrees, mt)
	}

	return &BridgeController{
		exitTrees:  exitTrees,
		networkIDs: networkIDs,
	}, nil
}

func (bt *BridgeController) getNetworkID(networkID uint) (uint8, error) {
	tID, found := bt.networkIDs[networkID]
	if !found {
		return 0, gerror.ErrNetworkNotRegister
	}
	return tID, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeController) AddDeposit(deposit *etherman.Deposit, depositID uint64, dbTx pgx.Tx) error {
	leaf := hashDeposit(deposit)
	tID, err := bt.getNetworkID(deposit.NetworkID)
	if err != nil {
		return err
	}
	return bt.exitTrees[tID].addLeaf(context.TODO(), depositID, leaf, deposit.DepositCount, dbTx)
}

// ReorgMT reorg the specific merkle tree.
func (bt *BridgeController) ReorgMT(depositCount uint, networkID uint, dbTx pgx.Tx) error {
	tID, err := bt.getNetworkID(networkID)
	if err != nil {
		return err
	}
	return bt.exitTrees[tID].resetLeaf(context.TODO(), depositCount, dbTx)
}

// GetExitRoot returns the dedicated merkle tree's root.
// only use for the test purpose
func (bt *BridgeController) GetExitRoot(networkID int, dbTx pgx.Tx) ([]byte, error) {
	return bt.exitTrees[networkID].getRoot(context.TODO(), dbTx)
}
