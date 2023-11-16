package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

var (
	updateGlobalExitRootSignatureHash = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	transferOwnershipSignatureHash    = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))

	// Bridge events
	depositEventSignatureHash         = crypto.Keccak256Hash([]byte("BridgeEvent(uint8,uint32,address,uint32,address,uint256,bytes,uint32)"))
	claimEventSignatureHash           = crypto.Keccak256Hash([]byte("ClaimEvent(uint32,uint32,address,address,uint256)"))
	newWrappedTokenEventSignatureHash = crypto.Keccak256Hash([]byte("NewWrappedToken(uint32,address,address,bytes)"))

	// Proxy events
	initializedSignatureHash    = crypto.Keccak256Hash([]byte("Initialized(uint8)"))
	adminChangedSignatureHash   = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))
	beaconUpgradedSignatureHash = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
	upgradedSignatureHash       = crypto.Keccak256Hash([]byte("Upgraded(address)"))

	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

// EventOrder is the the type used to identify the events order
type EventOrder string

const (
	// GlobalExitRootsOrder identifies a GlobalExitRoot event
	GlobalExitRootsOrder EventOrder = "GlobalExitRoot"
	// DepositsOrder identifies a Deposits event
	DepositsOrder EventOrder = "Deposit"
	// ClaimsOrder identifies a Claims event
	ClaimsOrder EventOrder = "Claim"
	// TokensOrder identifies a TokenWrapped event
	TokensOrder EventOrder = "TokenWrapped"
)

type ethClienter interface {
	ethereum.ChainReader
	ethereum.LogFilterer
	ethereum.TransactionReader
}

// Client is a simple implementation of EtherMan.
type Client struct {
	EtherClient                ethClienter
	PolygonBridge              *polygonzkevmbridge.Polygonzkevmbridge
	PolygonZkEVMGlobalExitRoot *polygonzkevmglobalexitroot.Polygonzkevmglobalexitroot
	SCAddresses                []common.Address
}

// NewClient creates a new etherman.
func NewClient(cfg Config, polygonBridgeAddr, polygonZkEVMGlobalExitRootAddress common.Address) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.L1URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.L1URL, err)
		return nil, err
	}
	// Create smc clients
	polygonBridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(polygonBridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	polygonZkEVMGlobalExitRoot, err := polygonzkevmglobalexitroot.NewPolygonzkevmglobalexitroot(polygonZkEVMGlobalExitRootAddress, ethClient)
	if err != nil {
		return nil, err
	}
	var scAddresses []common.Address
	scAddresses = append(scAddresses, polygonZkEVMGlobalExitRootAddress, polygonBridgeAddr)

	return &Client{EtherClient: ethClient, PolygonBridge: polygonBridge, PolygonZkEVMGlobalExitRoot: polygonZkEVMGlobalExitRoot, SCAddresses: scAddresses}, nil
}

// NewL2Client creates a new etherman for L2.
func NewL2Client(url string, bridgeAddr common.Address) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", url, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	scAddresses := []common.Address{bridgeAddr}

	return &Client{EtherClient: ethClient, PolygonBridge: bridge, SCAddresses: scAddresses}, nil
}

// GetRollupInfoByBlockRange function retrieves the Rollup information that are included in all this ethereum blocks
// from block x to block y.
func (etherMan *Client) GetRollupInfoByBlockRange(ctx context.Context, fromBlock uint64, toBlock *uint64) ([]Block, map[common.Hash][]Order, error) {
	// Filter query
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		Addresses: etherMan.SCAddresses,
	}
	if toBlock != nil {
		query.ToBlock = new(big.Int).SetUint64(*toBlock)
	}
	blocks, blocksOrder, err := etherMan.readEvents(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	return blocks, blocksOrder, nil
}

// Order contains the event order to let the synchronizer store the information following this order.
type Order struct {
	Name EventOrder
	Pos  int
}

func (etherMan *Client) readEvents(ctx context.Context, query ethereum.FilterQuery) ([]Block, map[common.Hash][]Order, error) {
	logs, err := etherMan.EtherClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	var blocks []Block
	blocksOrder := make(map[common.Hash][]Order)
	for _, vLog := range logs {
		err := etherMan.processEvent(ctx, vLog, &blocks, &blocksOrder)
		if err != nil {
			log.Warnf("error processing event. Retrying... Error: %s. vLog: %+v", err.Error(), vLog)
			return nil, nil, err
		}
	}
	return blocks, blocksOrder, nil
}

func (etherMan *Client) processEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	switch vLog.Topics[0] {
	case updateGlobalExitRootSignatureHash:
		return etherMan.updateGlobalExitRootEvent(ctx, vLog, blocks, blocksOrder)
	case depositEventSignatureHash:
		return etherMan.depositEvent(ctx, vLog, blocks, blocksOrder)
	case claimEventSignatureHash:
		return etherMan.claimEvent(ctx, vLog, blocks, blocksOrder)
	case newWrappedTokenEventSignatureHash:
		return etherMan.tokenWrappedEvent(ctx, vLog, blocks, blocksOrder)
	case initializedSignatureHash:
		log.Debug("Initialized event detected")
		return nil
	case adminChangedSignatureHash:
		log.Debug("AdminChanged event detected")
		return nil
	case beaconUpgradedSignatureHash:
		log.Debug("BeaconUpgraded event detected")
		return nil
	case upgradedSignatureHash:
		log.Debug("Upgraded event detected")
		return nil
	case transferOwnershipSignatureHash:
		log.Debug("TransferOwnership event detected")
		return nil
	}
	log.Warnf("Event not registered: %+v", vLog)
	return nil
}

