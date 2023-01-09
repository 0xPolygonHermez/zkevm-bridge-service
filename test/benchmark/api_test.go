package benchmark

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
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
	seed     uint
	store    string
	mtHeight uint8
	initSize int
}

func randBytes(length int) []byte {
	key := make([]byte, length)
	rand.Read(key)
	return key
}

func initAddresses(count int) []common.Address {
	res := make([]common.Address, count)
	for i := 0; i < count; i++ {
		res[i] = common.HexToAddress(string(randBytes(32)))
	}
	return res
}

func randDeposit(depositCnt uint, blockID uint64, networkID int) *etherman.Deposit {
	return &etherman.Deposit{
		LeafType:           0,
		OriginalNetwork:    networks[0],
		OriginalAddress:    common.Address{},
		Amount:             big.NewInt(1000000000000),
		DestinationNetwork: networks[1-networkID],
		DestinationAddress: addresses[rand.Intn(len(addresses))],
		DepositCount:       depositCnt,
		BlockID:            blockID,
		NetworkID:          networks[networkID],
		Metadata:           randBytes(20),
	}
}

func initServer(b *testing.B, bench benchmark) *bridgectrl.BridgeController {
	bt, store, err := operations.RunMockServer(bench.store, bench.mtHeight, networks)
	require.NoError(b, err)
	b.StartTimer()
	for i := 0; i < bench.initSize; i++ {
		id, err := store.AddBlock(context.TODO(), &etherman.Block{
			BlockNumber: uint64(i),
			BlockHash:   common.Hash{},
			ParentHash:  common.Hash{},
		}, nil)
		require.NoError(b, err)
		deposit := randDeposit(uint(i+1), id, 0)
		err = store.AddDeposit(context.TODO(), deposit, nil)
		require.NoError(b, err)
		bt.AddDeposit(deposit)
	}

	return bt
}

func runSuite(b *testing.B, bench benchmark) {
	b.StopTimer()
	_ = initServer(b, bench)

}

func BenchmarkApiTest(b *testing.B) {
	benchmarks := []benchmark{
		{10123452, "postgres", 32, 100},
		{10123452, "postgres", 32, 20},
	}
	for _, bench := range benchmarks {
		prefix := fmt.Sprintf("test-%s-%d", bench.store, bench.initSize)
		b.Run(prefix, func(sub *testing.B) {
			runSuite(sub, bench)
		})
	}
}
