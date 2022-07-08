package etherman

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/bridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/globalexitrootmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/matic"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/proofofefficiency"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewSimulatedEtherman creates an etherman that uses a simulated blockchain. It's important to notice that the ChainID of the auth
// must be 1337. The address that holds the auth will have an initial balance of 10 ETH
func NewSimulatedEtherman(cfg Config, auth *bind.TransactOpts) (etherman *ClientEtherMan, commit func(), maticAddr common.Address, err error) {
	// 10000000 ETH in wei
	balance, _ := new(big.Int).SetString("10000000000000000000000000", 10) //nolint:gomnd
	address := auth.From
	genesisAlloc := map[common.Address]core.GenesisAccount{
		address: {
			Balance: balance,
		},
	}
	blockGasLimit := uint64(999999999999999999) //nolint:gomnd
	client := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	// Deploy contracts
	totalSupply, _ := new(big.Int).SetString("10000000000000000000000000000", 10)                                //nolint:gomnd
	maticAddr, _, maticContract, err := matic.DeployMatic(auth, client, "Matic Token", "MATIC", 18, totalSupply) //nolint:gomnd
	if err != nil {
		return nil, nil, common.Address{}, err
	}
	rollupVerifierAddr := common.Address{}
	nonce, err := client.PendingNonceAt(context.TODO(), auth.From)
	if err != nil {
		return nil, nil, common.Address{}, err
	}
	calculatedBridgeAddr := crypto.CreateAddress(auth.From, nonce+1)
	const pos = 2
	calculatedPoEAddr := crypto.CreateAddress(auth.From, nonce+pos)
	var genesis [32]byte
	exitManagerAddr, _, exitManager, err := globalexitrootmanager.DeployGlobalexitrootmanager(auth, client, calculatedPoEAddr, calculatedBridgeAddr)
	if err != nil {
		return nil, nil, common.Address{}, err
	}
	bridgeAddr, _, bridge, err := bridge.DeployBridge(auth, client, 0, exitManagerAddr)
	if err != nil {
		return nil, nil, common.Address{}, err
	}
	poeAddr, _, poe, err := proofofefficiency.DeployProofofefficiency(auth, client, exitManagerAddr, maticAddr, rollupVerifierAddr, genesis)
	if err != nil {
		return nil, nil, common.Address{}, err
	}

	if calculatedBridgeAddr != bridgeAddr {
		return nil, nil, common.Address{}, fmt.Errorf("bridgeAddr (" + bridgeAddr.String() +
			") is different from the expected contract address (" + calculatedBridgeAddr.String() + ")")
	}
	if calculatedPoEAddr != poeAddr {
		return nil, nil, common.Address{}, fmt.Errorf("poeAddr (" + poeAddr.String() +
			") is different from the expected contract address (" + calculatedPoEAddr.String() + ")")
	}

	// Approve the bridge and poe to spend 10000 matic tokens
	approvedAmount, _ := new(big.Int).SetString("10000000000000000000000", 10) //nolint:gomnd
	_, err = maticContract.Approve(auth, bridgeAddr, approvedAmount)
	if err != nil {
		return nil, nil, common.Address{}, err
	}
	_, err = maticContract.Approve(auth, poeAddr, approvedAmount)
	if err != nil {
		return nil, nil, common.Address{}, err
	}

	client.Commit()
	return &ClientEtherMan{EtherClient: client, PoE: poe, Bridge: bridge, GlobalExitRootManager: exitManager, SCAddresses: []common.Address{poeAddr, bridgeAddr, exitManagerAddr}}, client.Commit, maticAddr, nil
}
