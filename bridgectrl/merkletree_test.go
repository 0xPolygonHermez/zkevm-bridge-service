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
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
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
					DepositCount:       uint(i),
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
					DepositCount:       uint(li),
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

func TestUpdateMT(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/root-vectors.json")
	require.NoError(t, err)

	var mtTestVectors []vectors.MTRootVectorRaw
	err = json.Unmarshal(data, &mtTestVectors)
	require.NoError(t, err)
	for ti, testVector := range mtTestVectors {
		input := testVector.ExistingLeaves
		log.Debug("input: ", input)
		dbCfg := pgstorage.NewConfigFromEnv()
		ctx := context.Background()
		err := pgstorage.InitOrReset(dbCfg)
		require.NoError(t, err)

		store, err := pgstorage.NewPostgresStorage(dbCfg)
		require.NoError(t, err)

		mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
		require.NoError(t, err)

		amount, result := new(big.Int).SetString(testVector.NewLeaf.Amount, 0)
		require.True(t, result)
		for i := 0; i <= ti; i++ {
			deposit := &etherman.Deposit{
				OriginalNetwork:    testVector.NewLeaf.OriginalNetwork,
				OriginalAddress:    common.HexToAddress(testVector.NewLeaf.TokenAddress),
				Amount:             amount,
				DestinationNetwork: testVector.NewLeaf.DestinationNetwork,
				DestinationAddress: common.HexToAddress(testVector.NewLeaf.DestinationAddress),
				BlockNumber:        0,
				DepositCount:       uint(i),
				Metadata:           common.FromHex(testVector.NewLeaf.Metadata),
			}
			_, err := store.AddDeposit(ctx, deposit, nil)
			require.NoError(t, err)
		}

		var leaves [][KeyLen]byte
		for _, v := range input {
			var res [KeyLen]byte
			copy(res[:], common.Hex2Bytes(v[2:]))
			leaves = append(leaves, res)
		}

		depositID := uint64(len(leaves))
		if depositID != 0 {
			err = mt.updateLeaf(ctx, depositID, leaves, nil)
			require.NoError(t, err)
			// Check root
			newRoot, err := mt.getRoot(ctx, nil)
			require.NoError(t, err)
			require.Equal(t, testVector.CurrentRoot[2:], hex.EncodeToString(newRoot[:]))
		}

		var res [KeyLen]byte
		copy(res[:], common.Hex2Bytes(testVector.NewLeaf.CurrentHash[2:]))
		leaves = append(leaves, res)
		depositID = uint64(len(leaves))
		err = mt.updateLeaf(ctx, depositID, leaves, nil)
		require.NoError(t, err)
		// Check new root
		newRoot, err := mt.getRoot(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, testVector.NewRoot[2:], hex.EncodeToString(newRoot[:]))
	}
}

func TestGetLeaves(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/fullmt-vector.sql")
	require.NoError(t, err)
	dbCfg := pgstorage.NewConfigFromEnv()
	ctx := context.Background()
	err = pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	_, err = store.Exec(ctx, string(data))
	require.NoError(t, err)

	mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
	require.NoError(t, err)
	leaves, err := mt.getLeaves(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 26, len(leaves))
	log.Debug("leaves: %+v", leaves)
}

