package ethermanv2

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	mockbridge "github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/bridge"
	"github.com/0xPolygonHermez/zkevm-node/ethermanv2/smartcontracts/proofofefficiency"
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

//This function prepare the blockchain, the wallet with funds and deploy the smc
func newTestingEnv() (ethman *Client, ethBackend *backends.SimulatedBackend, maticAddr common.Address, bridge *mockbridge.Bridge) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}
	ethman, ethBackend, maticAddr, bridge, err = NewSimulatedEtherman(Config{}, auth)
	if err != nil {
		log.Fatal(err)
	}
	return ethman, ethBackend, maticAddr, bridge
}

func TestGEREvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, _, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	amount := big.NewInt(1000000000000000)
	a := etherman.auth
	a.Value = amount
	_, err = etherman.Bridge.Bridge(a, common.Address{}, 1, etherman.auth.From, amount)
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, _, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)

	assert.Equal(t, big.NewInt(1), blocks[0].GlobalExitRoots[0].GlobalExitRootNum)
	assert.NotEqual(t, common.Hash{}, blocks[0].GlobalExitRoots[0].ExitRoots[0])
	assert.Equal(t, common.Hash{}, blocks[0].GlobalExitRoots[0].ExitRoots[1])
}

func TestSequencedBatchesEvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, _, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Make a bridge tx
	a := etherman.auth
	a.Value = big.NewInt(1000000000000000)
	_, err = etherman.Bridge.Bridge(a, common.Address{}, 1, a.From, a.Value)
	require.NoError(t, err)
	ethBackend.Commit()
	a.Value = big.NewInt(0)

	// Get the last ger
	ger, err := etherman.GlobalExitRootManager.GetLastGlobalExitRoot(nil)
	require.NoError(t, err)

	amount, err := etherman.PoE.CalculateForceProverFee(&bind.CallOpts{Pending: false})
	require.NoError(t, err)
	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	data, err := hex.DecodeString(rawTxs)
	require.NoError(t, err)
	_, err = etherman.PoE.ForceBatch(etherman.auth, data, amount)
	require.NoError(t, err)
	ethBackend.Commit()

	currentBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	var sequences []proofofefficiency.ProofOfEfficiencyBatchData
	sequences = append(sequences, proofofefficiency.ProofOfEfficiencyBatchData{
		GlobalExitRoot:        ger,
		Timestamp:             currentBlock.Time() - 1,
		ForceBatchesTimestamp: []uint64{30},
		Transactions:          common.Hex2Bytes(rawTxs),
	})
	sequences = append(sequences, proofofefficiency.ProofOfEfficiencyBatchData{
		GlobalExitRoot:        ger,
		Timestamp:             currentBlock.Time() + 1,
		ForceBatchesTimestamp: []uint64{},
		Transactions:          common.Hex2Bytes(rawTxs),
	})
	_, err = etherman.PoE.SequenceBatches(etherman.auth, sequences)
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)
	assert.Equal(t, 3, len(blocks))
	assert.Equal(t, 1, len(blocks[2].SequencedBatches))
	assert.Equal(t, common.Hex2Bytes(rawTxs), blocks[2].SequencedBatches[0][1].Transactions)
	assert.Equal(t, currentBlock.Time()-1, blocks[2].SequencedBatches[0][0].Timestamp)
	assert.Equal(t, ger, blocks[2].SequencedBatches[0][0].GlobalExitRoot)
	assert.Equal(t, []uint64{currentBlock.Time()}, blocks[2].SequencedBatches[0][0].ForceBatchesTimestamp)
	assert.Equal(t, 0, order[blocks[2].BlockHash][0].Pos)
}

func TestVerifyBatchEvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, _, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()

	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	tx := proofofefficiency.ProofOfEfficiencyBatchData{
		GlobalExitRoot:        common.Hash{},
		Timestamp:             initBlock.Time(),
		ForceBatchesTimestamp: []uint64{},
		Transactions:          common.Hex2Bytes(rawTxs),
	}
	_, err = etherman.PoE.SequenceBatches(etherman.auth, []proofofefficiency.ProofOfEfficiencyBatchData{tx})
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	var (
		proofA = [2]*big.Int{big.NewInt(1), big.NewInt(1)}
		proofC = [2]*big.Int{big.NewInt(1), big.NewInt(1)}
		proofB = [2][2]*big.Int{proofC, proofC}
	)
	_, err = etherman.PoE.VerifyBatch(etherman.auth, common.Hash{}, common.Hash{}, 1, proofA, proofB, proofC)
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)

	assert.Equal(t, uint64(3), blocks[1].BlockNumber)
	assert.Equal(t, uint64(1), blocks[1].VerifiedBatches[0].BatchNumber)
	assert.NotEqual(t, common.Address{}, blocks[1].VerifiedBatches[0].Aggregator)
	assert.NotEqual(t, common.Hash{}, blocks[1].VerifiedBatches[0].TxHash)
	assert.Equal(t, GlobalExitRootsOrder, order[blocks[1].BlockHash][0].Name)
	assert.Equal(t, VerifyBatchOrder, order[blocks[1].BlockHash][1].Name)
	assert.Equal(t, 0, order[blocks[1].BlockHash][0].Pos)
	assert.Equal(t, 0, order[blocks[1].BlockHash][1].Pos)
}

func TestSequenceForceBatchesEvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, _, _ := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	amount, err := etherman.PoE.CalculateForceProverFee(&bind.CallOpts{Pending: false})
	require.NoError(t, err)
	rawTxs := "f84901843b9aca00827b0c945fbdb2315678afecb367f032d93f642f64180aa380a46057361d00000000000000000000000000000000000000000000000000000000000000048203e9808073efe1fa2d3e27f26f32208550ea9b0274d49050b816cadab05a771f4275d0242fd5d92b3fb89575c070e6c930587c520ee65a3aa8cfe382fcad20421bf51d621c"
	data, err := hex.DecodeString(rawTxs)
	require.NoError(t, err)
	_, err = etherman.PoE.ForceBatch(etherman.auth, data, amount)
	require.NoError(t, err)
	ethBackend.Commit()

	err = ethBackend.AdjustTime((24*7 + 1) * time.Hour)
	require.NoError(t, err)
	ethBackend.Commit()

	_, err = etherman.PoE.SequenceForceBatches(etherman.auth, 1)
	require.NoError(t, err)
	ethBackend.Commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	blocks, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)
	assert.Equal(t, uint64(4), blocks[1].BlockNumber)
	assert.Equal(t, uint64(1), blocks[1].SequencedForceBatches[0].LastBatchSequenced)
	assert.Equal(t, uint64(1), blocks[1].SequencedForceBatches[0].ForceBatchNumber)
	assert.Equal(t, 0, order[blocks[1].BlockHash][0].Pos)
}

func TestBridgeEvents(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, maticAddr, bridge := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Deposit funds
	amount := big.NewInt(9000000000000000000)
	var destNetwork uint32 = 1 // 0 is reserved to mainnet. This variable is set in the smc
	destinationAddr := common.HexToAddress("0x61A1d716a74fb45d29f148C6C20A2eccabaFD753")
	_, err = bridge.Bridge(etherman.auth, maticAddr, destNetwork, destinationAddr, amount)
	require.NoError(t, err)

	// Mine the tx in a block
	ethBackend.Commit()

	block, order, err := etherman.GetRollupInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
	require.NoError(t, err)
	assert.Equal(t, DepositsOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, GlobalExitRootsOrder, order[block[0].BlockHash][1].Name)
	assert.Equal(t, uint64(2), block[0].BlockNumber)
	assert.Equal(t, big.NewInt(9000000000000000000), block[0].Deposits[0].Amount)
	assert.Equal(t, uint(destNetwork), block[0].Deposits[0].DestinationNetwork)
	assert.Equal(t, destinationAddr, block[0].Deposits[0].DestinationAddress)
	assert.Equal(t, 1, len(block[0].GlobalExitRoots))

	//Claim funds
	var (
		network  uint32
		smtProof [][32]byte
		index    uint32
	)
	mainnetExitRoot := block[0].GlobalExitRoots[0].ExitRoots[0]
	rollupExitRoot := block[0].GlobalExitRoots[0].ExitRoots[1]
	// globalExitRootNum := block[0].GlobalExitRoots[0].GlobalExitRootNum

	destNetwork = 1
	_, err = bridge.Claim(etherman.auth, smtProof, index, mainnetExitRoot, rollupExitRoot,
		network, maticAddr, destNetwork, etherman.auth.From, big.NewInt(1000000000000000000), []byte{})
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
	assert.Equal(t, uint64(3), block[0].BlockNumber)
	assert.NotEqual(t, common.Address{}, block[0].Claims[0].Token)
	assert.Equal(t, etherman.auth.From, block[0].Claims[0].DestinationAddress)
	assert.Equal(t, uint(0), block[0].Claims[0].Index)
	assert.Equal(t, uint(0), block[0].Claims[0].OriginalNetwork)
	assert.Equal(t, uint64(3), block[0].Claims[0].BlockNumber)
}
