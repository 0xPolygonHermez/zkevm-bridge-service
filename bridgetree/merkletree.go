package bridgetree

import (
	"context"
	"encoding/binary"
)

// MerkleTree struct
type MerkleTree struct {
	// store is the database storage to store all node data
	store Store
	// height is the depth of the merkle tree
	height uint8
	// counts is the array to track the number of existing nodes in each layer
	counts []uint64
	// zeroHashes is the pre-calculated zero hash array
	zeroHashes [][KeyLen]byte
}

// NewMerkleTree creates new MerkleTree
func NewMerkleTree(store Store, height uint8) *MerkleTree {
	counts := make([]uint64, 0)
	for i := 0; i <= int(height); i++ {
		counts = append(counts, 0)
	}
	return &MerkleTree{
		store:      store,
		height:     height,
		counts:     counts,
		zeroHashes: generateZeroHashes(height),
	}
}

func (mt *MerkleTree) addLeaf(ctx context.Context, leaf [KeyLen]byte) error {
	index := mt.counts[0]
	cur := leaf
	for height := 0; height < int(mt.height); height++ {
		// Set the current value in the specific height
		err := mt.store.Set(ctx, getByteKey(height, index), cur[:])
		if err != nil {
			return err
		}

		if index == mt.counts[height] {
			mt.counts[height] = mt.counts[height] + 1
		}

		sibling, err := mt.getValueByIndex(ctx, height, index^1)
		if err != nil {
			return err
		}
		if index%2 == 0 {
			cur = hash(cur, sibling)
		} else {
			cur = hash(sibling, cur)
		}
		index /= 2
	}
	// Set the root
	mt.counts[mt.height] = 1
	err := mt.store.Set(ctx, getByteKey(int(mt.height), index), cur[:])
	return err
}

func (mt *MerkleTree) getProofTreeByIndex(ctx context.Context, index uint64) ([][KeyLen]byte, error) {
	var proof [][KeyLen]byte
	currentIndex := index
	for height := 0; height < int(mt.height); height++ {
		currentIndex = currentIndex ^ 1

		sibling, err := mt.getValueByIndex(ctx, height, currentIndex)
		if err != nil {
			return proof, err
		}

		proof = append(proof, sibling)
		currentIndex = currentIndex / 2 //nolint:gomnd
	}
	return proof, nil
}

func (mt *MerkleTree) getRoot(ctx context.Context) ([KeyLen]byte, error) {
	if mt.counts[0] == 0 {
		return mt.zeroHashes[mt.height], nil
	}

	return mt.getValueByIndex(ctx, int(mt.height), 0)
}

func (mt *MerkleTree) getValueByIndex(ctx context.Context, height int, index uint64) ([KeyLen]byte, error) {
	if index >= mt.counts[height] {
		return mt.zeroHashes[height], nil
	}

	var res [KeyLen]byte
	value, err := mt.store.Get(ctx, getByteKey(height, index))
	copy(res[:], value)
	return res, err
}

func getByteKey(height int, index uint64) []byte {
	key := make([]byte, 8) //nolint:gomnd
	key = append(key, byte(height))
	binary.LittleEndian.PutUint64(key, index)
	return key
}
