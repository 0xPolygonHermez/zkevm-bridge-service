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
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
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

	bt, err := NewBridgeController(cfg, []uint{0, 1000}, store)
	require.NoError(t, err)

	ctx := context.TODO()
	t.Run("Test adding deposit for the bridge tree", func(t *testing.T) {
		for i, testVector := range testVectors {
			block := &etherman.Block{
				BlockNumber: uint64(i + 1),
				BlockHash:   utils.GenerateRandomHash(),
				ParentHash:  common.Hash{},
			}
			blockID, err := store.AddBlock(context.TODO(), block, nil)
			require.NoError(t, err)
			amount, _ := new(big.Int).SetString(testVector.Amount, 0)
			deposit := &etherman.Deposit{
				LeafType:           0,
				OriginalNetwork:    testVector.OriginalNetwork,
				OriginalAddress:    common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockID:            blockID,
				DepositCount:       uint(i),
				Metadata:           common.FromHex(testVector.Metadata),
			}
			leafHash := hashDeposit(deposit)
			assert.Equal(t, testVector.ExpectedHash, hex.EncodeToString(leafHash[:]))
			depositID, err := store.AddDeposit(ctx, deposit, nil)
			require.NoError(t, err)
			err = bt.AddDeposit(deposit, depositID, nil)
			require.NoError(t, err)

			// test reorg
			orgRoot, err := bt.exitTrees[0].store.GetRoot(ctx, uint(i), 0, nil)
			require.NoError(t, err)
			require.NoError(t, store.Reset(ctx, uint64(i), deposit.NetworkID, nil))
			err = bt.ReorgMT(uint(i), testVectors[i].OriginalNetwork, nil)
			require.NoError(t, err)
			blockID, err = store.AddBlock(context.TODO(), block, nil)
			require.NoError(t, err)
			deposit.BlockID = blockID
			depositID, err = store.AddDeposit(ctx, deposit, nil)
			require.NoError(t, err)
			err = bt.AddDeposit(deposit, depositID, nil)
			require.NoError(t, err)
			newRoot, err := bt.exitTrees[0].store.GetRoot(ctx, uint(i), 0, nil)
			require.NoError(t, err)
			assert.Equal(t, orgRoot, newRoot)

			var roots [2][]byte
			roots[0], err = bt.exitTrees[0].getRoot(ctx, nil)
			require.NoError(t, err)
			roots[1], err = bt.exitTrees[1].getRoot(ctx, nil)
			require.NoError(t, err)

			err = store.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				BlockNumber:    uint64(i + 1),
				GlobalExitRoot: Hash(common.BytesToHash(roots[0]), common.BytesToHash(roots[1])),
				ExitRoots:      []common.Hash{common.BytesToHash(roots[0]), common.BytesToHash(roots[1])},
				BlockID:        blockID,
			}, nil)
			require.NoError(t, err)

			isUpdated, err := store.AddTrustedGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
				BlockNumber:    0,
				GlobalExitRoot: Hash(common.BytesToHash(roots[0]), common.BytesToHash(roots[1])),
				ExitRoots:      []common.Hash{common.BytesToHash(roots[0]), common.BytesToHash(roots[1])},
				BlockID:        blockID,
			}, nil)
			require.True(t, isUpdated)
			require.NoError(t, err)
		}
	})
}
