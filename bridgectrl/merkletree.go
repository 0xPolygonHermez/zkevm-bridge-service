package bridgectrl

import (
	"context"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
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
			return nil, fmt.Errorf("height: %d, cur: %v, error: %v", h, cur, err)
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

	err := mt.store.SetRoot(ctx, cur[:], depositID, mt.network, dbTx)
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

func buildIntermediate(leaves [][KeyLen]byte) ([][][]byte, [][32]byte) {
	var (
		nodes  [][][]byte
		hashes [][KeyLen]byte
	)
	for i := 0; i < len(leaves); i += 2 {
		var left, right int = i, i + 1
		hash := Hash(leaves[left], leaves[right])
		nodes = append(nodes, [][]byte{hash[:], leaves[left][:], leaves[right][:]})
		hashes = append(hashes, hash)
	}
	return nodes, hashes
}

func (mt *MerkleTree) updateLeaf(ctx context.Context, depositID uint64, leaves [][KeyLen]byte, dbTx pgx.Tx) error {
	var (
		nodes [][][][]byte
		ns    [][][]byte
	)
	initLeavesCount := uint(len(leaves))
	if len(leaves) == 0 {
		leaves = append(leaves, zeroHashes[0])
	}

	for h := uint8(0); h < mt.height; h++ {
		if len(leaves)%2 == 1 {
			leaves = append(leaves, zeroHashes[h])
		}
		ns, leaves = buildIntermediate(leaves)
		nodes = append(nodes, ns)
	}
	if len(ns) != 1 {
		return fmt.Errorf("error: more than one root detected: %+v", nodes)
	}
	log.Debug("Root calculated: ", common.Bytes2Hex(ns[0][0]))
	err := mt.store.SetRoot(ctx, ns[0][0], depositID, mt.network, dbTx)
	if err != nil {
		return err
	}
	var nodesToStore [][]interface{}
	for _, leaves := range nodes {
		for _, leaf := range leaves {
			nodesToStore = append(nodesToStore, []interface{}{leaf[0], [][]byte{leaf[1], leaf[2]}, depositID})
		}
	}
	if err := mt.store.BulkSet(ctx, nodesToStore, dbTx); err != nil {
		return err
	}
	mt.count = initLeavesCount
	return nil
}

func (mt *MerkleTree) getLeaves(ctx context.Context, dbTx pgx.Tx) ([][KeyLen]byte, error) {
	root, err := mt.getRoot(ctx, dbTx)
	if err != nil {
		return nil, err
	}
	cur := [][]byte{root}
	// It starts in height-1 because 0 is the level of the leafs
	for h := int(mt.height - 1); h >= 0; h-- {
		var levelLeaves [][]byte
		for _, c := range cur {
			leaves, err := mt.store.Get(ctx, c, dbTx)
			if err != nil {
				var isZero bool
				curHash := common.BytesToHash(c)
				for _, h := range zeroHashes {
					if common.BytesToHash(h[:]) == curHash {
						isZero = true
					}
				}
				if !isZero {
					return nil, fmt.Errorf("height: %d, cur: %v, error: %v", h, cur, err)
				}
			}
			levelLeaves = append(levelLeaves, leaves...)
		}
		cur = levelLeaves
	}
	var result [][KeyLen]byte
	for _, l := range cur {
		var aux [KeyLen]byte
		copy(aux[:], l)
		result = append(result, aux)
	}
	return result, nil
}

func (mt *MerkleTree) buildMTRoot(leaves [][KeyLen]byte) (common.Hash, error) {
	var (
		nodes [][][][]byte
		ns    [][][]byte
	)
	if len(leaves) == 0 {
		leaves = append(leaves, zeroHashes[0])
	}

	for h := uint8(0); h < mt.height; h++ {
		if len(leaves)%2 == 1 {
			leaves = append(leaves, zeroHashes[h])
		}
		ns, leaves = buildIntermediate(leaves)
		nodes = append(nodes, ns)
	}
	if len(ns) != 1 {
		return common.Hash{}, fmt.Errorf("error: more than one root detected: %+v", nodes)
	}
	log.Debug("Root calculated: ", common.Bytes2Hex(ns[0][0]))

	return common.BytesToHash(ns[0][0]), nil
}

func (mt MerkleTree) storeLeaves(ctx context.Context, leaves [][KeyLen]byte, blockID uint64, dbTx pgx.Tx) error {
	root, err := mt.buildMTRoot(leaves)
	if err != nil {
		return err
	}
	// Check if root is already stored. If so, don't save the leaves because they are already stored on the db.
	exist, err := mt.store.IsRollupExitRoot(ctx, root, dbTx)
	if err != nil {
		return err
	}
	if !exist {
		var inserts [][]interface{}
		for i := range leaves {
			inserts = append(inserts, []interface{}{leaves[i][:], i + 1, root.Bytes(), blockID})
		}
		if err := mt.store.AddRollupExitLeaves(ctx, inserts, dbTx); err != nil {
			return err
		}
	}
	return nil
}

// func (mt MerkleTree) getLatestRollupExitLeaves(ctx context.Context, dbTx pgx.Tx) ([]etherman.RollupExitLeaf, error) {
// 	return mt.store.GetLatestRollupExitLeaves(ctx, dbTx)
// }

func (mt MerkleTree) addRollupExitLeaf(ctx context.Context, rollupLeaf etherman.RollupExitLeaf, dbTx pgx.Tx) error {
	storedRollupLeaves, err := mt.store.GetLatestRollupExitLeaves(ctx, dbTx)
	if err != nil {
		log.Error("error getting latest rollup exit leaves. Error: ", err)
		return err
	}
	// If rollupLeaf.RollupId is lower or equal than len(storedRollupLeaves), we can add it in the proper position of the array
	// if rollupLeaf.RollupId <= uint64(len(storedRollupLeaves)) {
	// 	if storedRollupLeaves[rollupLeaf.RollupId-1].RollupId == rollupLeaf.RollupId {
	// 		storedRollupLeaves[rollupLeaf.RollupId-1] = rollupLeaf
	// 	} else {
	// 		return fmt.Errorf("error: RollupId doesn't match")
	// 	}
	// } else {

	// If rollupLeaf.RollupId is higher than len(storedRollupLeaves), We have to add empty rollups until the new rollupID
	for i := len(storedRollupLeaves); i < int(rollupLeaf.RollupId); i++ {
		storedRollupLeaves = append(storedRollupLeaves, etherman.RollupExitLeaf{
			BlockID:  rollupLeaf.BlockID,
			RollupId: uint(i + 1),
		})
	}
	if storedRollupLeaves[rollupLeaf.RollupId-1].RollupId == rollupLeaf.RollupId {
		storedRollupLeaves[rollupLeaf.RollupId-1] = rollupLeaf
	} else {
		return fmt.Errorf("error: RollupId doesn't match")
	}
	// }
	var leaves [][KeyLen]byte
	for _, l := range storedRollupLeaves {
		var aux [KeyLen]byte
		copy(aux[:], l.Leaf[:])
		leaves = append(leaves, aux)
	}
	err = mt.storeLeaves(ctx, leaves, rollupLeaf.BlockID, dbTx)
	if err != nil {
		log.Error("error storing leaves. Error: ", err)
		return err
	}
	return nil
}

func ComputeSiblings(rollupIndex uint, leaves [][KeyLen]byte, height uint8) ([][KeyLen]byte, common.Hash, error) {
	var ns [][][]byte
	if len(leaves) == 0 {
		leaves = append(leaves, zeroHashes[0])
	}
	var siblings [][KeyLen]byte
	index := rollupIndex
	for h := uint8(0); h < height; h++ {
		if len(leaves)%2 == 1 {
			leaves = append(leaves, zeroHashes[h])
		}
		if index%2 == 1 { //If it is odd
			siblings = append(siblings, leaves[index-1])
		} else { // It is even
			if len(leaves) > 1 {
				siblings = append(siblings, leaves[index+1])
			}
		}
		var (
			nsi    [][][]byte
			hashes [][KeyLen]byte
		)
		for i := 0; i < len(leaves); i += 2 {
			var left, right int = i, i + 1
			hash := Hash(leaves[left], leaves[right])
			nsi = append(nsi, [][]byte{hash[:], leaves[left][:], leaves[right][:]})
			hashes = append(hashes, hash)
		}
		// Find the index of the leave in the next level of the tree.
		// Divide the index by 2 to find the position in the upper level
		index = uint(float64(index) / 2) //nolint:gomnd
		ns = nsi
		leaves = hashes
	}
	if len(ns) != 1 {
		return nil, common.Hash{}, fmt.Errorf("error: more than one root detected: %+v", ns)
	}

	return siblings, common.BytesToHash(ns[0][0]), nil
}

func calculateRoot(leafHash common.Hash, smtProof [][KeyLen]byte, index uint, height uint8) common.Hash {
	var node [KeyLen]byte
	copy(node[:], leafHash[:])

	// Check merkle proof
	var h uint8
	for h = 0; h < height; h++ {
		if ((index >> h) & 1) == 1 {
			node = Hash(smtProof[h], node)
		} else {
			node = Hash(node, smtProof[h])
		}
	}
	return common.BytesToHash(node[:])
}
