package server

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygon/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygon/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygon/zkevm-bridge-service/etherman"
	"github.com/0xPolygon/zkevm-bridge-service/utils/gerror"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetClaimProofbyGER(t *testing.T) {
	cfg := Config{
		CacheSize: 32,
	}
	mockStorage := newBridgeServiceStorageMock(t)
	sut := NewBridgeService(cfg, 32, []uint32{0, 1}, mockStorage)
	var (
		depositCnt uint32
		networkID  uint32
	)
	GER := common.Hash{}
	deposit := &etherman.Deposit{}
	mockStorage.EXPECT().GetDeposit(mock.Anything, depositCnt, networkID, mock.Anything).Return(deposit, nil)
	exitRoot := etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{{}, {}},
	}
	mockStorage.EXPECT().GetL1ExitRootByGER(mock.Anything, GER, mock.Anything).Return(&exitRoot, nil)
	mockStorage.EXPECT().GetDepositCountByRoot(mock.Anything, mock.Anything, networkID, mock.Anything).Return(uint32(0), nil)
	node := [][]byte{{}, {}}
	mockStorage.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(node, nil)
	smtProof, smtRollupProof, globaExitRoot, err := sut.GetClaimProofbyGER(context.Background(), depositCnt, networkID, GER, nil)
	require.NoError(t, err)
	require.NotNil(t, smtProof)
	require.NotNil(t, smtRollupProof)
	require.NotNil(t, globaExitRoot)
}

const testTreeHeight = 32

// inMemoryExitTree replicates the persistence pattern of bridgectrl.MerkleTree.addLeaf:
// for every level of an inserted leaf's path it stores parent -> (left child, right child)
// in a reverse hash table, exactly like the mt.rht table used by the bridge service.
type inMemoryExitTree struct {
	count      uint32
	siblings   [][bridgectrl.KeyLen]byte
	zeroHashes [][bridgectrl.KeyLen]byte
	// nodes is the reverse hash table: parent hash -> [left, right]
	nodes map[[bridgectrl.KeyLen]byte][][]byte
	root  [bridgectrl.KeyLen]byte
}

func newInMemoryExitTree() *inMemoryExitTree {
	zeroHashes := make([][bridgectrl.KeyLen]byte, testTreeHeight+1)
	for h := 1; h <= testTreeHeight; h++ {
		zeroHashes[h] = bridgectrl.Hash(zeroHashes[h-1], zeroHashes[h-1])
	}
	siblings := make([][bridgectrl.KeyLen]byte, testTreeHeight)
	for h := 0; h < testTreeHeight; h++ {
		siblings[h] = zeroHashes[h]
	}
	return &inMemoryExitTree{
		siblings:   siblings,
		zeroHashes: zeroHashes,
		nodes:      make(map[[bridgectrl.KeyLen]byte][][]byte),
		root:       zeroHashes[testTreeHeight],
	}
}

// addLeaf mirrors bridgectrl.(*MerkleTree).addLeaf.
func (t *inMemoryExitTree) addLeaf(leaf [bridgectrl.KeyLen]byte) {
	index := t.count
	cur := leaf
	isFilledSubTree := true
	for h := uint8(0); h < testTreeHeight; h++ {
		var parent [bridgectrl.KeyLen]byte
		if index&(1<<h) > 0 {
			parent = bridgectrl.Hash(t.siblings[h], cur)
			t.nodes[parent] = [][]byte{append([]byte{}, t.siblings[h][:]...), append([]byte{}, cur[:]...)}
		} else {
			if isFilledSubTree {
				t.siblings[h] = cur
				isFilledSubTree = false
			}
			parent = bridgectrl.Hash(cur, t.zeroHashes[h])
			t.nodes[parent] = [][]byte{append([]byte{}, cur[:]...), append([]byte{}, t.zeroHashes[h][:]...)}
		}
		cur = parent
	}
	t.root = cur
	t.count++
}

// computeRootFromProof rebuilds the root from a leaf, its index and a proof
// (same algorithm as the on-chain claim verification).
func computeRootFromProof(leaf [bridgectrl.KeyLen]byte, index uint32, proof [][bridgectrl.KeyLen]byte) common.Hash {
	node := leaf
	for h := uint8(0); h < testTreeHeight; h++ {
		if (index>>h)&1 == 1 {
			node = bridgectrl.Hash(proof[h], node)
		} else {
			node = bridgectrl.Hash(node, proof[h])
		}
	}
	return common.BytesToHash(node[:])
}

func newTestDeposit(depositCount uint32) *etherman.Deposit {
	return &etherman.Deposit{
		LeafType:           0,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Amount:             big.NewInt(1000000 + int64(depositCount)),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		DepositCount:       depositCount,
		NetworkID:          0,
		Metadata:           []byte{},
	}
}

