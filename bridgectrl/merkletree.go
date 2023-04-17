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
	network uint8
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
func NewMerkleTree(ctx context.Context, store merkleTreeStore, height, network uint8, isZeroHashesAdded bool) (*MerkleTree, error) {
	depositCnt, err := store.GetLastDepositCount(ctx, network, nil)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			rootID, err1 := store.SetRoot(ctx, zeroHashes[height][:], 0, network, nil)
			if err1 != nil {
				return nil, err
			}
			if !isZeroHashesAdded {
				for h := uint8(0); h < height; h++ {
					// h+1 is the position of the parent node and h is the position of the values of the children nodes. As all the nodes of the same nodes has the same values
					// we can only store the info ones
					err := store.Set(ctx, zeroHashes[h+1][:], [][]byte{zeroHashes[h][:], zeroHashes[h][:]}, rootID, nil)
					if err != nil {
						return nil, err
					}
				}
			}
		} else {
			return nil, err
		}
	}

	mt := &MerkleTree{
		store:   store,
		network: network,
		height:  height,
		count:   depositCnt,
	}
	root, err := mt.getRoot(ctx, nil)
	if err != nil {
		return nil, err
	}
	mt.siblings, err = mt.initSiblings(ctx, root, nil)

	return mt, err
}

// initSiblings returns the siblings of the node at the given index.
// it is used to initialize the siblings array in the beginning.
func (mt *MerkleTree) initSiblings(ctx context.Context, root []byte, dbTx pgx.Tx) ([][KeyLen]byte, error) {
	var (
		left     [KeyLen]byte
		siblings [][KeyLen]byte
	)

	// the merkle tree is 0-indexed, so we need to subtract 1
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

func (mt *MerkleTree) addLeaf(ctx context.Context, leaf [KeyLen]byte, dbTx pgx.Tx) error {
	index := mt.count
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

	mt.count++
	rootID, err := mt.store.SetRoot(ctx, cur[:], mt.count, mt.network, dbTx)
	if err != nil {
		return err
	}
	for _, leaf := range leaves {
		err := mt.store.Set(ctx, leaf[0], [][]byte{leaf[1], leaf[2]}, rootID, dbTx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mt *MerkleTree) resetLeaf(ctx context.Context, depositCount uint, dbTx pgx.Tx) error {
	err := mt.store.ResetMT(ctx, depositCount, mt.network, dbTx)
	if err != nil {
		return err
	}

	mt.count = depositCount
	root, err := mt.getRoot(ctx, dbTx)
	if err != nil {
		return err
	}
	mt.siblings, err = mt.initSiblings(ctx, root, dbTx)

	return err
}

// this function is used to get the current root of the merkle tree
func (mt *MerkleTree) getRoot(ctx context.Context, dbTx pgx.Tx) ([]byte, error) {
	return mt.store.GetRoot(ctx, mt.count, mt.network, dbTx)
}
