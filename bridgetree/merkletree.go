package bridgetree

import (
	"context"
)

// zeroHashes is the pre-calculated zero hash array
var zeroHashes [][KeyLen]byte

// MerkleTree struct
type MerkleTree struct {
	// store is the database storage to store all node data
	store merkleTreeStore
	// height is the depth of the merkle tree
	height uint8
	// count is the number of deposit
	count uint
	// root is the value of the root node, count and root are only used in the synchronizer side
	root [KeyLen]byte
}

func init() {
	zeroHashes = generateZeroHashes(64) // nolint
}

// NewMerkleTree creates new MerkleTree.
func NewMerkleTree(ctx context.Context, store merkleTreeStore, height uint8) (*MerkleTree, error) {
	for h := uint8(0); h < height; h++ {
		err := store.Set(ctx, zeroHashes[h+1][:], [][]byte{zeroHashes[h][:], zeroHashes[h][:]})
		if err != nil {
			return nil, err
		}
	}
	return &MerkleTree{
		store:  store,
		height: height,
		count:  0,
		root:   zeroHashes[height],
	}, nil
}

func (mt *MerkleTree) getSiblings(ctx context.Context, index uint, root [KeyLen]byte) ([][KeyLen]byte, error) {
	var (
		left, right [KeyLen]byte
		siblings    [][KeyLen]byte
	)

	cur := root
	for h := mt.height - 1; ; h-- {
		value, err := mt.store.Get(ctx, cur[:])
		copy(left[:], value[0])
		copy(right[:], value[1])

		if err != nil {
			return nil, err
		}

		if index&(1<<h) > 0 {
			siblings = append(siblings, left)
			cur = right
		} else {
			siblings = append(siblings, right)
			cur = left
		}

		if h == 0 {
			break
		}
	}

	return siblings, nil
}

func (mt *MerkleTree) addLeaf(ctx context.Context, leaf [KeyLen]byte) error {
	var parent [KeyLen]byte

	index := mt.count
	cur := leaf

	siblings, err := mt.getSiblings(ctx, index, mt.root)
	if err != nil {
		return err
	}

	for h := uint8(0); h < mt.height; h++ {
		if index&(1<<h) > 0 {
			parent = hash(siblings[mt.height-h-1], cur)
			err := mt.store.Set(ctx, parent[:], [][]byte{siblings[mt.height-h-1][:], cur[:]})
			if err != nil {
				return err
			}
		} else {
			parent = hash(cur, siblings[mt.height-h-1])
			err := mt.store.Set(ctx, parent[:], [][]byte{cur[:], siblings[mt.height-h-1][:]})
			if err != nil {
				return err
			}
		}
		cur = parent
	}

	// Set the root value
	mt.root = cur
	mt.count++
	return mt.store.SetMTRoot(ctx, mt.count, cur[:])
}
