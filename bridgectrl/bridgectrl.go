package bridgectrl

import (
	"context"
	"fmt"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
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
func NewBridgeController(cfg Config, networks []uint, bridgeStore interface{}, mtStore interface{}) (*BridgeController, error) {
	var (
		networkIDs = make(map[uint]uint8)
		exitTrees  []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i)
		mt, err := NewMerkleTree(context.TODO(), mtStore.(merkleTreeStore), cfg.Height, uint8(i))
		if err != nil {
			return nil, err
		}
		exitTrees = append(exitTrees, mt)
	}

	return &BridgeController{
		exitTrees:  exitTrees,
		networkIDs: networkIDs,
		storage:    bridgeStore.(bridgeStorage),
	}, nil
}

// AddDeposit adds deposit information to the bridge tree.
func (bt *BridgeController) AddDeposit(deposit *etherman.Deposit) error {
	leaf := hashDeposit(deposit)

	tID, found := bt.networkIDs[deposit.NetworkID]
	if !found {
		return gerror.ErrNetworkNotRegister
	}
	return bt.exitTrees[tID].addLeaf(context.TODO(), leaf)
}

// GetClaim returns claim information to the user.
func (bt *BridgeController) GetClaim(networkID uint, index uint) ([][KeyLen]byte, *etherman.GlobalExitRoot, error) {
	var (
		proof          [][KeyLen]byte
		globalExitRoot *etherman.GlobalExitRoot
		err            error
	)

	tID, found := bt.networkIDs[networkID]
	if !found {
		return proof, nil, gerror.ErrNetworkNotRegister
	}
	ctx := context.TODO()
	localExitRoot, err := bt.storage.GetRoot(ctx, index+1, tID, nil)
	if err != nil {
		return proof, nil, fmt.Errorf("getting the local exit root from the merkle tree failed, error: %v", err)
	}

	globalExitRoot, err = bt.storage.GetGERByLocalExitRoot(ctx, common.BytesToHash(localExitRoot), uint8(tID+1), nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return proof, nil, fmt.Errorf("getting the GER failed, error: %v", err)
		}
		return proof, nil, gerror.ErrDepositNotSynced
	}

	proof, err = bt.exitTrees[tID].getSiblings(ctx, index, globalExitRoot.ExitRoots[tID])
	if err != nil {
		return proof, nil, fmt.Errorf("getting the proof failed, errror: %v, index: %d, root: %v", err, index, globalExitRoot.ExitRoots[tID])
	}

	return proof, globalExitRoot, err
}

// ReorgMT reorg the specific merkle tree.
func (bt *BridgeController) ReorgMT(depositCount uint, networkID uint) error {
	tID, found := bt.networkIDs[networkID]
	if !found {
		return gerror.ErrNetworkNotRegister
	}
	return bt.exitTrees[tID].resetLeaf(context.TODO(), depositCount)
}

// MockAddDeposit adds deposit information to the bridge tree with globalExitRoot.
func (bt *BridgeController) MockAddDeposit(deposit *etherman.Deposit) error {
	err := bt.AddDeposit(deposit)
	if err != nil {
		return err
	}
	return bt.storage.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
		BlockNumber: 0,
		Timestamp:   time.Now(),
		ExitRoots:   []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
		BlockID:     deposit.BlockID,
	}, nil)
}

// GetTokenWrapped returns tokenWrapped information.
func (bt *BridgeController) GetTokenWrapped(origNetwork uint, origTokenAddr common.Address) (*etherman.TokenWrapped, error) {
	tokenWrapped, err := bt.storage.GetTokenWrapped(context.Background(), origNetwork, origTokenAddr, nil)
	if err != nil {
		return nil, err
	}
	return tokenWrapped, err
}
