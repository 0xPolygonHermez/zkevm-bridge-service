package etherman

import (
	"context"
	"math/big"
	"testing"
	// "encoding/binary"

	mockbridge "github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	log.Init(log.Config{
		Level:   "debug",
		Outputs: []string{"stdout"},
	})
}

// This function prepare the blockchain, the wallet with funds and deploy the smc
func newTestingEnv() (*Client, *backends.SimulatedBackend, *bind.TransactOpts, common.Address, *mockbridge.Polygonzkevmbridge, *polygonzkevm.Polygonzkevm) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}
	ethman, ethBackend, polAddr, bridge, zkevm, err := NewSimulatedEtherman(Config{}, auth)
	if err != nil {
		log.Fatal(err)
	}
	return ethman, ethBackend, auth, polAddr, bridge, zkevm
}

func TestGEREvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, auth, _, _, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	amount := big.NewInt(1000000000000000)
	auth.Value = amount
	_, err = etherman.PolygonBridge.BridgeAsset(auth, 1, auth.From, amount, common.Address{}, true, []byte{})
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, _, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)

	assert.NotEqual(t, common.Hash{}, blocks[0].GlobalExitRoots[0].ExitRoots[0])
	assert.Equal(t, common.Hash{}, blocks[0].GlobalExitRoots[0].ExitRoots[1])
}

func TestBridgeEvents(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, auth, polAddr, bridge, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Deposit funds
	amount := big.NewInt(9000000000000000000)
	var destNetwork uint32 = 1 // 0 is reserved to mainnet. This variable is set in the smc
	destinationAddr := common.HexToAddress("0x61A1d716a74fb45d29f148C6C20A2eccabaFD753")
	_, err = bridge.BridgeAsset(auth, destNetwork, destinationAddr, amount, polAddr, true, []byte{})
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	block, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
	require.NoError(t, err)
	assert.Equal(t, DepositsOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, GlobalExitRootsOrder, order[block[0].BlockHash][1].Name)
	assert.Equal(t, uint64(5), block[0].BlockNumber)
	assert.Equal(t, big.NewInt(9000000000000000000), block[0].Deposits[0].Amount)
	assert.Equal(t, uint(destNetwork), block[0].Deposits[0].DestinationNetwork)
	assert.Equal(t, destinationAddr, block[0].Deposits[0].DestinationAddress)
	assert.Equal(t, 1, len(block[0].GlobalExitRoots))

	//Claim funds
	var (
		network  uint32
		smtProofLocalExitRoot, smtProofRollupExitRoot [32][32]byte
		globalIndex, _ = big.NewInt(0).SetString("18446744073709551650", 0)

	)
	mainnetExitRoot := block[0].GlobalExitRoots[0].ExitRoots[0]
	rollupExitRoot := block[0].GlobalExitRoots[0].ExitRoots[1]

	destNetwork = 1
	_, err = bridge.ClaimAsset(auth, smtProofLocalExitRoot, smtProofRollupExitRoot, globalIndex, mainnetExitRoot, rollupExitRoot,
		network, polAddr, destNetwork, auth.From, big.NewInt(1000000000000000000), []byte{})
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	//Read claim event
	initBlock, err = etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	block, order, err = etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
	require.NoError(t, err)
	assert.Equal(t, TokensOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, ClaimsOrder, order[block[0].BlockHash][1].Name)
	assert.Equal(t, big.NewInt(1000000000000000000), block[0].Claims[0].Amount)
	assert.Equal(t, uint64(6), block[0].BlockNumber)
	assert.NotEqual(t, common.Address{}, block[0].Claims[0].OriginalAddress)
	assert.Equal(t, auth.From, block[0].Claims[0].DestinationAddress)
	assert.Equal(t, uint(34), block[0].Claims[0].Index)
	assert.Equal(t, uint64(0), block[0].Claims[0].RollupIndex)
	assert.Equal(t, true, block[0].Claims[0].MainnetFlag)
	assert.Equal(t, uint(0), block[0].Claims[0].OriginalNetwork)
	assert.Equal(t, uint64(6), block[0].Claims[0].BlockNumber)
}

