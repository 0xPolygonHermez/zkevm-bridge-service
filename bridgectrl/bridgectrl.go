package bridgectrl

import (
	"context"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
	// MainNetworkID is the chain ID for the main network
	MainNetworkID = uint(0)
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
	if networkID == MainNetworkID {
		globalExitRoot, err = bt.storage.GetLatestTrustedExitRoot(ctx, nil)
	} else {
		globalExitRoot, err = bt.storage.GetLatestL1SyncedExitRoot(ctx, nil)
	}
	if err != nil {
		return proof, nil, fmt.Errorf("getting the last GER failed, error: %v", err)
	}
	depositCnt, err := bt.storage.GetDepositCountByRoot(ctx, globalExitRoot.ExitRoots[tID][:], tID, nil)
	if err != nil {
		return proof, nil, fmt.Errorf("getting deposit count from the MT root failed, error: %v, root: %v, network: %d", err, globalExitRoot.ExitRoots[tID][:], tID)
	}
	if depositCnt < index {
		return proof, nil, gerror.ErrDepositNotSynced
	}

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

// GetTokenWrapped returns tokenWrapped information.
func (bt *BridgeController) GetTokenWrapped(origNetwork uint, origTokenAddr common.Address) (*etherman.TokenWrapped, error) {
	tokenWrapped, err := bt.storage.GetTokenWrapped(context.Background(), origNetwork, origTokenAddr, nil)
	if err != nil {
		return nil, err
	}
	return tokenWrapped, err
}

// GetExitRoot returns the dedicated merkle tree's root
func (bt *BridgeController) GetExitRoot(networkID int) []byte {
	return bt.exitTrees[networkID].root[:]
}
