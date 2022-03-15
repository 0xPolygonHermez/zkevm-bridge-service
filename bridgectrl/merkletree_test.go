package bridgectrl

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
	testHeight = 32
)

type leafVectorRaw struct {
	Leaves        []string   `json:"leaves"`
	ExpectedRoots []string   `json:"expectedRoots"`
	ExpectedCount uint       `json:"expectedCount"`
	Prooves       [][]string `json:"prooves"`
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

	var testVectors []leafVectorRaw
	err = json.Unmarshal(data, &testVectors)
	require.NoError(t, err)

	dbCfg := pgstorage.NewConfigFromEnv()

	ctx := context.WithValue(context.Background(), contextKeyNetwork, uint8(1)) //nolint

	for ti, testVector := range testVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			err = pgstorage.InitOrReset(dbCfg)
			require.NoError(t, err)

			store, err := pgstorage.NewPostgresStorage(dbCfg)
			require.NoError(t, err)

			mt, err := NewMerkleTree(ctx, store, uint8(testHeight))
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.ExpectedRoots[0])

			for i := 0; i < len(testVector.Leaves); i++ {
				// convert string to byte array
				leafValue, err := formatBytes32String(testVector.Leaves[i])
				require.NoError(t, err)

				err = mt.addLeaf(ctx, leafValue)
				require.NoError(t, err)

				assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.ExpectedRoots[i+1])

				prooves, err := mt.getSiblings(ctx, uint(i), mt.root)
				require.NoError(t, err)

				proofStrings := make([]string, 0)
				for i := 0; i < len(prooves); i++ {
					proofStrings = append(proofStrings, hex.EncodeToString(prooves[i][:]))
				}
				assert.Equal(t, proofStrings, testVector.Prooves[i])
			}
			assert.Equal(t, mt.count, testVector.ExpectedCount)

			for i := len(testVector.Leaves) - 1; i >= 0; i-- {
				err := mt.resetLeaf(ctx, uint(i))
				require.NoError(t, err)
				assert.Equal(t, hex.EncodeToString(mt.root[:]), testVector.ExpectedRoots[i])
				assert.Equal(t, mt.count, uint(i))
			}
		})
	}
}