func TestDecodeGlobalIndex(t *testing.T) {
	globalIndex, _ := big.NewInt(0).SetString("4294967307", 0)

	var buf [32]byte
	gi := globalIndex.FillBytes(buf[:])
	for _, n := range gi {
        t.Logf("%08b ", n)
    }
	mainnetFlag, rollupIndex, localExitRootIndex, err := decodeGlobalIndex(globalIndex)
	require.NoError(t, err)
	assert.Equal(t, false, mainnetFlag)
	assert.Equal(t, uint64(1), rollupIndex)
	assert.Equal(t, uint64(11), localExitRootIndex)

	globalIndex, _ = big.NewInt(0).SetString("8589934604", 0)

	gi = globalIndex.FillBytes(buf[:])
	for _, n := range gi {
        t.Logf("%08b ", n)
    }
	mainnetFlag, rollupIndex, localExitRootIndex, err = decodeGlobalIndex(globalIndex)
	require.NoError(t, err)
	assert.Equal(t, false, mainnetFlag)
	assert.Equal(t, uint64(2), rollupIndex)
	assert.Equal(t, uint64(12), localExitRootIndex)

	globalIndex, _ = big.NewInt(0).SetString("18446744073709551627", 0)

	gi = globalIndex.FillBytes(buf[:])
	for _, n := range gi {
        t.Logf("%08b ", n)
    }
	mainnetFlag, rollupIndex, localExitRootIndex, err = decodeGlobalIndex(globalIndex)
	require.NoError(t, err)
	assert.Equal(t, true, mainnetFlag)
	assert.Equal(t, uint64(0), rollupIndex)
	assert.Equal(t, uint64(11), localExitRootIndex)

	globalIndex, _ = big.NewInt(0).SetString("18446744073709551616", 0)

	gi = globalIndex.FillBytes(buf[:])
	for _, n := range gi {
        t.Logf("%08b ", n)
    }
	mainnetFlag, rollupIndex, localExitRootIndex, err = decodeGlobalIndex(globalIndex)
	require.NoError(t, err)
	assert.Equal(t, true, mainnetFlag)
	assert.Equal(t, uint64(0), rollupIndex)
	assert.Equal(t, uint64(0), localExitRootIndex)
}

func TestVerifyBatchEvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, auth, _, _, zkevm := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()

	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	tx := polygonzkevm.PolygonRollupBaseBatchData{
		GlobalExitRoot:     common.Hash{},
		Timestamp:          initBlock.Time(),
		MinForcedTimestamp: 0,
		Transactions:       common.Hex2Bytes(rawTxs),
	}
	_, err = zkevm.SequenceBatches(auth, []polygonzkevm.PolygonRollupBaseBatchData{tx}, auth.From)
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	_, err = etherman.PolygonRollupManager.VerifyBatchesTrustedAggregator(auth, 1, uint64(0), uint64(0), uint64(1), [32]byte{}, [32]byte{}, auth.From, [24][32]byte{})
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)
	t.Logf("Blocks: %+v, \nOrder: %+v", blocks, order)
	assert.Equal(t, uint64(6), blocks[0].BlockNumber)
	assert.Equal(t, uint64(1), blocks[0].VerifiedBatches[0].BatchNumber)
	assert.NotEqual(t, common.Address{}, blocks[0].VerifiedBatches[0].Aggregator)
	assert.NotEqual(t, common.Hash{}, blocks[0].VerifiedBatches[0].TxHash)
	assert.Equal(t, GlobalExitRootsOrder, order[blocks[0].BlockHash][0].Name)
	assert.Equal(t, VerifyBatchOrder, order[blocks[0].BlockHash][1].Name)
	assert.Equal(t, 0, order[blocks[0].BlockHash][0].Pos)
	assert.Equal(t, 0, order[blocks[0].BlockHash][1].Pos)
}

func TestGenerateGlobalIndex(t *testing.T) {
	globalIndex, _ := big.NewInt(0).SetString("4294967307", 0)
	mainnetFlag, rollupIndex, localExitRootIndex := false, uint(1), uint(11)
	globalIndexGenerated := GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	t.Log("First test number:")
	for _, n := range globalIndexGenerated.Bytes() {
        t.Logf("%08b ", n)
    }
	assert.Equal(t, globalIndex, globalIndexGenerated)
	
	globalIndex, _ = big.NewInt(0).SetString("8589934604", 0)
	mainnetFlag, rollupIndex, localExitRootIndex = false, uint(2), uint(12)
	globalIndexGenerated = GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	t.Log("Second test number:")
	for _, n := range globalIndexGenerated.Bytes() {
        t.Logf("%08b ", n)
    }
	assert.Equal(t, globalIndex, globalIndexGenerated)

	globalIndex, _ = big.NewInt(0).SetString("18446744073709551627", 0)
	mainnetFlag, rollupIndex, localExitRootIndex = true, uint(0), uint(11)
	globalIndexGenerated = GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	t.Log("Third test number:")
	for _, n := range globalIndexGenerated.Bytes() {
        t.Logf("%08b ", n)
    }
	assert.Equal(t, globalIndex, globalIndexGenerated)
}