func (etherMan *Client) updateGlobalExitRootEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("UpdateGlobalExitRoot event detected")
	globalExitRoot, err := etherMan.PolygonZkEVMGlobalExitRoot.ParseUpdateGlobalExitRoot(vLog)
	if err != nil {
		return err
	}
	fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
	if err != nil {
		return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
	}
	t := time.Unix(int64(fullBlock.Time()), 0)
	var gExitRoot GlobalExitRoot
	gExitRoot.ExitRoots = make([]common.Hash, 0)
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, common.BytesToHash(globalExitRoot.MainnetExitRoot[:]))
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, common.BytesToHash(globalExitRoot.RollupExitRoot[:]))
	gExitRoot.GlobalExitRoot = hash(globalExitRoot.MainnetExitRoot, globalExitRoot.RollupExitRoot)
	gExitRoot.BlockNumber = vLog.BlockNumber
	gExitRoot.Time = t
	log.Debugf("UpdateGlobalExitRoot event[%+v] blockNumber[%v]", gExitRoot, vLog.BlockNumber)

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		block := prepareBlock(vLog, t, fullBlock)
		block.GlobalExitRoots = append(block.GlobalExitRoots, gExitRoot)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].GlobalExitRoots = append((*blocks)[len(*blocks)-1].GlobalExitRoots, gExitRoot)
	} else {
		log.Error("Error processing UpdateGlobalExitRoot event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing UpdateGlobalExitRoot event")
	}
	or := Order{
		Name: GlobalExitRootsOrder,
		Pos:  len((*blocks)[len(*blocks)-1].GlobalExitRoots) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) depositEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("Deposit event detected")
	d, err := etherMan.PolygonBridge.ParseBridgeEvent(vLog)
	if err != nil {
		return err
	}
	var deposit Deposit
	deposit.Amount = d.Amount
	deposit.BlockNumber = vLog.BlockNumber
	deposit.OriginalNetwork = uint(d.OriginNetwork)
	deposit.DestinationAddress = d.DestinationAddress
	deposit.DestinationNetwork = uint(d.DestinationNetwork)
	deposit.OriginalAddress = d.OriginAddress
	deposit.DepositCount = uint(d.DepositCount)
	deposit.TxHash = vLog.TxHash
	deposit.Metadata = d.Metadata
	deposit.LeafType = d.LeafType
	log.Debugf("Deposit event[%+v] blockNumber[%v]", deposit, vLog.BlockNumber)

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.Deposits = append(block.Deposits, deposit)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].Deposits = append((*blocks)[len(*blocks)-1].Deposits, deposit)
	} else {
		log.Error("Error processing deposit event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing Deposit event")
	}
	or := Order{
		Name: DepositsOrder,
		Pos:  len((*blocks)[len(*blocks)-1].Deposits) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) claimEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("Claim event detected")
	c, err := etherMan.PolygonBridge.ParseClaimEvent(vLog)
	if err != nil {
		return err
	}
	var claim Claim
	claim.Amount = c.Amount
	claim.DestinationAddress = c.DestinationAddress
	claim.Index = uint(c.Index)
	claim.OriginalNetwork = uint(c.OriginNetwork)
	claim.OriginalAddress = c.OriginAddress
	claim.BlockNumber = vLog.BlockNumber
	claim.TxHash = vLog.TxHash
	log.Debugf("Claim event[%+v] blockNumber[%v]", claim, vLog.BlockNumber)

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.Claims = append(block.Claims, claim)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].Claims = append((*blocks)[len(*blocks)-1].Claims, claim)
	} else {
		log.Error("Error processing claim event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing claim event")
	}
	or := Order{
		Name: ClaimsOrder,
		Pos:  len((*blocks)[len(*blocks)-1].Claims) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) tokenWrappedEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("TokenWrapped event detected")
	tw, err := etherMan.PolygonBridge.ParseNewWrappedToken(vLog)
	if err != nil {
		return err
	}
	var tokenWrapped TokenWrapped
	tokenWrapped.OriginalNetwork = uint(tw.OriginNetwork)
	tokenWrapped.OriginalTokenAddress = tw.OriginTokenAddress
	tokenWrapped.WrappedTokenAddress = tw.WrappedTokenAddress
	tokenWrapped.BlockNumber = vLog.BlockNumber
	log.Debugf("TokenWrapped event[%+v] blockNumber[%v]", tokenWrapped, vLog.BlockNumber)

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.Tokens = append(block.Tokens, tokenWrapped)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].Tokens = append((*blocks)[len(*blocks)-1].Tokens, tokenWrapped)
	} else {
		log.Error("Error processing TokenWrapped event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing TokenWrapped event")
	}
	or := Order{
		Name: TokensOrder,
		Pos:  len((*blocks)[len(*blocks)-1].Tokens) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func prepareBlock(vLog types.Log, t time.Time, fullBlock *types.Block) Block {
	var block Block
	block.BlockNumber = vLog.BlockNumber
	block.BlockHash = vLog.BlockHash
	block.ParentHash = fullBlock.ParentHash()
	block.ReceivedAt = t
	return block
}

func hash(data ...[32]byte) [32]byte {
	var res [32]byte
	hash := sha3.NewLegacyKeccak256()
	for _, d := range data {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(res[:], hash.Sum(nil))
	return res
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (etherMan *Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return etherMan.EtherClient.HeaderByNumber(ctx, number)
}

// EthBlockByNumber function retrieves the ethereum block information by ethereum block number.
func (etherMan *Client) EthBlockByNumber(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	block, err := etherMan.EtherClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) || err.Error() == "block does not exist in blockchain" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return block, nil
}

// GetNetworkID gets the network ID of the dedicated chain.
func (etherMan *Client) GetNetworkID(ctx context.Context) (uint, error) {
	networkID, err := etherMan.PolygonBridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		return 0, err
	}
	return uint(networkID), nil
}
