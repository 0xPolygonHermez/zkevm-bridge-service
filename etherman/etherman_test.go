package etherman

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hermeznetwork/hermez-core/log"
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
	assert.Equal(t, DepositsOrder, order[block[1].BlockHash][0].Name)
	assert.Equal(t, GlobalExitRootsOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, GlobalExitRootsOrder, order[block[1].BlockHash][1].Name)
	assert.Equal(t, uint64(2), block[1].BlockNumber)
	assert.Equal(t, big.NewInt(9000000000000000000), block[1].Deposits[0].Amount)
	assert.Equal(t, uint(1), block[1].Deposits[0].DestinationNetwork)
	assert.Equal(t, destinationAddr, block[1].Deposits[0].DestinationAddress)
	assert.Equal(t, 1, len(block[0].GlobalExitRoots))
	assert.Equal(t, 1, len(block[1].GlobalExitRoots))

	//Claim funds
	var (
		network  uint32
		smtProof [][32]byte
		index    uint64
	)
	mainnetExitRoot := block[1].GlobalExitRoots[0].MainnetExitRoot
	rollupExitRoot := block[1].GlobalExitRoots[0].RollupExitRoot

	_, err = etherman.Bridge.Claim(auth, maticAddr, big.NewInt(1000000000000000000), network,
		network, auth.From, smtProof, index, big.NewInt(2), mainnetExitRoot, rollupExitRoot)
	require.NoError(t, err)

	// Mine the tx in a block
	commit()

	//Read claim event
	initBlock, err = etherman.EtherClient.BlockByNumber(ctx, nil)
	require.NoError(t, err)
	block, order, err = etherman.GetBridgeInfoByBlockRange(ctx, initBlock.NumberU64(), nil)
	require.NoError(t, err)
	assert.Equal(t, ClaimsOrder, order[block[0].BlockHash][0].Name)
	assert.Equal(t, big.NewInt(1000000000000000000), block[0].Claims[0].Amount)
	assert.Equal(t, uint64(3), block[0].BlockNumber)
	assert.NotEqual(t, common.Address{}, block[0].Claims[0].Token)
	assert.Equal(t, auth.From, block[0].Claims[0].DestinationAddress)
	assert.Equal(t, uint64(0), block[0].Claims[0].Index)
	assert.Equal(t, uint(0), block[0].Claims[0].OriginalNetwork)
	assert.Equal(t, uint64(3), block[0].Claims[0].BlockNumber)
}
