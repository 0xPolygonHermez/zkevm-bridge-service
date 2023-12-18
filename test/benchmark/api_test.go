package benchmark

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/client"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	networks  []uint = []uint{0, 1}
	addresses        = initAddresses(10)
)

func init() {
	log.Init(log.Config{
		Level: "ERROR",
	})
}

type benchmark struct {
	seed     int64
	store    string
	mtHeight uint8
	initSize int
	postSize int
}

func randBytes(r *rand.Rand, length int) []byte {
	key := make([]byte, length)
	r.Read(key) //nolint: gosec
	return key
}

func initAddresses(count int) []common.Address {
	res := make([]common.Address, count)
	r := rand.New(rand.NewSource(12345678)) //nolint: gosec
	for i := 0; i < count; i++ {
		res[i] = common.BytesToAddress(randBytes(r, 32))
	}
	return res
}

func randDeposit(r *rand.Rand, depositCnt uint, blockID uint64, networkID int) *etherman.Deposit {
	return &etherman.Deposit{
		LeafType:           0,
		OriginalNetwork:    networks[0],
		OriginalAddress:    common.Address{},
		Amount:             big.NewInt(1000000000000),
		DestinationNetwork: networks[1-networkID],
		DestinationAddress: addresses[rand.Intn(len(addresses))], //nolint: gosec
		DepositCount:       depositCnt,
		BlockID:            blockID,
		NetworkID:          networks[networkID],
		Metadata:           randBytes(r, 20),
	}
}

func initServer(b *testing.B, bench benchmark) *bridgectrl.BridgeController {
	b.StopTimer()
	r := rand.New(rand.NewSource(bench.seed)) //nolint: gosec
	bt, store, err := operations.RunMockServer(bench.store, bench.mtHeight, networks)
	require.NoError(b, err)
	b.StartTimer()
	counts := []uint{0, 0}
	for i := 0; i < bench.initSize+bench.postSize; i++ {
		networkID := rand.Intn(2) //nolint: gosec
		dbTx, err := store.BeginDBTransaction(context.Background())
		require.NoError(b, err)
		id, err := store.AddBlock(context.TODO(), &etherman.Block{
			BlockNumber: uint64(i),
			BlockHash:   utils.GenerateRandomHash(),
			ParentHash:  utils.GenerateRandomHash(),
		}, dbTx)
		require.NoError(b, err)
		deposit := randDeposit(r, counts[networkID], id, networkID)
		counts[networkID]++
		depositID, err := store.AddDeposit(context.TODO(), deposit, dbTx)
		require.NoError(b, err)
		require.NoError(b, bt.AddDeposit(deposit, depositID, dbTx))
		if i > bench.initSize {
			require.NoError(b, store.Commit(context.TODO(), dbTx))
			continue
		}
		var roots [2][]byte
		roots[0], err = bt.GetExitRoot(0, dbTx)
		require.NoError(b, err)
		roots[1], err = bt.GetExitRoot(1, dbTx)
		require.NoError(b, err)

		if networkID == 0 {
			err = store.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				GlobalExitRoot: bridgectrl.Hash(common.BytesToHash(roots[0]), common.BytesToHash(roots[1])),
				ExitRoots:      []common.Hash{common.BytesToHash(roots[0]), common.BytesToHash(roots[1])},
				BlockID:        id,
			}, dbTx)
		} else {
			var isUpdated bool
			isUpdated, err = store.AddTrustedGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				GlobalExitRoot: bridgectrl.Hash(common.BytesToHash(roots[0]), common.BytesToHash(roots[1])),
				ExitRoots:      []common.Hash{common.BytesToHash(roots[0]), common.BytesToHash(roots[1])},
			}, dbTx)
			require.True(b, isUpdated)
		}
		require.NoError(b, err)
		err = store.AddClaim(context.TODO(), &etherman.Claim{
			Index:              deposit.DepositCount,
			OriginalNetwork:    deposit.OriginalNetwork,
			Amount:             deposit.Amount,
			NetworkID:          deposit.DestinationNetwork,
			DestinationAddress: deposit.DestinationAddress,
			BlockID:            id,
		}, dbTx)
		require.NoError(b, err)
		require.NoError(b, store.Commit(context.TODO(), dbTx))
	}

	require.NoError(b, store.UpdateDepositsStatusForTesting(context.TODO(), nil))

	return bt
}

