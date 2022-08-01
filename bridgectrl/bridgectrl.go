package bridgectrl

import (
	"context"
	"math/big"

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
func NewBridgeController(cfg Config, networks []uint, storage bridgeStorage, store merkleTreeStore) (*BridgeController, error) {
	var (
		networkIDs = make(map[uint]uint8)
		exitTrees  []*MerkleTree
	)

	for i, network := range networks {
		networkIDs[network] = uint8(i)
		mt, err := NewMerkleTree(context.TODO(), store, cfg.Height, uint8(i))
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
		globalExitRoot, err = bt.storage.GetLatestL1SyncedExitRoot(ctx, nil)
	} else {
		globalExitRoot, err = bt.storage.GetLatestTrustedExitRoot(ctx, nil)
	}

	if err != nil {
		return proof, nil, err
	}
	depositCnt, err := bt.storage.GetDepositCountByRoot(ctx, globalExitRoot.ExitRoots[tID][:], tID, nil)
	if err != nil {
		return proof, nil, err
	}
	if depositCnt < index {
		return proof, nil, gerror.ErrDepositNotSynced
	}

	proof, err = bt.exitTrees[tID].getSiblings(ctx, index, globalExitRoot.ExitRoots[tID])
	if err != nil {
		return proof, nil, err
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

// CheckExitRoot checks if each exitRoot is synchronized exactly
func (bt *BridgeController) CheckExitRoot(globalExitRoot etherman.GlobalExitRoot) error {
	for i, exitRoot := range globalExitRoot.ExitRoots {
		_, err := bt.storage.GetDepositCountByRoot(context.TODO(), exitRoot[:], uint8(i), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// MockAddDeposit adds deposit information to the bridge tree with globalExitRoot.
func (bt *BridgeController) MockAddDeposit(deposit *etherman.Deposit) error {
	err := bt.AddDeposit(deposit)
	if err != nil {
		return err
	}
	return bt.storage.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
		BlockNumber:       0,
		GlobalExitRootNum: big.NewInt(int64(deposit.DepositCount)),
		ExitRoots:         []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
		BlockID:           deposit.BlockID,
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
