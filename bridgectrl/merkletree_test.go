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
				OriginalAddress:    common.HexToAddress(testVector.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(ti + 1),
				Metadata:           common.FromHex(testVector.Metadata),
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
	ctx := context.Background()

	for ti, testVector := range mtTestVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			err = pgstorage.InitOrReset(dbCfg)
			require.NoError(t, err)

			store, err := pgstorage.NewPostgresStorage(dbCfg)
			require.NoError(t, err)

			mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
			require.NoError(t, err)

			amount, result := new(big.Int).SetString(testVector.NewLeaf.Amount, 0)
			require.True(t, result)
			var (
				depositIDs []uint64
				deposit    *etherman.Deposit
			)
			for i := 0; i <= ti; i++ {
				deposit = &etherman.Deposit{
					OriginalNetwork:    testVector.NewLeaf.OriginalNetwork,
					OriginalAddress:    common.HexToAddress(testVector.NewLeaf.TokenAddress),
					Amount:             amount,
					DestinationNetwork: testVector.NewLeaf.DestinationNetwork,
					DestinationAddress: common.HexToAddress(testVector.NewLeaf.DestinationAddress),
					BlockNumber:        0,
					DepositCount:       uint(i + 1),
					Metadata:           common.FromHex(testVector.NewLeaf.Metadata),
				}
				depositID, err := store.AddDeposit(ctx, deposit, nil)
				require.NoError(t, err)
				depositIDs = append(depositIDs, depositID)
			}

			for i, leaf := range testVector.ExistingLeaves {
				leafValue, err := formatBytes32String(leaf[2:])
				require.NoError(t, err)

				err = mt.addLeaf(ctx, depositIDs[i], leafValue, uint(i), nil)
				require.NoError(t, err)
			}
			curRoot, err := mt.getRoot(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(curRoot), testVector.CurrentRoot[2:])

			leafHash := hashDeposit(deposit)
			err = mt.addLeaf(ctx, depositIDs[len(depositIDs)-1], leafHash, uint(len(testVector.ExistingLeaves)), nil)
			require.NoError(t, err)
			newRoot, err := mt.getRoot(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(newRoot), testVector.NewRoot[2:])
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
	ctx := context.Background()

	for ti, testVector := range mtTestVectors {
		t.Run(fmt.Sprintf("Test vector %d", ti), func(t *testing.T) {
			err = pgstorage.InitOrReset(dbCfg)
			require.NoError(t, err)

			store, err := pgstorage.NewPostgresStorage(dbCfg)
			require.NoError(t, err)

			mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
			require.NoError(t, err)
			var cur, sibling [KeyLen]byte
			for li, leaf := range testVector.Deposits {
				amount, result := new(big.Int).SetString(leaf.Amount, 0)
				require.True(t, result)
				block := &etherman.Block{
					BlockNumber: uint64(li + 1),
					BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fc"),
					ParentHash:  common.Hash{},
				}
				blockID, err := store.AddBlock(context.TODO(), block, nil)
				require.NoError(t, err)
				deposit := &etherman.Deposit{
					OriginalNetwork:    leaf.OriginalNetwork,
					OriginalAddress:    common.HexToAddress(leaf.TokenAddress),
					Amount:             amount,
					DestinationNetwork: leaf.DestinationNetwork,
					DestinationAddress: common.HexToAddress(leaf.DestinationAddress),
					BlockID:            blockID,
					DepositCount:       uint(li + 1),
					Metadata:           common.FromHex(leaf.Metadata),
				}
				depositID, err := store.AddDeposit(ctx, deposit, nil)
				require.NoError(t, err)
				leafHash := hashDeposit(deposit)
				if li == int(testVector.Index) {
					cur = leafHash
				}
				err = mt.addLeaf(ctx, depositID, leafHash, uint(li), nil)
				require.NoError(t, err)
			}
			root, err := mt.getRoot(ctx, nil)
			require.NoError(t, err)
			assert.Equal(t, hex.EncodeToString(root), testVector.ExpectedRoot[2:])

			for h := 0; h < int(mt.height); h++ {
				copy(sibling[:], common.FromHex(testVector.MerkleProof[h]))
				if testVector.Index&(1<<h) != 0 {
					cur = Hash(sibling, cur)
				} else {
					cur = Hash(cur, sibling)
				}
			}
			assert.Equal(t, hex.EncodeToString(cur[:]), testVector.ExpectedRoot[2:])
		})
	}
}
