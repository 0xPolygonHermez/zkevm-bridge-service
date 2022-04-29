package bridgectrl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func formatBytes32String(text string) ([KeyLen]byte, error) {
	bText, err := hex.DecodeString(text)
	if err != nil {
		return [KeyLen]byte{}, err
	}

	if len(bText) > 32 {
		return [KeyLen]byte{}, fmt.Errorf("text is more than 32 bytes long")
	}
	var res [KeyLen]byte
	copy(res[:], bText)
	return res, nil
}

func TestLeafHash(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/leaf-vectors.json")
	require.NoError(t, err)

	var leafVectors []vectors.DepositVectorRaw
	err = json.Unmarshal(data, &leafVectors)
	require.NoError(t, err)

	for ti, testVector := range leafVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			amount, err := new(big.Int).SetString(testVector.Amount, 0)
			require.True(t, err)

			deposit := &etherman.Deposit{
				OriginalNetwork:    testVector.OriginalNetwork,
				TokenAddress:       common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(ti + 1),
			}
			leafHash := hashDeposit(deposit)
			assert.Equal(t, testVector.ExpectedHash[2:], hex.EncodeToString(leafHash[:]))
		})
	}
}

func TestMTAddLeaf(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/root-vectors.json")
	require.NoError(t, err)

	var mtTestVectors []vectors.MTRootVectorRaw
	err = json.Unmarshal(data, &mtTestVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()

	ctx := context.WithValue(context.Background(), contextKeyNetwork, uint8(1)) //nolint

	for ti, testVector := range mtTestVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			err = pgstorage.InitOrReset(dbCfg)
			require.NoError(t, err)

			store, err := pgstorage.NewPostgresStorage(dbCfg, 0)
			require.NoError(t, err)

			mt, err := NewMerkleTree(ctx, store, uint8(32))
			require.NoError(t, err)

			for _, leaf := range testVector.ExistingLeaves {
				leafValue, err := formatBytes32String(leaf[2:])
				require.NoError(t, err)

				err = mt.addLeaf(ctx, leafValue)
				require.NoError(t, err)
			}

			assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.CurrentRoot[2:])

			amount, result := new(big.Int).SetString(testVector.NewLeaf.Amount, 0)
			require.True(t, result)

			deposit := &etherman.Deposit{
				OriginalNetwork:    testVector.NewLeaf.OriginalNetwork,
				TokenAddress:       common.HexToAddress(testVector.NewLeaf.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.NewLeaf.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.NewLeaf.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(ti + 1),
			}
			leafHash := hashDeposit(deposit)
			err = mt.addLeaf(ctx, leafHash)
			require.NoError(t, err)

			assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.NewRoot[2:])
		})
	}
}

func TestMTGetProof(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/claim-vectors.json")
	require.NoError(t, err)

	var mtTestVectors []vectors.MTClaimVectorRaw
	err = json.Unmarshal(data, &mtTestVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()

	ctx := context.WithValue(context.Background(), contextKeyNetwork, uint8(1)) //nolint

	for ti, testVector := range mtTestVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			err = pgstorage.InitOrReset(dbCfg)
			require.NoError(t, err)

			store, err := pgstorage.NewPostgresStorage(dbCfg, 0)
			require.NoError(t, err)

			mt, err := NewMerkleTree(ctx, store, uint8(32))
			require.NoError(t, err)

			for li, leaf := range testVector.Deposits {
				amount, result := new(big.Int).SetString(leaf.Amount, 0)
				require.True(t, result)

				deposit := &etherman.Deposit{
					OriginalNetwork:    leaf.OriginalNetwork,
					TokenAddress:       common.HexToAddress(leaf.TokenAddress),
					Amount:             amount,
					DestinationNetwork: leaf.DestinationNetwork,
					DestinationAddress: common.HexToAddress(leaf.DestinationAddress),
					BlockNumber:        0,
					DepositCount:       uint(li + 1),
				}

				leafHash := hashDeposit(deposit)
				err = mt.addLeaf(ctx, leafHash)
				require.NoError(t, err)
			}

			assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.ExpectedRoot[2:])

			prooves, err := mt.getSiblings(ctx, testVector.Index, mt.root)
			require.NoError(t, err)

			for i, proof := range prooves {
				assert.Equal(t, hex.EncodeToString(proof[:]), testVector.MerkleProof[i][2:])
			}
		})
	}
}
