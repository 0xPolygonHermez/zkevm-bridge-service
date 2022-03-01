package etherman

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hermeznetwork/hermez-core/log"
	"github.com/hermeznetwork/hermez-core/test/vectors"
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
func newTestingEnv() (ethman *ClientEtherMan, commit func(), maticAddr common.Address, auth *bind.TransactOpts) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err = bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}
	ethman, commit, maticAddr, err = NewSimulatedEtherman(Config{}, auth)
	if err != nil {
		log.Fatal(err)
	}

	return ethman, commit, maticAddr, auth
}

func TestBridgeEvents(t *testing.T) {
	// Set up testing environment
	etherman, commit, maticAddr, auth := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Deposit funds
	amount := big.NewInt(9000000000000000000)
	var destNetwork uint32 = 1 // 0 is reserved to mainnet. This variable is set in the smc
	destinationAddr := common.HexToAddress("0x61A1d716a74fb45d29f148C6C20A2eccabaFD753")
	_, err = etherman.Bridge.Bridge(auth, maticAddr, amount, destNetwork, destinationAddr)
	require.NoError(t, err)

	// Mine the tx in a block
	commit()

	block, order, err := etherman.GetBridgeInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
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
	mainnetExitRoot := block[0].GlobalExitRoots[0].MainnetExitRoot
	rollupExitRoot := block[0].GlobalExitRoots[0].RollupExitRoot
	globalExitRootNum := block[0].GlobalExitRoots[0].GlobalExitRootNum

	destNetwork = 1
	_, err = etherman.Bridge.Claim(auth, maticAddr, big.NewInt(1000000000000000000), destNetwork,
		network, auth.From, smtProof, index, globalExitRootNum, mainnetExitRoot, rollupExitRoot)
	require.NoError(t, err)

	// Mine the tx in a block
	commit()

	//Read claim event
	initBlock, err = etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	block, order, err = etherman.GetBridgeInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
	require.NoError(t, err)
	assert.Equal(t, TokensOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, ClaimsOrder, order[block[0].BlockHash][1].Name)
	assert.Equal(t, big.NewInt(1000000000000000000), block[0].Claims[0].Amount)
	assert.Equal(t, uint64(3), block[0].BlockNumber)
	assert.NotEqual(t, common.Address{}, block[0].Claims[0].Token)
	assert.Equal(t, auth.From, block[0].Claims[0].DestinationAddress)
	assert.Equal(t, uint(0), block[0].Claims[0].Index)
	assert.Equal(t, uint(1), block[0].Claims[0].OriginalNetwork)
	assert.Equal(t, uint64(3), block[0].Claims[0].BlockNumber)
}

func TestSCEvents(t *testing.T) {
	// Set up testing environment
	etherman, commit, _, auth := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	callDataTestCases := readTests()

	//prepare txs
	dHex := strings.Replace(callDataTestCases[1].BatchL2Data, "0x", "", -1)
	data, err := hex.DecodeString(dHex)
	require.NoError(t, err)

	//send propose batch l1 tx
	matic := new(big.Int)
	matic, ok := matic.SetString(callDataTestCases[1].MaticAmount, 10)
	if !ok {
		log.Fatal("error decoding maticAmount")
	}
	_, err = etherman.PoE.SendBatch(auth, data, matic)
	require.NoError(t, err)

	//prepare txs
	dHex = strings.Replace(callDataTestCases[0].BatchL2Data, "0x", "", -1)
	data, err = hex.DecodeString(dHex)
	require.NoError(t, err)

	matic, err = etherman.PoE.CalculateSequencerCollateral(&bind.CallOpts{Pending: false})
	require.NoError(t, err)
	matic.Add(matic, big.NewInt(1000000000000000000))
	_, err = etherman.PoE.SendBatch(auth, data, matic)
	require.NoError(t, err)

	// Mine the tx in a block
	commit()

	// Now read the event
	finalBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	finalBlockNumber := finalBlock.NumberU64()
	block, _, err := etherman.GetBridgeInfoByBlockRange(ctx, initBlock.NumberU64(), &finalBlockNumber)
	require.NoError(t, err)

	assert.Equal(t, big.NewInt(1000), block[0].Batches[0].ChainID)
	assert.NotEqual(t, common.Address{}, block[0].Batches[0].Sequencer)
	log.Debugf("Block Received with new sendBatch that contains the GlobalExitRoot %s\n", block[0].Batches[0].GlobalExitRoot.String())
}

func readTests() []vectors.TxEventsSendBatchTestCase {
	// Load test vectors
	txEventsSendBatchTestCases, err := vectors.LoadTxEventsSendBatchTestCases("../test/vectors/smc-txevents-sendbatch-test-vector.json")
	if err != nil {
		log.Fatal(err)
	}
	return txEventsSendBatchTestCases
}
