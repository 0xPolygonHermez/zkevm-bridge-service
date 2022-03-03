package bridgetree

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type depositVectorRaw struct {
	OriginalNetwork    uint   `json:"origNetwork"`
	TokenAddress       string `json:"tokenAddress"`
	Amount             string `json:"amount"`
	DestinationNetwork uint   `json:"destNetwork"`
	DestinationAddress string `json:"destAddress"`
	BlockNumber        uint64 `json:"blockNumber"`
	DepositCount       uint   `json:"depositCount"`
	ExpectedHash       string `json:"expectedHash"`
	ExpectedRoot       string `json:"expectedRoot"`
}

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
	data, err := os.ReadFile("test/vectors/mainnet-raw.json")
	require.NoError(t, err)

	var testVectors []depositVectorRaw
	err = json.Unmarshal(data, &testVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()
	err = pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)

	cfg := Config{
		Height: uint8(testHeight),
		Store:  "postgres",
	}

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	bt, err := NewBridgeTree(cfg, []uint64{0, 1000}, store, store)
	require.NoError(t, err)

	t.Run("Test adding deposit for the bridge tree", func(t *testing.T) {
		for _, testVector := range testVectors {
			amount, _ := new(big.Int).SetString(testVector.Amount, 10)
			deposit := &etherman.Deposit{
				OriginalNetwork:    testVector.OriginalNetwork,
				TokenAddress:       common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockNumber:        testVector.BlockNumber,
				DepositCount:       testVector.DepositCount,
			}
			leafHash := hashDeposit(deposit)
			assert.Equal(t, testVector.ExpectedHash, hex.EncodeToString(leafHash[:]))

			block := &etherman.Block{
				BlockNumber: testVector.BlockNumber,
				BlockHash:   common.Hash{},
			}
			err := bt.storage.AddBlock(context.Background(), block)
			require.NoError(t, err)
			err = bt.AddDeposit(deposit)
			require.NoError(t, err)

			assert.Equal(t, testVector.ExpectedRoot, hex.EncodeToString(bt.exitRootTrees[0].root[:]))
		}
	})

	t.Run("Test getting claims", func(t *testing.T) {
		for _, testVector := range testVectors {
			prooves := make([][KeyLen]byte, testHeight)
			deposit, globalExitRoot, err := bt.GetClaim(testVector.OriginalNetwork, testVector.DepositCount, prooves)
			require.NoError(t, err)

			log.Println(deposit)
			log.Println(globalExitRoot)
			log.Println(prooves)
		}
	})
}