// newSutWithExitTree wires a bridgeService whose storage mock serves an
// in-memory replica of the exit tree of the given network frozen at the moment
// a GER was published, while the deposit table contains more (later) deposits.
// The GetDepositCountByRoot mock resolves the last deposit count only for the
// frozen root, mirroring the mt.root table semantics.
func newSutWithExitTree(t *testing.T, tree *inMemoryExitTree, deposits []*etherman.Deposit, ger common.Hash, networkID uint32) *bridgeService {
	t.Helper()
	mockStorage := newBridgeServiceStorageMock(t)
	sut := NewBridgeService(Config{CacheSize: 32}, testTreeHeight, []uint32{0, 1}, mockStorage)

	treeRoot := common.BytesToHash(tree.root[:])
	mockStorage.EXPECT().GetDeposit(mock.Anything, mock.Anything, networkID, mock.Anything).RunAndReturn(
		func(_ context.Context, depositCnt, _ uint32, _ interface{}) (*etherman.Deposit, error) {
			if int(depositCnt) < len(deposits) {
				return deposits[depositCnt], nil
			}
			return nil, gerror.ErrStorageNotFound
		})
	mainnetRoot := common.Hash{}
	rollupRoot := common.Hash{}
	if networkID == 0 {
		mainnetRoot = treeRoot
	} else {
		// The rollup exit tree contains the frozen L2 exit root (LER) as its only leaf.
		leaves := [][bridgectrl.KeyLen]byte{tree.root}
		_, computedRollupRoot, err := bridgectrl.ComputeSiblings(networkID-1, leaves, testTreeHeight)
		require.NoError(t, err)
		rollupRoot = computedRollupRoot
		mockStorage.EXPECT().GetRollupExitLeavesByRoot(mock.Anything, rollupRoot, mock.Anything).Return(
			[]etherman.RollupExitLeaf{{RollupId: networkID, Leaf: treeRoot, Root: rollupRoot}}, nil)
	}
	mockStorage.EXPECT().GetL1ExitRootByGER(mock.Anything, ger, mock.Anything).Return(
		&etherman.GlobalExitRoot{
			GlobalExitRoot: ger,
			ExitRoots:      []common.Hash{mainnetRoot, rollupRoot},
		}, nil)
	mockStorage.EXPECT().GetDepositCountByRoot(mock.Anything, mock.Anything, networkID, mock.Anything).RunAndReturn(
		func(_ context.Context, root []byte, _ uint32, _ interface{}) (uint32, error) {
			if common.BytesToHash(root) == treeRoot && tree.count > 0 {
				return tree.count - 1, nil
			}
			return 0, gerror.ErrStorageNotFound
		})
	// Maybe(): requests rejected by the inclusion check never reach the tree traversal.
	mockStorage.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, key []byte, _ interface{}) ([][]byte, error) {
			var k [bridgectrl.KeyLen]byte
			copy(k[:], key)
			if node, ok := tree.nodes[k]; ok {
				return node, nil
			}
			return nil, gerror.ErrStorageNotFound
		}).Maybe()
	return sut
}

