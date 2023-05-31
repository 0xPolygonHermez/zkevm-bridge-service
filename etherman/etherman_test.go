package etherman

import (
	"context"
	"math/big"
	"testing"

	mockbridge "github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/polygonzkevmbridge"
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
func newTestingEnv() (ethman *Client, ethBackend *backends.SimulatedBackend, auth *bind.TransactOpts, maticAddr common.Address, bridge *mockbridge.Polygonzkevmbridge) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	auth, err = bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(1337))
	if err != nil {
		log.Fatal(err)
	}
	ethman, ethBackend, maticAddr, bridge, err = NewSimulatedEtherman(Config{}, auth)
	if err != nil {
		log.Fatal(err)
	}
	return ethman, ethBackend, auth, maticAddr, bridge
}

func TestGEREvent(t *testing.T) {
	// Set up testing environment
	etherman, ethBackend, auth, _, _ := newTestingEnv()

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
	etherman, ethBackend, auth, maticAddr, bridge := newTestingEnv()

	// Read currentBlock
	ctx := context.Background()
	initBlock, err := etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Deposit funds
	amount := big.NewInt(9000000000000000000)
	var destNetwork uint32 = 1 // 0 is reserved to mainnet. This variable is set in the smc
	destinationAddr := common.HexToAddress("0x61A1d716a74fb45d29f148C6C20A2eccabaFD753")
	_, err = bridge.BridgeAsset(auth, destNetwork, destinationAddr, amount, maticAddr, true, []byte{})
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
		smtProof [32][32]byte
		index    uint32
	)
	mainnetExitRoot := block[0].GlobalExitRoots[0].ExitRoots[0]
	rollupExitRoot := block[0].GlobalExitRoots[0].ExitRoots[1]

	destNetwork = 1
	_, err = bridge.ClaimAsset(auth, smtProof, index, mainnetExitRoot, rollupExitRoot,
		network, maticAddr, destNetwork, auth.From, big.NewInt(1000000000000000000), []byte{})
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
	assert.NotEqual(t, common.Address{}, block[0].Claims[0].OriginalAddress)
	assert.Equal(t, auth.From, block[0].Claims[0].DestinationAddress)
	assert.Equal(t, uint(0), block[0].Claims[0].Index)
	assert.Equal(t, uint(0), block[0].Claims[0].OriginalNetwork)
	assert.Equal(t, uint64(3), block[0].Claims[0].BlockNumber)
}
