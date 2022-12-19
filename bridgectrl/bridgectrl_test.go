package bridgectrl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/vectors"
	"github.com/ethereum/go-ethereum/common"
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

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	id, err := store.AddBlock(context.TODO(), &etherman.Block{
		BlockNumber: 0,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
		ParentHash:  common.Hash{},
	}, nil)
	require.NoError(t, err)

	bt, err := NewBridgeController(cfg, []uint{0, 1000}, store, store)
	require.NoError(t, err)

	ctx := context.TODO()
	t.Run("Test adding deposit for the bridge tree", func(t *testing.T) {
		for i, testVector := range testVectors {
			amount, _ := new(big.Int).SetString(testVector.Amount, 0)
			deposit := &etherman.Deposit{
				LeafType:           0,
				OriginalNetwork:    testVector.OriginalNetwork,
				OriginalAddress:    common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(i),
				Metadata:           common.FromHex(testVector.Metadata),
			}
			leafHash := hashDeposit(deposit)
			assert.Equal(t, testVector.ExpectedHash, hex.EncodeToString(leafHash[:]))

			err = bt.AddDeposit(deposit)
			require.NoError(t, err)

			// test reorg
			orgRoot, err := bt.exitTrees[0].store.GetRoot(ctx, uint(i+1), 0, nil)
			require.NoError(t, err)
			err = bt.ReorgMT(uint(i), testVectors[i].OriginalNetwork)
			require.NoError(t, err)
			err = bt.AddDeposit(deposit)
			require.NoError(t, err)
			newRoot, err := bt.exitTrees[0].store.GetRoot(ctx, uint(i+1), 0, nil)
			require.NoError(t, err)
			assert.Equal(t, orgRoot, newRoot)

			err = store.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				BlockNumber:    uint64(i + 1),
				GlobalExitRoot: hash(common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])),
				ExitRoots:      []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
				BlockID:        id,
			}, nil)
			require.NoError(t, err)

			err = store.AddTrustedGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				BlockNumber:    0,
				GlobalExitRoot: hash(common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])),
				ExitRoots:      []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
				BlockID:        id,
			}, nil)
			require.NoError(t, err)
		}

		for i, testVector := range testVectors {
			proof, _, err := bt.GetClaim(testVector.OriginalNetwork, uint(i))
			require.NoError(t, err)
			require.Equal(t, len(proof), 32)
		}
	})
}
