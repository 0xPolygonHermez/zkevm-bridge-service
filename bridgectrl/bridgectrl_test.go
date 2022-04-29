package bridgectrl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Change dir to project root
	// This is important because we have relative paths to files containing test vectors
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestBridgeTree(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/deposit-raw.json")
	require.NoError(t, err)

	var testVectors []vectors.DepositVectorRaw
	err = json.Unmarshal(data, &testVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()
	err = pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)

	cfg := Config{
		Height: uint8(32), //nolint:gomnd
		Store:  "postgres",
	}

	store, err := pgstorage.NewPostgresStorage(dbCfg, 0)
	require.NoError(t, err)

	id, err := store.AddBlock(context.TODO(), &etherman.Block{
		BlockNumber:     0,
		BlockHash:       common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		ParentHash:      common.Hash{},
		Batches:         []etherman.Batch{},
		Deposits:        []etherman.Deposit{},
		GlobalExitRoots: []etherman.GlobalExitRoot{},
		Claims:          []etherman.Claim{},
		Tokens:          []etherman.TokenWrapped{},
	})
	require.NoError(t, err)

	bt, err := NewBridgeController(cfg, []uint{0, 1000}, store, store)
	require.NoError(t, err)

	t.Run("Test adding deposit for the bridge tree", func(t *testing.T) {
		for i, testVector := range testVectors {
			amount, _ := new(big.Int).SetString(testVector.Amount, 10)
			deposit := &etherman.Deposit{
				OriginalNetwork:    testVector.OriginalNetwork,
				TokenAddress:       common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(i + 1),
			}
			leafHash := hashDeposit(deposit)
			assert.Equal(t, testVector.ExpectedHash, hex.EncodeToString(leafHash[:]))

			err = bt.AddDeposit(deposit)
			require.NoError(t, err)

			err = store.AddExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				BlockNumber:       0,
				GlobalExitRootNum: big.NewInt(int64(i)),
				ExitRoots:         []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
				BlockID:           id,
			})
			require.NoError(t, err)
		}

		for i, testVector := range testVectors {
			merkleProof, globalExitRoot, err := bt.GetClaim(testVector.OriginalNetwork, uint(i+1))
			require.NoError(t, err)

			log.Println(globalExitRoot)
			log.Println(merkleProof)
		}

		for i := len(testVectors) - 1; i >= 0; i-- {
			err := bt.ReorgMT(uint(i+1), testVectors[i].OriginalNetwork)
			require.NoError(t, err)
		}
	})
}