// TestGetProofByGERRejectsDepositNotInTree is the regression test for the
// off-by-one bug of the merkle-proof-by-ger API.
//
// Scenario: the exit root referenced by the GER was computed when the tree had
// 3 leaves (deposits 0, 1, 2). Deposit 3 was synced into the DB later, so it
// exists, but it is NOT part of that exit root.
//
// Before the fix, querying deposit_cnt=3 returned a 32-sibling proof without
// error: bridgectrl.MerkleTree.addLeaf persists frontier nodes with zero-hash
// right children, so getProof could walk one index past the last leaf whenever
// the last inserted leaf had an even index. The returned siblings were a valid
// Merkle proof of the ZERO leaf at index 3 (a proof of vacancy), presented as
// a legitimate claim proof that would revert on-chain.
//
// With the fix, the deposit count is validated against the root
// (GetDepositCountByRoot) before traversing the tree, so any deposit_cnt
// beyond the last included leaf is rejected with a clear error.
func TestGetProofByGERRejectsDepositNotInTree(t *testing.T) {
	ctx := context.Background()
	ger := common.HexToHash("0x1234")

	deposits := make([]*etherman.Deposit, 5)
	for i := range deposits {
		deposits[i] = newTestDeposit(uint32(i))
	}

	// Freeze the exit tree with 3 leaves: deposits 0..2. Last leaf index = 2 (even),
	// the parity that used to trigger the off-by-one.
	tree := newInMemoryExitTree()
	for i := 0; i < 3; i++ {
		tree.addLeaf(bridgectrl.HashDeposit(deposits[i]))
	}
	root := common.BytesToHash(tree.root[:])
	sut := newSutWithExitTree(t, tree, deposits, ger, 0)

	t.Run("included deposit returns a valid proof", func(t *testing.T) {
		_, proof, _, err := sut.GetClaimProofbyGER(ctx, 2, 0, ger, nil)
		require.NoError(t, err)
		require.Len(t, proof, testTreeHeight)
		require.Equal(t, root, computeRootFromProof(bridgectrl.HashDeposit(deposits[2]), 2, proof),
			"proof for an included deposit must verify against the exit root")
	})

	t.Run("deposit_cnt == leaf count is rejected (off-by-one regression)", func(t *testing.T) {
		_, proof, _, err := sut.GetClaimProofbyGER(ctx, 3, 0, ger, nil)
		require.Error(t, err, "deposit 3 is not part of this GER's exit root, no proof must be returned")
		require.ErrorContains(t, err, "is not included in the exit root")
		require.Nil(t, proof)
	})

	t.Run("rejection surfaces through the GetProofByGER endpoint as well", func(t *testing.T) {
		resp, err := sut.GetProofByGER(ctx, &pb.GetProofByGERRequest{
			NetId:      0,
			DepositCnt: 3,
			Ger:        ger.Hex(),
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "is not included in the exit root")
		require.Nil(t, resp)
	})

	t.Run("deposit_cnt beyond leaf count + 1 is rejected too", func(t *testing.T) {
		_, _, _, err := sut.GetClaimProofbyGER(ctx, 4, 0, ger, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "is not included in the exit root")
	})
}

// TestGetProofByGERRejectsDepositNotInTreeEvenLeafCount covers the parity that
// never triggered the off-by-one (last inserted leaf with an odd index). The
// request must still be rejected, now with the explicit inclusion error
// instead of the incidental storage-lookup failure of the tree traversal.
func TestGetProofByGERRejectsDepositNotInTreeEvenLeafCount(t *testing.T) {
	ctx := context.Background()
	ger := common.HexToHash("0x5678")

	deposits := make([]*etherman.Deposit, 3)
	for i := range deposits {
		deposits[i] = newTestDeposit(uint32(i))
	}

	// Freeze the exit tree with 2 leaves: deposits 0..1. Last leaf index = 1 (odd).
	tree := newInMemoryExitTree()
	for i := 0; i < 2; i++ {
		tree.addLeaf(bridgectrl.HashDeposit(deposits[i]))
	}
	sut := newSutWithExitTree(t, tree, deposits, ger, 0)

	_, _, _, err := sut.GetClaimProofbyGER(ctx, 2, 0, ger, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "is not included in the exit root")
}

// TestGetProofByGERRejectsDepositNotInTreeRollup exercises the rollup branch of
// GetClaimProofbyGER: the L2 exit tree (LER) referenced through the rollup exit
// root has 3 leaves, and the same off-by-one index (deposit_cnt == leaf count)
// must be rejected there as well.
func TestGetProofByGERRejectsDepositNotInTreeRollup(t *testing.T) {
	ctx := context.Background()
	ger := common.HexToHash("0x9abc")
	const networkID = uint32(1)

	deposits := make([]*etherman.Deposit, 5)
	for i := range deposits {
		deposits[i] = newTestDeposit(uint32(i))
		deposits[i].NetworkID = networkID
		deposits[i].DestinationNetwork = 0
	}

	// Freeze the L2 exit tree with 3 leaves: deposits 0..2. Last leaf index = 2 (even).
	tree := newInMemoryExitTree()
	for i := 0; i < 3; i++ {
		tree.addLeaf(bridgectrl.HashDeposit(deposits[i]))
	}
	ler := common.BytesToHash(tree.root[:])
	sut := newSutWithExitTree(t, tree, deposits, ger, networkID)

	t.Run("included deposit returns a valid proof against the LER", func(t *testing.T) {
		_, proof, rollupProof, err := sut.GetClaimProofbyGER(ctx, 2, networkID, ger, nil)
		require.NoError(t, err)
		require.Len(t, proof, testTreeHeight)
		require.Len(t, rollupProof, testTreeHeight)
		require.Equal(t, ler, computeRootFromProof(bridgectrl.HashDeposit(deposits[2]), 2, proof),
			"proof for an included deposit must verify against the L2 exit root")
	})

	t.Run("deposit_cnt == leaf count is rejected (off-by-one regression)", func(t *testing.T) {
		_, _, _, err := sut.GetClaimProofbyGER(ctx, 3, networkID, ger, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "is not included in the exit root")
	})
}
