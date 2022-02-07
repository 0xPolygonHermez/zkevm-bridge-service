package etherman

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-bridge/log"
)

var (
	depositEventSignatureHash              = crypto.Keccak256Hash([]byte("DepositEvent(address,uint256,uint32,address,uint32)"))
	updateGlobalExitRootEventSignatureHash = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	claimEventSignatureHash                = crypto.Keccak256Hash([]byte("WithdrawEvent(uint64,uint32,address,uint256,address)"))

	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

type ethClienter interface {
	ethereum.ChainReader
	ethereum.LogFilterer
	ethereum.TransactionReader
}

type ClientEtherMan struct {
	EtherClient ethClienter
	Bridge      *bridge.Bridge
	SCAddresses []common.Address
}

type EtherMan interface {
}

// NewEtherman creates a new etherman
func NewEtherman(cfg Config, bridgeAddr common.Address) (*ClientEtherMan, error) {
	// TODO: PoEAddr can be got from bridge smc. Son only bridge smc is required
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.URL, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := bridge.NewBridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	var scAddresses []common.Address
	scAddresses = append(scAddresses, bridgeAddr)

	return &ClientEtherMan{EtherClient: ethClient, Bridge: bridge, SCAddresses: scAddresses}, nil
}

// GetRollupInfoByBlock function retrieves the Rollup information that are included in a specific ethereum block
func (etherMan *ClientEtherMan) GetBridigeInfoByBlock(ctx context.Context, blockNumber uint64, blockHash *common.Hash) ([]Block, map[common.Hash][]Order, error) {
	// First filter query
	var blockNumBigInt *big.Int
	if blockHash == nil {
		blockNumBigInt = new(big.Int).SetUint64(blockNumber)
	}
	query := ethereum.FilterQuery{
		BlockHash: blockHash,
		FromBlock: blockNumBigInt,
		ToBlock:   blockNumBigInt,
		Addresses: etherMan.SCAddresses,
	}
	blocks, order, err := etherMan.readEvents(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	return blocks, order, nil
}

// GetRollupInfoByBlockRange function retrieves the Rollup information that are included in all this ethereum blocks
// from block x to block y
func (etherMan *ClientEtherMan) GetBridgeInfoByBlockRange(ctx context.Context, fromBlock uint64, toBlock *uint64) ([]Block, map[common.Hash][]Order, error) {
	// First filter query
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		Addresses: etherMan.SCAddresses,
	}
	if toBlock != nil {
		query.ToBlock = new(big.Int).SetUint64(*toBlock)
	}
	blocks, order, err := etherMan.readEvents(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	return blocks, order, nil
}

func (etherMan *ClientEtherMan) readEvents(ctx context.Context, query ethereum.FilterQuery) ([]Block, map[common.Hash][]Order, error) {
	// logs, err := etherMan.EtherClient.FilterLogs(ctx, query)
	// if err != nil {
	// 	return []Block{}, nil, err
	// }

	// blockOrder := make(map[common.Hash][]Order)
	// blocks := make(map[common.Hash]Block)
	// var blockKeys []common.Hash

	// for _, vLog := range logs {
	// 	block, err := etherMan.processEvent(ctx, vLog)
	// }

	return nil, nil, nil
}

func (etherMan *ClientEtherMan) processEvent(ctx context.Context, vLog types.Log) (*Block, error) {
	return nil, nil
}
