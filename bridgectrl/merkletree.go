package bridgectrl

import (
	"context"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
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
	// root is the value of the root node, count and root are only used in the synchronizer side
	root [KeyLen]byte
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
func NewMerkleTree(ctx context.Context, store merkleTreeStore, height, network uint8) (*MerkleTree, error) {
	depositCnt, err := store.GetLastDepositCount(ctx, network, nil)
	if err != nil {
		if err == gerror.ErrStorageNotFound {
			for h := uint8(0); h < height; h++ {
				// h+1 is the position of the parent node and h is the position of the values of the children nodes. As all the nodes of the same nodes has the same values
				// we can only store the info ones
				err := store.Set(ctx, zeroHashes[h+1][:], [][]byte{zeroHashes[h][:], zeroHashes[h][:]}, nil)
				if err != nil {
					return nil, err
				}
			}
			err1 := store.SetRoot(ctx, zeroHashes[height][:], 0, network, nil)
			if err1 != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var mtRoot [KeyLen]byte
	root, err := store.GetRoot(ctx, depositCnt, network, nil)
	if err != nil {
		return nil, err
	}
	copy(mtRoot[:], root)

	return &MerkleTree{
		store:   store,
		network: network,
		height:  height,
		count:   depositCnt,
		root:    mtRoot,
	}, nil
}

func (mt *MerkleTree) getSiblings(ctx context.Context, index uint, root [KeyLen]byte) ([][KeyLen]byte, error) {
	var (
		left, right [KeyLen]byte
		siblings    [][KeyLen]byte
	)

	cur := root
	// It starts in height-1 because 0 is the level of the leafs
	for h := mt.height - 1; ; h-- {
		value, err := mt.store.Get(ctx, cur[:], nil)
		if err != nil {
			return nil, fmt.Errorf("height: %d, cur: %v, error: %w", h, cur, err)
		}

		copy(left[:], value[0])
		copy(right[:], value[1])

		/*
					*        Root                (level h=3 => height=4)
					*      /     \
					*	 O5       O6             (level h=2)
					*	/ \      / \
					*  O1  O2   O3  O4           (level h=1)
			        *  /\   /\   /\ /\
					* 0  1 2  3 4 5 6 7 Leafs    (level h=0)
					* Example 1:
					* Choose index = 3 => 011 binary
					* Assuming we are in level 1 => h=1; 1<<h = 010 binary
					* Now, let's do AND operation => 011&010=010 which is higher than 0 so we need the left sibling (O1)
					* Example 2:
					* Choose index = 4 => 100 binary
					* Assuming we are in level 1 => h=1; 1<<h = 010 binary
					* Now, let's do AND operation => 100&010=000 which is not higher than 0 so we need the right sibling (O4)
					* Example 3:
					* Choose index = 4 => 100 binary
					* Assuming we are in level 2 => h=2; 1<<h = 100 binary
					* Now, let's do AND operation => 100&100=100 which is higher than 0 so we need the left sibling (O5)
		*/

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

	// We need to invert the siblings to go from leafs to the top
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
	err = mt.store.SetRoot(ctx, cur[:], mt.count, mt.network, nil)
	return err
}

func (mt *MerkleTree) resetLeaf(ctx context.Context, depositCount uint) error {
	err := mt.store.ResetMT(ctx, depositCount, mt.network, nil)
	if err != nil {
		return err
	}

	mt.count = depositCount
	root, err := mt.store.GetRoot(ctx, depositCount, mt.network, nil)
	if err != nil {
		return err
	}

	copy(mt.root[:], root)
	return nil
}
