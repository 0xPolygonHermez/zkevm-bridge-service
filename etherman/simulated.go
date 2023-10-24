package etherman

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/mockpolygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/mockverifier"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	mockbridge "github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewSimulatedEtherman creates an etherman that uses a simulated blockchain. It's important to notice that the ChainID of the auth
// must be 1337. The address that holds the auth will have an initial balance of 10 ETH
func NewSimulatedEtherman(cfg Config, auth *bind.TransactOpts) (*Client, *backends.SimulatedBackend, common.Address, *mockbridge.Polygonzkevmbridge, *polygonzkevm.Polygonzkevm, error) {
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
	const polDecimalPlaces = 18
	totalSupply, _ := new(big.Int).SetString("10000000000000000000000000000", 10) //nolint:gomnd
	polAddr, _, polContract, err := pol.DeployPol(auth, client, "Pol Token", "POL", polDecimalPlaces, totalSupply)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	rollupVerifierAddr, _, _, err := mockverifier.DeployMockverifier(auth, client)
	if err != nil {
		return nil, nil, common.Address{}, nil, nil, err
	}
	nonce, err := client.PendingNonceAt(context.TODO(), auth.From)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	const posBridge = 1
	calculatedBridgeAddr := crypto.CreateAddress(auth.From, nonce+posBridge)
	const posRollupManager = 2
	calculatedRollupManagerAddr := crypto.CreateAddress(auth.From, nonce+posRollupManager)
	genesis := common.HexToHash("0xfd3434cd8f67e59d73488a2b8da242dd1f02849ea5dd99f0ca22c836c3d5b4a9") // Random value. Needs to be different to 0x0
	exitManagerAddr, _, globalExitRoot, err := polygonzkevmglobalexitroot.DeployPolygonzkevmglobalexitroot(auth, client, calculatedRollupManagerAddr, calculatedBridgeAddr)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	bridgeAddr, _, mockbr, err := mockbridge.DeployPolygonzkevmbridge(auth, client)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}

	mockRollupManagerAddr, _, mockRollupManager, err := mockpolygonrollupmanager.DeployMockpolygonrollupmanager(auth, client, exitManagerAddr, polAddr, bridgeAddr)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	if calculatedRollupManagerAddr != mockRollupManagerAddr {
		return nil, nil, common.Address{}, nil, nil, fmt.Errorf("RollupManagerAddr (%s) is different from the expected contract address (%s)",
			mockRollupManagerAddr.String(), calculatedRollupManagerAddr.String())
	}
	initZkevmAddr, _, _, err := polygonzkevm.DeployPolygonzkevm(auth, client, exitManagerAddr, polAddr, bridgeAddr, mockRollupManagerAddr)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	_, err = mockbr.Initialize(auth, 0, common.Address{}, 0, exitManagerAddr, mockRollupManagerAddr)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}

	_, err = mockRollupManager.InitializeMock(auth, auth.From, 10000, 10000, auth.From, auth.From, auth.From) //nolint:gomnd
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	_, err = mockRollupManager.AddNewRollupType(auth, initZkevmAddr, rollupVerifierAddr, 5, 0, genesis, "PolygonZkEvm Rollup") //nolint:gomnd
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	client.Commit()

	br, err := polygonzkevmbridge.NewPolygonzkevmbridge(bridgeAddr, client)
	if err != nil {
		return nil, nil, common.Address{}, nil, nil, err
	}

	rollUpTypeID, err := mockRollupManager.RollupTypeCount(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	var zkevmChainID uint64 = 100
	_, err = mockRollupManager.CreateNewRollup(auth, rollUpTypeID, zkevmChainID, auth.From, auth.From, common.Address{}, 0, "http://localhost", "PolygonZkEvm Rollup")
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	client.Commit()

	rollupID, err := mockRollupManager.ChainIDToRollupID(&bind.CallOpts{Pending: false}, zkevmChainID)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	rollupData, err := mockRollupManager.RollupIDToRollupData(&bind.CallOpts{Pending: false}, rollupID)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	zkevmAddr := rollupData.RollupContract

	if calculatedBridgeAddr != bridgeAddr {
		return nil, nil, common.Address{}, nil, nil, fmt.Errorf("bridgeAddr (%s) is different from the expected contract address (%s)",
			bridgeAddr.String(), calculatedBridgeAddr.String())
	}

	rollupManager, err := polygonrollupmanager.NewPolygonrollupmanager(mockRollupManagerAddr, client)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}

	trueZkevm, err := polygonzkevm.NewPolygonzkevm(zkevmAddr, client) //nolint
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}

	// Approve the bridge and zkevm to spend 10000 pol tokens.
	approvedAmount, _ := new(big.Int).SetString("10000000000000000000000", 10) //nolint:gomnd
	_, err = polContract.Approve(auth, bridgeAddr, approvedAmount)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	_, err = polContract.Approve(auth, zkevmAddr, approvedAmount)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}

	_, err = trueZkevm.ActivateForceBatches(auth)
	if err != nil {
		log.Error("error: ", err)
		return nil, nil, common.Address{}, nil, nil, err
	}
	client.Commit()

	r, _ := trueZkevm.IsForcedBatchAllowed(&bind.CallOpts{Pending: false})
	log.Debug("IsforcedBatch: ", r)

	client.Commit()
	return &Client{EtherClient: client, PolygonBridge: br, PolygonZkEVMGlobalExitRoot: globalExitRoot, PolygonRollupManager: rollupManager, SCAddresses: []common.Address{exitManagerAddr, bridgeAddr, mockRollupManagerAddr}}, client, polAddr, mockbr, trueZkevm, nil
}