func TestBuildMTRootAndStore(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/root-vectors.json")
	require.NoError(t, err)

	var mtTestVectors []vectors.MTRootVectorRaw
	err = json.Unmarshal(data, &mtTestVectors)
	require.NoError(t, err)
	for _, testVector := range mtTestVectors {
		input := testVector.ExistingLeaves
		log.Debug("input: ", input)
		dbCfg := pgstorage.NewConfigFromEnv()
		ctx := context.Background()
		err := pgstorage.InitOrReset(dbCfg)
		require.NoError(t, err)

		store, err := pgstorage.NewPostgresStorage(dbCfg)
		require.NoError(t, err)

		mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
		require.NoError(t, err)

		var leaves [][KeyLen]byte
		for _, v := range input {
			var res [KeyLen]byte
			copy(res[:], common.Hex2Bytes(v[2:]))
			leaves = append(leaves, res)
		}

		if len(leaves) != 0 {
			root, err := mt.buildMTRoot(leaves)
			require.NoError(t, err)
			require.Equal(t, testVector.CurrentRoot, root.String())
		}

		var res [KeyLen]byte
		copy(res[:], common.Hex2Bytes(testVector.NewLeaf.CurrentHash[2:]))
		leaves = append(leaves, res)
		newRoot, err := mt.buildMTRoot(leaves)
		require.NoError(t, err)
		require.Equal(t, testVector.NewRoot, newRoot.String())

		// Insert values into db
		var blockNumber uint64
		err = mt.storeLeaves(ctx, leaves, blockNumber, nil)
		require.NoError(t, err)
		result, err := mt.store.GetRollupExitLeavesByRoot(ctx, newRoot, nil)
		require.NoError(t, err)
		for i := range leaves {
			require.Equal(t, len(leaves), len(result))
			require.Equal(t, leaves[i][:], result[i].Leaf.Bytes())
			require.Equal(t, newRoot, result[i].Root)
			require.Equal(t, uint(i+1), result[i].RollupId)
		}
	}
}

func TestComputeSiblings(t *testing.T) {
	data, err := os.ReadFile("test/vectors/src/mt-bridge/fullmt-vector.sql")
	require.NoError(t, err)
	dbCfg := pgstorage.NewConfigFromEnv()
	ctx := context.Background()
	err = pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)

	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	_, err = store.Exec(ctx, string(data))
	require.NoError(t, err)

	mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
	require.NoError(t, err)
	leaves, err := mt.getLeaves(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 26, len(leaves))
	siblings, root, err := ComputeSiblings(1, leaves, mt.height)
	require.NoError(t, err)
	require.Equal(t, "0x4ed479841384358f765966486782abb598ece1d4f834a22474050d66a18ad296", root.String())
	expectedProof := []string{"0x83fc198de31e1b2b1a8212d2430fbb7766c13d9ad305637dea3759065606475d", "0x2815e0bbb1ec18b8b1bc64454a86d072e12ee5d43bb559b44059e01edff0af7a", "0x7fb6cc0f2120368a845cf435da7102ff6e369280f787bc51b8a989fc178f7252", "0x407db5edcdc0ddd4f7327f208f46db40c4c4dbcc46c94a757e1d1654acbd8b72", "0xce2cdd1ef2e87e82264532285998ff37024404ab3a2b77b50eb1ad856ae83e14", "0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d", "0x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968", "0xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83", "0x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af", "0xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0", "0xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5", "0xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892", "0x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c", "0xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb", "0x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc", "0xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2", "0x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f", "0xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a", "0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0", "0xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0", "0xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2", "0xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9", "0x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377", "0x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652", "0xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef", "0x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d", "0xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0", "0x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e", "0x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e", "0x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322", "0x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735", "0x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9"}
	for i := 0; i < len(siblings); i++ {
		require.Equal(t, expectedProof[i], "0x"+hex.EncodeToString(siblings[i][:]))
	}
}

