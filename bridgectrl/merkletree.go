package bridgectrl

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
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
	depositCnt, err := store.GetLastDepositCount(ctx, nil)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			for h := uint8(0); h < height; h++ {
				err := store.Set(ctx, zeroHashes[h+1][:], [][]byte{zeroHashes[h][:], zeroHashes[h][:]}, nil)
				if err != nil {
					return nil, err
				}
			}
			err1 := store.SetRoot(ctx, zeroHashes[height][:], 0, nil)
			if err1 != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var mtRoot [KeyLen]byte
	root, err := store.GetRoot(ctx, depositCnt, nil)
	if err != nil {
		return nil, err
	}
	copy(mtRoot[:], root)

	return &MerkleTree{
		store:  store,
		height: height,
		count:  depositCnt,
		root:   mtRoot,
	}, nil
}

func (mt *MerkleTree) getSiblings(ctx context.Context, index uint, root [KeyLen]byte) ([][KeyLen]byte, error) {
	var (
		left, right [KeyLen]byte
		siblings    [][KeyLen]byte
	)

	cur := root
	for h := mt.height - 1; ; h-- {
		value, err := mt.store.Get(ctx, cur[:], nil)
		if err != nil {
			return nil, err
		}

		copy(left[:], value[0])
		copy(right[:], value[1])

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

	for st, en := 0, len(siblings)-1; st < en; st, en = st+1, en-1 {
		siblings[st], siblings[en] = siblings[en], siblings[st]
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
			parent = hash(siblings[h], cur)
			err := mt.store.Set(ctx, parent[:], [][]byte{siblings[h][:], cur[:]}, nil)
			if err != nil {
				return err
			}
		} else {
			parent = hash(cur, siblings[h])
			err := mt.store.Set(ctx, parent[:], [][]byte{cur[:], siblings[h][:]}, nil)
			if err != nil {
				return err
			}
		}
		cur = parent
	}

	// Set the root value
	mt.root = cur
	mt.count++
	err = mt.store.SetRoot(ctx, cur[:], mt.count, nil)
	return err
}

func (mt *MerkleTree) resetLeaf(ctx context.Context, depositCount uint) error {
	err := mt.store.ResetMT(ctx, depositCount, nil)
	if err != nil {
		return err
	}

	mt.count = depositCount
	root, err := mt.store.GetRoot(ctx, depositCount, nil)
	if err != nil {
		return err
	}

	copy(mt.root[:], root)
	return nil
}
