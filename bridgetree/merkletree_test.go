package bridgetree

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	contextKeyTableName   = "postgres-table-name"
	contextValueTableName = "merkletree.mainnet"
	height                = 32
)

type testVectorRaw struct {
	Leaves         []string   `json:"leaves"`
	ExpectedRoots  []string   `json:"expectedRoots"`
	ExpectedCounts []uint64   `json:"expectedCounts"`
	Prooves        [][]string `json:"prooves"`
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

func formatBytes32String(text string) ([32]byte, error) {
	bText := []byte(text)
	if len(bText) > 31 {
		return [32]byte{}, fmt.Errorf("text is more than 31 bytes long")
	}
	var res [32]byte
	copy(res[:], bText)
	return res, nil
}

func TestMerkleTree(t *testing.T) {
	data, err := os.ReadFile("test/vectors/mt-raw.json")
	require.NoError(t, err)

	var testVectors []testVectorRaw
	err = json.Unmarshal(data, &testVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()
	err = pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), contextKeyTableName, contextValueTableName) // nolint

	for ti, testVector := range testVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			mt := NewMerkleTree(store, uint8(height))
			root, err := mt.getRoot(ctx)
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(root[:]), testVector.ExpectedRoots[0])

			for i := 0; i < len(testVector.Leaves); i++ {
				// convert string to byte array
				leafValue, err := formatBytes32String(testVector.Leaves[i])
				require.NoError(t, err)

				err = mt.addLeaf(ctx, leafValue)
				require.NoError(t, err)

				root, err := mt.getRoot(ctx)
				require.NoError(t, err)
				assert.Equal(t, hex.EncodeToString(root[:]), testVector.ExpectedRoots[i+1])

				prooves, err := mt.getProofTreeByIndex(ctx, uint64(i))
				require.NoError(t, err)
				proofStrings := make([]string, 0)

				for _, proof := range prooves {
					proofStrings = append(proofStrings, hex.EncodeToString(proof[:]))
				}
				assert.Equal(t, proofStrings, testVector.Prooves[i])
			}
			assert.Equal(t, mt.counts, testVector.ExpectedCounts)
		})
	}
}