func addDeposit(b *testing.B, bench benchmark) {
	b.StopTimer()
	r := rand.New(rand.NewSource(bench.seed)) //nolint: gosec
	bt, store, err := operations.RunMockServer(bench.store, bench.mtHeight, networks)
	require.NoError(b, err)
	var (
		deposits   []*etherman.Deposit
		depositIDs []uint64
	)
	for i := 0; i < bench.initSize; i++ {
		deposit := randDeposit(r, uint(i), 0, 0)
		depositID, err := store.AddDeposit(context.TODO(), deposit, nil)
		require.NoError(b, err)
		deposits = append(deposits, deposit)
		depositIDs = append(depositIDs, depositID)
	}
	b.StartTimer()
	for i := 0; i < bench.initSize; i++ {
		dbTx, err := store.BeginDBTransaction(context.Background())
		require.NoError(b, err)
		require.NoError(b, bt.AddDeposit(deposits[i], depositIDs[i], dbTx))
		require.NoError(b, store.Commit(context.TODO(), dbTx))
	}
}

func runSuite(b *testing.B, bench benchmark) {
	url := "http://localhost:8080"
	err := operations.WaitRestHealthy(url)
	require.NoError(b, err)

	restClient := client.NewRestClient(url)
	version, err := restClient.GetVersion()
	require.NoError(b, err)
	require.Equal(b, "v1", version)

	offset := uint(0)
	limit := uint(100)

	b.ResetTimer()
	b.Run("get bridges endpoint", func(sub *testing.B) {
		sub.ReportAllocs()
		deposits, totalCount, err := restClient.GetBridges(addresses[0].Hex(), offset, limit)
		require.NoError(sub, err)
		require.Greater(sub, len(deposits), 0)
		require.Greater(sub, totalCount, uint64(0))
	})

	b.Run("get merkle proof endpoint", func(sub *testing.B) {
		sub.ReportAllocs()
		sub.StopTimer()
		deposits, _, err := restClient.GetBridges(addresses[1].Hex(), offset, limit)
		require.NoError(sub, err)
		sub.StartTimer()
		deposit := deposits[len(deposits)-1]
		require.True(sub, deposit.ReadyForClaim)

		proof, err := restClient.GetMerkleProof(deposit.NetworkId, deposit.DepositCnt)
		require.NoError(sub, err)
		require.Equal(sub, len(proof.MerkleProof), int(bench.mtHeight))
	})
}

func BenchmarkApiSmallTest(b *testing.B) {
	benchmarks := []benchmark{
		{480459882, "postgres", 32, 500, 100},
	}
	for _, bench := range benchmarks {
		prefix := fmt.Sprintf("test-add-deposit-to-merkle-tree-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			sub.ReportAllocs()
			addDeposit(sub, bench)
		})
		prefix = fmt.Sprintf("test-sync-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			sub.ReportAllocs()
			initServer(sub, bench)
		})
		prefix = fmt.Sprintf("test-api-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			runSuite(sub, bench)
		})
	}
}

func BenchmarkApiMediumTest(b *testing.B) {
	benchmarks := []benchmark{
		{518638291, "postgres", 32, 10000, 100},
	}
	for _, bench := range benchmarks {
		prefix := fmt.Sprintf("test-sync-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			initServer(sub, bench)
		})
		prefix = fmt.Sprintf("test-api-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			runSuite(sub, bench)
		})
	}
}

func BenchmarkApiLargeTest(b *testing.B) {
	benchmarks := []benchmark{
		{756509998, "postgres", 32, 100000, 100},
	}
	for _, bench := range benchmarks {
		prefix := fmt.Sprintf("test-sync-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			initServer(sub, bench)
		})
		prefix = fmt.Sprintf("test-api-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			runSuite(sub, bench)
		})
	}
}
