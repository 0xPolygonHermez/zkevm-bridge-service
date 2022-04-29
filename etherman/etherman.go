package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/globalexitrootmanager"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/proofofefficiency"
	"github.com/hermeznetwork/hermez-core/log"
)

var (
	newBatchEventSignatureHash             = crypto.Keccak256Hash([]byte("SendBatch(uint64,address,uint64,bytes32)"))
	consolidateBatchSignatureHash          = crypto.Keccak256Hash([]byte("VerifyBatch(uint64,address)"))
	newSequencerSignatureHash              = crypto.Keccak256Hash([]byte("RegisterSequencer(address,string,uint64)"))
	ownershipTransferredSignatureHash      = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	depositEventSignatureHash              = crypto.Keccak256Hash([]byte("BridgeEvent(address,uint256,uint32,uint32,address,uint32)"))
	updateGlobalExitRootEventSignatureHash = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(uint256,bytes32,bytes32)"))
	claimEventSignatureHash                = crypto.Keccak256Hash([]byte("ClaimEvent(uint32,uint32,address,uint256,address)"))
	newWrappedTokenEventSignatureHash      = crypto.Keccak256Hash([]byte("NewWrappedToken(uint32,address,address)"))

	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

type ethClienter interface {
	ethereum.ChainReader
	ethereum.LogFilterer
	ethereum.TransactionReader
}

// ClientEtherMan struct
type ClientEtherMan struct {
	EtherClient           ethClienter
	PoE                   *proofofefficiency.Proofofefficiency
	Bridge                *bridge.Bridge
	GlobalExitRootManager *globalexitrootmanager.Globalexitrootmanager
	SCAddresses           []common.Address
}

// NewEtherman creates a new etherman.
func NewEtherman(cfg Config, poeAddr common.Address, bridgeAddr common.Address, globalExitRootManAddr common.Address) (*ClientEtherMan, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.L1URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.L1URL, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := bridge.NewBridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	globalExitRoot, err := globalexitrootmanager.NewGlobalexitrootmanager(globalExitRootManAddr, ethClient)
	if err != nil {
		return nil, err
	}
	poe, err := proofofefficiency.NewProofofefficiency(poeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	scAddresses := []common.Address{poeAddr, bridgeAddr, globalExitRootManAddr}

	return &ClientEtherMan{EtherClient: ethClient, PoE: poe, Bridge: bridge, GlobalExitRootManager: globalExitRoot, SCAddresses: scAddresses}, nil
}

// NewL2Etherman creates a new etherman.
func NewL2Etherman(url string, bridgeAddr common.Address) (*ClientEtherMan, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", url, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := bridge.NewBridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	scAddresses := []common.Address{bridgeAddr}

	return &ClientEtherMan{EtherClient: ethClient, Bridge: bridge, SCAddresses: scAddresses}, nil
}

// GetBridgeInfoByBlockRange function retrieves the Bridge information that are included in all this ethereum blocks
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
	logs, err := etherMan.EtherClient.FilterLogs(ctx, query)
	if err != nil {
		return []Block{}, nil, err
	}

	blockOrder := make(map[common.Hash][]Order)
	blocks := make(map[common.Hash]Block)
	var blockKeys []common.Hash

	for _, vLog := range logs {
		block, err := etherMan.processEvent(ctx, vLog)
		if err != nil {
			log.Warnf("error processing event. Retrying... Error: %w. vLog: %+v", err, vLog)
			break
		}
		if block == nil {
			continue
		}
		if b, exists := blocks[block.BlockHash]; exists {
			if len(block.Batches) != 0 {
				b.Batches = append(blocks[block.BlockHash].Batches, block.Batches...)
				or := Order{
					Name: BatchesOrder,
					Pos:  len(b.Batches) - 1,
				}
				blockOrder[b.BlockHash] = append(blockOrder[b.BlockHash], or)
			}
			if len(block.Deposits) != 0 {
				b.Deposits = append(blocks[block.BlockHash].Deposits, block.Deposits...)
				or := Order{
					Name: DepositsOrder,
					Pos:  len(b.Deposits) - 1,
				}
				blockOrder[b.BlockHash] = append(blockOrder[b.BlockHash], or)
			}
			if len(block.GlobalExitRoots) != 0 {
				b.GlobalExitRoots = append(blocks[block.BlockHash].GlobalExitRoots, block.GlobalExitRoots...)
				or := Order{
					Name: GlobalExitRootsOrder,
					Pos:  len(b.GlobalExitRoots) - 1,
				}
				blockOrder[b.BlockHash] = append(blockOrder[b.BlockHash], or)
			}
			if len(block.Claims) != 0 {
				b.Claims = append(blocks[block.BlockHash].Claims, block.Claims...)
				or := Order{
					Name: ClaimsOrder,
					Pos:  len(b.Claims) - 1,
				}
				blockOrder[b.BlockHash] = append(blockOrder[b.BlockHash], or)
			}
			if len(block.Tokens) != 0 {
				b.Tokens = append(blocks[block.BlockHash].Tokens, block.Tokens...)
				or := Order{
					Name: TokensOrder,
					Pos:  len(b.Tokens) - 1,
				}
				blockOrder[b.BlockHash] = append(blockOrder[b.BlockHash], or)
			}
			blocks[block.BlockHash] = b
		} else {
			if len(block.Batches) != 0 {
				or := Order{
					Name: BatchesOrder,
					Pos:  len(block.Batches) - 1,
				}
				blockOrder[block.BlockHash] = append(blockOrder[block.BlockHash], or)
			}
			if len(block.Deposits) != 0 {
				or := Order{
					Name: DepositsOrder,
					Pos:  len(block.Deposits) - 1,
				}
				blockOrder[block.BlockHash] = append(blockOrder[block.BlockHash], or)
			}
			if len(block.GlobalExitRoots) != 0 {
				or := Order{
					Name: GlobalExitRootsOrder,
					Pos:  len(block.GlobalExitRoots) - 1,
				}
				blockOrder[block.BlockHash] = append(blockOrder[block.BlockHash], or)
			}
			if len(block.Claims) != 0 {
				or := Order{
					Name: ClaimsOrder,
					Pos:  len(block.Claims) - 1,
				}
				blockOrder[block.BlockHash] = append(blockOrder[block.BlockHash], or)
			}
			if len(block.Tokens) != 0 {
				or := Order{
					Name: TokensOrder,
					Pos:  len(block.Tokens) - 1,
				}
				blockOrder[block.BlockHash] = append(blockOrder[block.BlockHash], or)
			}
			blocks[block.BlockHash] = *block
			blockKeys = append(blockKeys, block.BlockHash)
		}
	}
	var blockArr []Block
	for _, hash := range blockKeys {
		blockArr = append(blockArr, blocks[hash])
	}
	return blockArr, blockOrder, nil
}

func (etherMan *ClientEtherMan) processEvent(ctx context.Context, vLog types.Log) (*Block, error) {
	switch vLog.Topics[0] {
	case newBatchEventSignatureHash:
		batchEvent, err := etherMan.PoE.ParseSendBatch(vLog)
		if err != nil {
			return nil, err
		}
		// Indexed parameters using topics
		var head types.Header
		head.TxHash = vLog.TxHash
		head.Difficulty = big.NewInt(0)
		head.Number = new(big.Int).SetUint64(uint64(batchEvent.NumBatch))

		var batch Batch
		batch.Sequencer = batchEvent.Sequencer
		batch.ChainID = new(big.Int).SetUint64(uint64(batchEvent.BatchChainID))
		batch.GlobalExitRoot = batchEvent.LastGlobalExitRoot
		batch.Header = &head
		batch.BlockNumber = vLog.BlockNumber
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		batch.ReceivedAt = fullBlock.ReceivedAt

		var block Block
		block.BlockNumber = vLog.BlockNumber
		block.BlockHash = vLog.BlockHash
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.Batches = append(block.Batches, batch)
		return &block, nil
	case consolidateBatchSignatureHash:
		var head types.Header
		head.Number = new(big.Int).SetBytes(vLog.Topics[1][:])

		var batch Batch
		batch.Header = &head
		batch.BlockNumber = vLog.BlockNumber
		batch.Aggregator = common.BytesToAddress(vLog.Topics[2].Bytes())
		batch.ConsolidatedTxHash = vLog.TxHash
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		batch.ConsolidatedAt = &fullBlock.ReceivedAt

		var block Block
		block.BlockNumber = vLog.BlockNumber
		block.BlockHash = vLog.BlockHash
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.Batches = append(block.Batches, batch)

		log.Debug("Consolidated tx hash: ", vLog.TxHash, batch.ConsolidatedTxHash)
		return &block, nil
	case newSequencerSignatureHash:
		return nil, nil
	case ownershipTransferredSignatureHash:
		ownership, err := etherMan.Bridge.ParseOwnershipTransferred(vLog)
		if err != nil {
			return nil, err
		}
		emptyAddr := common.Address{}
		if ownership.PreviousOwner == emptyAddr {
			log.Debug("New bridge smc deployment detected. Deployment account: ", ownership.NewOwner)
		} else {
			log.Debug("Bridge smc OwnershipTransferred from account ", ownership.PreviousOwner, " to ", ownership.NewOwner)
		}
		return nil, nil
	case depositEventSignatureHash:
		log.Debug("Deposit event detected")
		deposit, err := etherMan.Bridge.ParseBridgeEvent(vLog)
		if err != nil {
			return nil, err
		}
		var (
			block      Block
			depositAux Deposit
		)
		depositAux.Amount = deposit.Amount
		depositAux.BlockNumber = vLog.BlockNumber
		depositAux.OriginalNetwork = uint(deposit.OriginNetwork)
		depositAux.DestinationAddress = deposit.DestinationAddress
		depositAux.DestinationNetwork = uint(deposit.DestinationNetwork)
		depositAux.TokenAddress = deposit.TokenAddres
		depositAux.DepositCount = uint(deposit.DepositCount)
		block.BlockHash = vLog.BlockHash
		block.BlockNumber = vLog.BlockNumber
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", block.BlockNumber, err)
		}
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.Deposits = append(block.Deposits, depositAux)
		return &block, nil
	case updateGlobalExitRootEventSignatureHash:
		log.Debug("UpdateGlobalExitRoot event detected")
		globalExitRoot, err := etherMan.GlobalExitRootManager.ParseUpdateGlobalExitRoot(vLog)
		if err != nil {
			return nil, err
		}
		var (
			block     Block
			gExitRoot GlobalExitRoot
		)
		gExitRoot.ExitRoots = []common.Hash{globalExitRoot.MainnetExitRoot, globalExitRoot.RollupExitRoot}
		gExitRoot.GlobalExitRootNum = globalExitRoot.GlobalExitRootNum
		gExitRoot.BlockNumber = vLog.BlockNumber
		block.BlockHash = vLog.BlockHash
		block.BlockNumber = vLog.BlockNumber
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", block.BlockNumber, err)
		}
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.GlobalExitRoots = append(block.GlobalExitRoots, gExitRoot)
		return &block, nil
	case claimEventSignatureHash:
		log.Debug("Claim event detected")
		claim, err := etherMan.Bridge.ParseClaimEvent(vLog)
		if err != nil {
			return nil, err
		}
		var (
			block    Block
			claimAux Claim
		)
		claimAux.Amount = claim.Amount
		claimAux.DestinationAddress = claim.DestinationAddress
		claimAux.Index = uint(claim.Index)
		claimAux.OriginalNetwork = uint(claim.OriginalNetwork)
		claimAux.Token = claim.Token
		claimAux.BlockNumber = vLog.BlockNumber
		block.BlockHash = vLog.BlockHash
		block.BlockNumber = vLog.BlockNumber
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", block.BlockNumber, err)
		}
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.Claims = append(block.Claims, claimAux)
		return &block, nil
	case newWrappedTokenEventSignatureHash:
		tokenWrapped, err := etherMan.Bridge.ParseNewWrappedToken(vLog)
		if err != nil {
			return nil, err
		}
		var (
			block    Block
			newToken TokenWrapped
		)
		newToken.OriginalNetwork = uint(tokenWrapped.OriginalNetwork)
		newToken.OriginalTokenAddress = tokenWrapped.OriginalTokenAddress
		newToken.WrappedTokenAddress = tokenWrapped.WrappedTokenAddress
		block.BlockHash = vLog.BlockHash
		block.BlockNumber = vLog.BlockNumber
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", block.BlockNumber, err)
		}
		block.ParentHash = fullBlock.ParentHash()
		block.ReceivedAt = fullBlock.ReceivedAt
		block.Tokens = append(block.Tokens, newToken)
		return &block, nil
	}
	log.Warn("Event not registered: ", vLog)
	return nil, nil
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (etherMan *ClientEtherMan) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return etherMan.EtherClient.HeaderByNumber(ctx, number)
}

// BlockByNumber function retrieves the ethereum block information by ethereum block number.
func (etherMan *ClientEtherMan) BlockByNumber(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	block, err := etherMan.EtherClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) || err.Error() == "block does not exist in blockchain" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return block, nil
}

// GetNetworkID function retrieves the networkID.
func (etherMan *ClientEtherMan) GetNetworkID(ctx context.Context) (uint, error) {
	networkID, err := etherMan.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		return 0, err
	}
	return uint(networkID), nil
}
