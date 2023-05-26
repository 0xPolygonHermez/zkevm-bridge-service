package bridgectrl

import (
	"context"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/jackc/pgx/v4"
)

// zeroHashes is the pre-calculated zero hash array
var zeroHashes [][KeyLen]byte

// MerkleTree struct
type MerkleTree struct {
	// store is the database storage to store all node data
	store   merkleTreeStore
	network uint
	// height is the depth of the merkle tree
	height uint8
	// count is the number of deposit
	count uint
	// siblings is the array of sibling of the last leaf added
	siblings [][KeyLen]byte
}

func init() {
	/*
	* We set 64 levels because the height is not known yet. Also it is initialized here to avoid run this
	* function twice (one for mainnetExitTree and another for RollupExitTree).
	* If we receive a height of 32, we would need to use only the first 32 values of the array.
	* If we need more level than 64 for the mt we need to edit this value here and set for example 128.
	 */
	zeroHashes = generateZeroHashes(64) // nolint
}

// NewMerkleTree creates new MerkleTree.
func NewMerkleTree(ctx context.Context, store merkleTreeStore, height uint8, network uint) (*MerkleTree, error) {
	depositCnt, err := store.GetLastDepositCount(ctx, network, nil)
	if err != nil {
		if err != gerror.ErrStorageNotFound {
			return nil, err
		}
		depositCnt = 0
	} else {
		depositCnt++
	}

	mt := &MerkleTree{
		store:   store,
		network: network,
		height:  height,
		count:   depositCnt,
	}
	mt.siblings, err = mt.initSiblings(ctx, nil)

	return mt, err
}

// initSiblings returns the siblings of the node at the given index.
// it is used to initialize the siblings array in the beginning.
func (mt *MerkleTree) initSiblings(ctx context.Context, dbTx pgx.Tx) ([][KeyLen]byte, error) {
	var (
		left     [KeyLen]byte
		siblings [][KeyLen]byte
	)

	if mt.count == 0 {
		for h := 0; h < int(mt.height); h++ {
			copy(left[:], zeroHashes[h][:])
			siblings = append(siblings, left)
		}
		return siblings, nil
	}

	root, err := mt.getRoot(ctx, dbTx)
	if err != nil {
		return nil, err
	}
	// index is the index of the last node
	index := mt.count - 1
	cur := root

	// It starts in height-1 because 0 is the level of the leafs
	for h := int(mt.height - 1); h >= 0; h-- {
		value, err := mt.store.Get(ctx, cur, dbTx)
		if err != nil {
			return nil, fmt.Errorf("height: %d, cur: %v, error: %w", h, cur, err)
		}

		copy(left[:], value[0])
		// we will keep the left sibling of the last node
		siblings = append(siblings, left)

		if index&(1<<h) > 0 {
			cur = value[1]
		} else {
			cur = value[0]
		}
	}

	// We need to invert the siblings to go from leafs to the top
	for st, en := 0, len(siblings)-1; st < en; st, en = st+1, en-1 {
		siblings[st], siblings[en] = siblings[en], siblings[st]
	}

	return siblings, nil
}

func (mt *MerkleTree) addLeaf(ctx context.Context, depositID uint64, leaf [KeyLen]byte, index uint, dbTx pgx.Tx) error {
	if index != mt.count {
		return fmt.Errorf("mismatched deposit count: %d, expected: %d", index, mt.count)
	}
	cur := leaf
	isFilledSubTree := true

	var leaves [][][]byte
	for h := uint8(0); h < mt.height; h++ {
		if index&(1<<h) > 0 {
			var child [KeyLen]byte
			copy(child[:], cur[:])
			parent := Hash(mt.siblings[h], child)
			cur = parent
			leaves = append(leaves, [][]byte{parent[:], mt.siblings[h][:], child[:]})
		} else {
			if isFilledSubTree {
				// we will update the sibling when the sub tree is complete
				copy(mt.siblings[h][:], cur[:])
				// we have a left child in this layer, it means the right child is empty so the sub tree is not completed
				isFilledSubTree = false
			}
			var child [KeyLen]byte
			copy(child[:], cur[:])
			parent := Hash(child, zeroHashes[h])
			cur = parent
			// the sibling of 0 bit should be the zero hash, since we are in the last node of the tree
			leaves = append(leaves, [][]byte{parent[:], child[:], zeroHashes[h][:]})
		}
	}

	err := mt.store.SetRoot(ctx, cur[:], depositID, mt.count, mt.network, dbTx)
	if err != nil {
		return err
	}
	var nodes [][]interface{}
	for _, leaf := range leaves {
		nodes = append(nodes, []interface{}{leaf[0], [][]byte{leaf[1], leaf[2]}, depositID})
	}
	if err := mt.store.BulkSet(ctx, nodes, dbTx); err != nil {
		return err
	}

	mt.count++
	return nil
}

func (mt *MerkleTree) resetLeaf(ctx context.Context, depositCount uint, dbTx pgx.Tx) error {
	var err error
	mt.count = depositCount
	mt.siblings, err = mt.initSiblings(ctx, dbTx)
	return err
}

// this function is used to get the current root of the merkle tree
func (mt *MerkleTree) getRoot(ctx context.Context, dbTx pgx.Tx) ([]byte, error) {
	if mt.count == 0 {
		return zeroHashes[mt.height][:], nil
	}
	return mt.store.GetRoot(ctx, mt.count-1, mt.network, dbTx)
}