func TestCheckMerkleProof(t *testing.T) {
	expectedRoot := common.HexToHash("0x5ba002329b53c11a2f1dfe90b11e031771842056cf2125b43da8103c199dcd7f")
	var index uint
	var height uint8 = 32
	amount, _ := big.NewInt(0).SetString("10000000000000000000", 0)
	deposit := &etherman.Deposit{
		OriginalNetwork:    0,
		OriginalAddress:    common.Address{},
		Amount:             amount,
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4"),
		BlockNumber:        0,
		DepositCount:       0,
		Metadata:           []byte{},
	}
	leafHash := hashDeposit(deposit)
	smtProof := [][KeyLen]byte{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.HexToHash("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
		common.HexToHash("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.HexToHash("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
		common.HexToHash("0x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968"),
		common.HexToHash("0xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83"),
		common.HexToHash("0x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af"),
		common.HexToHash("0xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0"),
		common.HexToHash("0xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5"),
		common.HexToHash("0xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892"),
		common.HexToHash("0x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c"),
		common.HexToHash("0xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb"),
		common.HexToHash("0x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc"),
		common.HexToHash("0xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2"),
		common.HexToHash("0x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f"),
		common.HexToHash("0xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a"),
		common.HexToHash("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"),
		common.HexToHash("0xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0"),
		common.HexToHash("0xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2"),
		common.HexToHash("0xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9"),
		common.HexToHash("0x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377"),
		common.HexToHash("0x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652"),
		common.HexToHash("0xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef"),
		common.HexToHash("0x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d"),
		common.HexToHash("0xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0"),
		common.HexToHash("0x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e"),
		common.HexToHash("0x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e"),
		common.HexToHash("0x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322"),
		common.HexToHash("0x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735"),
		common.HexToHash("0x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9"),
	}
	root := calculateRoot(leafHash, smtProof, index, height)
	assert.Equal(t, expectedRoot, root)
}

func TestCheckMerkleProof2(t *testing.T) {
	expectedLeafHash := common.HexToHash("0x697a56b92100081c2637a9d162509380ab75ec163c69e2ce0ebe1c444977c5e0")
	expectedRollup1Root := common.HexToHash("0x7c942bb17191a0c14d50dc079873bc7d84922be5c9a5307aa58b8b34d70cd67b")
	expectedRollupsTreeRoot := common.HexToHash("0x8e00abaf690edf420ba30016e4573a805300e89d634714007ae98519294cc58d")

	var index uint
	var height uint8 = 32
	amount, _ := big.NewInt(0).SetString("90000000000000000", 0)
	deposit := &etherman.Deposit{
		OriginalNetwork:    0,
		OriginalAddress:    common.Address{},
		Amount:             amount,
		DestinationNetwork: 2,
		DestinationAddress: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		BlockNumber:        3673,
		DepositCount:       0,
		Metadata:           []byte{},
	}
	leafBytes := hashDeposit(deposit)
	leafHash := common.BytesToHash(leafBytes[:])
	t.Log("leafHash: ", leafHash)
	assert.Equal(t, expectedLeafHash, leafHash)
	smtProof := [][KeyLen]byte{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.HexToHash("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
		common.HexToHash("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.HexToHash("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
		common.HexToHash("0x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968"),
		common.HexToHash("0xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83"),
		common.HexToHash("0x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af"),
		common.HexToHash("0xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0"),
		common.HexToHash("0xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5"),
		common.HexToHash("0xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892"),
		common.HexToHash("0x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c"),
		common.HexToHash("0xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb"),
		common.HexToHash("0x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc"),
		common.HexToHash("0xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2"),
		common.HexToHash("0x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f"),
		common.HexToHash("0xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a"),
		common.HexToHash("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"),
		common.HexToHash("0xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0"),
		common.HexToHash("0xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2"),
		common.HexToHash("0xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9"),
		common.HexToHash("0x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377"),
		common.HexToHash("0x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652"),
		common.HexToHash("0xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef"),
		common.HexToHash("0x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d"),
		common.HexToHash("0xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0"),
		common.HexToHash("0x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e"),
		common.HexToHash("0x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e"),
		common.HexToHash("0x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322"),
		common.HexToHash("0x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735"),
		common.HexToHash("0x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9"),
	}
	root := calculateRoot(leafHash, smtProof, index, height)
	t.Log("root: ", root)
	assert.Equal(t, expectedRollup1Root, root)

	leafHash2 := expectedRollup1Root
	smtProof2 := [][KeyLen]byte{
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"),
		common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		common.HexToHash("0x21ddb9a356815c3fac1026b6dec5df3124afbadb485c9ba5a3e3398a04b7ba85"),
		common.HexToHash("0xe58769b32a1beaf1ea27375a44095a0d1fb664ce2dd358e7fcbfb78c26a19344"),
		common.HexToHash("0x0eb01ebfc9ed27500cd4dfc979272d1f0913cc9f66540d7e8005811109e1cf2d"),
		common.HexToHash("0x887c22bd8750d34016ac3c66b5ff102dacdd73f6b014e710b51e8022af9a1968"),
		common.HexToHash("0xffd70157e48063fc33c97a050f7f640233bf646cc98d9524c6b92bcf3ab56f83"),
		common.HexToHash("0x9867cc5f7f196b93bae1e27e6320742445d290f2263827498b54fec539f756af"),
		common.HexToHash("0xcefad4e508c098b9a7e1d8feb19955fb02ba9675585078710969d3440f5054e0"),
		common.HexToHash("0xf9dc3e7fe016e050eff260334f18a5d4fe391d82092319f5964f2e2eb7c1c3a5"),
		common.HexToHash("0xf8b13a49e282f609c317a833fb8d976d11517c571d1221a265d25af778ecf892"),
		common.HexToHash("0x3490c6ceeb450aecdc82e28293031d10c7d73bf85e57bf041a97360aa2c5d99c"),
		common.HexToHash("0xc1df82d9c4b87413eae2ef048f94b4d3554cea73d92b0f7af96e0271c691e2bb"),
		common.HexToHash("0x5c67add7c6caf302256adedf7ab114da0acfe870d449a3a489f781d659e8becc"),
		common.HexToHash("0xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2"),
		common.HexToHash("0x2733e50f526ec2fa19a22b31e8ed50f23cd1fdf94c9154ed3a7609a2f1ff981f"),
		common.HexToHash("0xe1d3b5c807b281e4683cc6d6315cf95b9ade8641defcb32372f1c126e398ef7a"),
		common.HexToHash("0x5a2dce0a8a7f68bb74560f8f71837c2c2ebbcbf7fffb42ae1896f13f7c7479a0"),
		common.HexToHash("0xb46a28b6f55540f89444f63de0378e3d121be09e06cc9ded1c20e65876d36aa0"),
		common.HexToHash("0xc65e9645644786b620e2dd2ad648ddfcbf4a7e5b1a3a4ecfe7f64667a3f0b7e2"),
		common.HexToHash("0xf4418588ed35a2458cffeb39b93d26f18d2ab13bdce6aee58e7b99359ec2dfd9"),
		common.HexToHash("0x5a9c16dc00d6ef18b7933a6f8dc65ccb55667138776f7dea101070dc8796e377"),
		common.HexToHash("0x4df84f40ae0c8229d0d6069e5c8f39a7c299677a09d367fc7b05e3bc380ee652"),
		common.HexToHash("0xcdc72595f74c7b1043d0e1ffbab734648c838dfb0527d971b602bc216c9619ef"),
		common.HexToHash("0x0abf5ac974a1ed57f4050aa510dd9c74f508277b39d7973bb2dfccc5eeb0618d"),
		common.HexToHash("0xb8cd74046ff337f0a7bf2c8e03e10f642c1886798d71806ab1e888d9e5ee87d0"),
		common.HexToHash("0x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e"),
		common.HexToHash("0x662ee4dd2dd7b2bc707961b1e646c4047669dcb6584f0d8d770daf5d7e7deb2e"),
		common.HexToHash("0x388ab20e2573d171a88108e79d820e98f26c0b84aa8b2f4aa4968dbb818ea322"),
		common.HexToHash("0x93237c50ba75ee485f4c22adf2f741400bdf8d6a9cc7df7ecae576221665d735"),
		common.HexToHash("0x8448818bb4ae4562849e949e17ac16e0be16688e156b5cf15e098c627c0056a9"),
	}
	root2 := calculateRoot(leafHash2, smtProof2, index, height)
	t.Log("rollupsExitRoot: ", root2)
	assert.Equal(t, expectedRollupsTreeRoot, root2)
}

func TestPerformanceComputeRoot(t *testing.T) {
	ctx := context.Background()
	dbCfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)
	store, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)
	mt, err := NewMerkleTree(ctx, store, uint8(32), 0)
	require.NoError(t, err)
	var leaves [][KeyLen]byte
	initTime := time.Now().Unix()
	log.Debug("Init creating leaves: ", initTime)
	for i := 0; i < 10000000; i++ {
		leaves = append(leaves, common.Hash{})
	}
	log.Debug("End creating leaves: ", time.Now().Unix()-initTime)
	initTime = time.Now().Unix()
	log.Debug("Init computing root: ", initTime)
	_, err = mt.buildMTRoot(leaves)
	require.NoError(t, err)
	log.Debug("End creating leaves: ", time.Now().Unix()-initTime)
}
