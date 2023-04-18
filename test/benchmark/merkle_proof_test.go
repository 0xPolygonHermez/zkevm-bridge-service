package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/stretchr/testify/require"
)

func getProofStep(store operations.StorageInterface, index uint, root []byte) ([][bridgectrl.KeyLen]byte, error) {
	var (
		siblings         [][bridgectrl.KeyLen]byte
		left, right, cur [bridgectrl.KeyLen]byte
	)

	copy(cur[:], root)
	ctx := context.Background()
	// It starts in height-1 because 0 is the level of the leafs
	for h := 31; h >= 0; h-- {
		value, err := store.Get(ctx, cur[:], nil)
		copy(left[:], value[0])
		copy(right[:], value[1])
		if err != nil {
			return nil, fmt.Errorf("height: %d, cur: %v, error: %w", h, cur, err)
		}

		if index&(1<<h) > 0 {
			siblings = append(siblings, left)
			cur = right
		} else {
			siblings = append(siblings, right)
			cur = left
		}
	}

	// We need to invert the siblings to go from leafs to the top
	for st, en := 0, len(siblings)-1; st < en; st, en = st+1, en-1 {
		siblings[st], siblings[en] = siblings[en], siblings[st]
	}

	return siblings, nil
}

func getProof(store operations.StorageInterface, index uint, root []byte) ([][bridgectrl.KeyLen]byte, error) {
	var (
		siblings    [][bridgectrl.KeyLen]byte
		left, right [bridgectrl.KeyLen]byte
	)

	ctx := context.Background()
	nodes, err := store.GetNodes(ctx, uint(index), root, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %w", err)
	}
	for h := 0; h < 32; h++ {
		copy(left[:], nodes[h][0])
		copy(right[:], nodes[h][1])

		// It starts in height-1 because 0 is the level of the leafs
		if index&(1<<(31-uint8(h))) > 0 {
			siblings = append(siblings, left)
		} else {
			siblings = append(siblings, right)
		}
	}

	// We need to invert the siblings to go from leafs to the top
	for st, en := 0, len(siblings)-1; st < en; st, en = st+1, en-1 {
		siblings[st], siblings[en] = siblings[en], siblings[st]
	}

	return siblings, nil
}

func initStore(b *testing.B, bench benchmark) operations.StorageInterface {
	r := rand.New(rand.NewSource(bench.seed)) //nolint: gosec
	bt, store, err := operations.RunMockServer(bench.store, bench.mtHeight, networks)
	require.NoError(b, err)
	ctx := context.Background()
	for i := 0; i < bench.initSize; i++ {
		dbTx, err := store.BeginDBTransaction(ctx)
		require.NoError(b, err)
		id, err := store.AddBlock(ctx, &etherman.Block{
			BlockNumber: uint64(i),
			BlockHash:   utils.GenerateRandomHash(),
			ParentHash:  utils.GenerateRandomHash(),
		}, dbTx)
		require.NoError(b, err)
		deposit := randDeposit(r, uint(i), id, 0)
		depositID, err := store.AddDeposit(ctx, deposit, dbTx)
		require.NoError(b, err)
		require.NoError(b, bt.AddDeposit(deposit, depositID, dbTx))
		require.NoError(b, dbTx.Commit(ctx))
	}
	return store
}

func BenchmarkMerkleProof(b *testing.B) {
	bench := benchmark{756509998, "postgres", 32, 1000, 0}
	ctx := context.Background()
	store := initStore(b, bench)
	b.ResetTimer()
	prefix := fmt.Sprintf("test-merkle-proof-one-query-%s-%d", bench.store, bench.initSize)
	b.Run(prefix, func(sub *testing.B) {
		sub.ReportAllocs()
		for n := 0; n < sub.N; n++ {
			for i := 0; i < bench.initSize; i++ {
				root, err := store.GetRoot(ctx, uint(i), 0, nil)
				require.NoError(sub, err)
				_, err = getProof(store, uint(i), root)
				require.NoError(sub, err)
			}
		}
	})
	prefix = fmt.Sprintf("test-merkle-proof-step-by-step-%s-%d", bench.store, bench.initSize)
	b.Run(prefix, func(sub *testing.B) {
		sub.ReportAllocs()
		for n := 0; n < sub.N; n++ {
			for i := 0; i < bench.initSize; i++ {
				root, err := store.GetRoot(ctx, uint(i), 0, nil)
				require.NoError(sub, err)
				_, err = getProofStep(store, uint(i), root)
				require.NoError(sub, err)
			}
		}
	})
}
