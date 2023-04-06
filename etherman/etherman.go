package etherman

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevm"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmglobalexitroot"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

var (
	updateGlobalExitRootSignatureHash              = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	forcedBatchSignatureHash                       = crypto.Keccak256Hash([]byte("ForceBatch(uint64,bytes32,address,bytes)"))
	sequencedBatchesEventSignatureHash             = crypto.Keccak256Hash([]byte("SequenceBatches(uint64)"))
	forceSequencedBatchesSignatureHash             = crypto.Keccak256Hash([]byte("SequenceForceBatches(uint64)"))
	verifyBatchesSignatureHash                     = crypto.Keccak256Hash([]byte("VerifyBatches(uint64,bytes32,address)"))
	verifyBatchesTrustedAggregatorSignatureHash    = crypto.Keccak256Hash([]byte("VerifyBatchesTrustedAggregator(uint64,bytes32,address)"))
	setTrustedSequencerURLSignatureHash            = crypto.Keccak256Hash([]byte("SetTrustedSequencerURL(string)"))
	setTrustedSequencerSignatureHash               = crypto.Keccak256Hash([]byte("SetTrustedSequencer(address)"))
	transferOwnershipSignatureHash                 = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	emergencyStateActivatedSignatureHash           = crypto.Keccak256Hash([]byte("EmergencyStateActivated()"))
	emergencyStateDeactivatedSignatureHash         = crypto.Keccak256Hash([]byte("EmergencyStateDeactivated()"))
	updateZkEVMVersionSignatureHash                = crypto.Keccak256Hash([]byte("UpdateZkEVMVersion(uint64,uint64,string)"))
	consolidatePendingStateSignatureHash           = crypto.Keccak256Hash([]byte("ConsolidatePendingState(uint64,bytes32,uint64)"))
	setTrustedAggregatorTimeoutSignatureHash       = crypto.Keccak256Hash([]byte("SetTrustedAggregatorTimeout(uint64)"))
	setTrustedAggregatorSignatureHash              = crypto.Keccak256Hash([]byte("SetTrustedAggregator(address)"))
	setPendingStateTimeoutSignatureHash            = crypto.Keccak256Hash([]byte("SetPendingStateTimeout(uint64)"))
	setMultiplierBatchFeeSignatureHash             = crypto.Keccak256Hash([]byte("SetMultiplierBatchFee(uint16)"))
	setVerifyBatchTimeTargetSignatureHash          = crypto.Keccak256Hash([]byte("SetVerifyBatchTimeTarget(uint64)"))
	setForceBatchTimeoutSignatureHash              = crypto.Keccak256Hash([]byte("SetForceBatchTimeout(uint64)"))
	activateForceBatchesSignatureHash              = crypto.Keccak256Hash([]byte("ActivateForceBatches()"))
	transferAdminRoleSignatureHash                 = crypto.Keccak256Hash([]byte("TransferAdminRole(address)"))
	acceptAdminRoleSignatureHash                   = crypto.Keccak256Hash([]byte("AcceptAdminRole(address)"))
	proveNonDeterministicPendingStateSignatureHash = crypto.Keccak256Hash([]byte("ProveNonDeterministicPendingState(bytes32,bytes32)"))
	overridePendingStateSignatureHash              = crypto.Keccak256Hash([]byte("OverridePendingState(uint64,bytes32,address)"))

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
	// SequenceBatchesOrder identifies a VerifyBatch event
	SequenceBatchesOrder EventOrder = "SequenceBatches"
	// TrustedVerifyBatchOrder identifies a TrustedVerifyBatch event
	TrustedVerifyBatchOrder EventOrder = "TrustedVerifyBatch"
	// SequenceForceBatchesOrder identifies a SequenceForceBatches event
	SequenceForceBatchesOrder EventOrder = "SequenceForceBatches"
	// ForcedBatchesOrder identifies a ForcedBatches event
	ForcedBatchesOrder EventOrder = "ForcedBatches"
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
	EtherClient           ethClienter
	PoE                   *polygonzkevm.Polygonzkevm
	Bridge                *polygonzkevmbridge.Polygonzkevmbridge
	GlobalExitRootManager *polygonzkevmglobalexitroot.Polygonzkevmglobalexitroot
	SCAddresses           []common.Address
}

// NewClient creates a new etherman.
func NewClient(cfg Config, PoEAddr, bridgeAddr, globalExitRootManAddr common.Address) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.L1URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.L1URL, err)
		return nil, err
	}
	// Create smc clients
	poe, err := polygonzkevm.NewPolygonzkevm(PoEAddr, ethClient)
	if err != nil {
		return nil, err
	}
	bridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	globalExitRoot, err := polygonzkevmglobalexitroot.NewPolygonzkevmglobalexitroot(globalExitRootManAddr, ethClient)
	if err != nil {
		return nil, err
	}
	var scAddresses []common.Address
	scAddresses = append(scAddresses, PoEAddr, globalExitRootManAddr, bridgeAddr)

	return &Client{EtherClient: ethClient, PoE: poe, Bridge: bridge, GlobalExitRootManager: globalExitRoot, SCAddresses: scAddresses}, nil
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

	return &Client{EtherClient: ethClient, Bridge: bridge, SCAddresses: scAddresses}, nil
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
	case sequencedBatchesEventSignatureHash:
		return etherMan.sequencedBatchesEvent(ctx, vLog, blocks, blocksOrder)
	case updateGlobalExitRootSignatureHash:
		return etherMan.updateGlobalExitRootEvent(ctx, vLog, blocks, blocksOrder)
	case forcedBatchSignatureHash:
		return etherMan.forcedBatchEvent(ctx, vLog, blocks, blocksOrder)
	case verifyBatchesTrustedAggregatorSignatureHash:
		return etherMan.verifyBatchesTrustedAggregator(ctx, vLog, blocks, blocksOrder)
	case verifyBatchesSignatureHash:
		log.Warn("VerifyBatches event not implemented yet")
		return nil
	case forceSequencedBatchesSignatureHash:
		return etherMan.forceSequencedBatchesEvent(ctx, vLog, blocks, blocksOrder)
	case depositEventSignatureHash:
		return etherMan.depositEvent(ctx, vLog, blocks, blocksOrder)
	case claimEventSignatureHash:
		return etherMan.claimEvent(ctx, vLog, blocks, blocksOrder)
	case newWrappedTokenEventSignatureHash:
		return etherMan.tokenWrappedEvent(ctx, vLog, blocks, blocksOrder)
	case initializedSignatureHash:
		log.Debug("Initialized event detected")
		return nil
	case setTrustedSequencerSignatureHash:
		log.Debug("SetTrustedSequencer event detected")
		return nil
	case setTrustedSequencerURLSignatureHash:
		log.Debug("SetTrustedSequencerURL event detected")
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
	case emergencyStateActivatedSignatureHash:
		log.Debug("EmergencyStateActivated event detected")
		return nil
	case emergencyStateDeactivatedSignatureHash:
		log.Debug("EmergencyStateDeactivated event detected")
		return nil
	case transferOwnershipSignatureHash:
		log.Debug("TransferOwnership event detected")
		return nil
	case updateZkEVMVersionSignatureHash:
		log.Debug("UpdateZkEVMVersion event detected")
		return nil
	case consolidatePendingStateSignatureHash:
		log.Debug("ConsolidatePendingState event detected")
		return nil
	case setTrustedAggregatorTimeoutSignatureHash:
		log.Debug("SetTrustedAggregatorTimeout event detected")
		return nil
	case setTrustedAggregatorSignatureHash:
		log.Debug("setTrustedAggregator event detected")
		return nil
	case setPendingStateTimeoutSignatureHash:
		log.Debug("SetPendingStateTimeout event detected")
		return nil
	case setMultiplierBatchFeeSignatureHash:
		log.Debug("SetMultiplierBatchFee event detected")
		return nil
	case setVerifyBatchTimeTargetSignatureHash:
		log.Debug("SetVerifyBatchTimeTarget event detected")
		return nil
	case setForceBatchTimeoutSignatureHash:
		log.Debug("SetForceBatchTimeout event detected")
		return nil
	case activateForceBatchesSignatureHash:
		log.Debug("ActivateForceBatches event detected")
		return nil
	case transferAdminRoleSignatureHash:
		log.Debug("TransferAdminRole event detected")
		return nil
	case acceptAdminRoleSignatureHash:
		log.Debug("AcceptAdminRole event detected")
		return nil
	case proveNonDeterministicPendingStateSignatureHash:
		log.Debug("ProveNonDeterministicPendingState event detected")
		return nil
	case overridePendingStateSignatureHash:
		log.Debug("OverridePendingState event detected")
		return nil
	}
	log.Warnf("Event not registered: %+v", vLog)
	return nil
}

func (etherMan *Client) updateGlobalExitRootEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("UpdateGlobalExitRoot event detected")
	globalExitRoot, err := etherMan.GlobalExitRootManager.ParseUpdateGlobalExitRoot(vLog)
	if err != nil {
		return err
	}
	fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
	if err != nil {
		return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
	}
	var gExitRoot GlobalExitRoot
	gExitRoot.ExitRoots = make([]common.Hash, 0)
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, common.BytesToHash(globalExitRoot.MainnetExitRoot[:]))
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, common.BytesToHash(globalExitRoot.RollupExitRoot[:]))
	gExitRoot.GlobalExitRoot = hash(globalExitRoot.MainnetExitRoot, globalExitRoot.RollupExitRoot)
	gExitRoot.BlockNumber = vLog.BlockNumber

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		t := time.Unix(int64(fullBlock.Time()), 0)
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
	d, err := etherMan.Bridge.ParseBridgeEvent(vLog)
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
	c, err := etherMan.Bridge.ParseClaimEvent(vLog)
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
	tw, err := etherMan.Bridge.ParseNewWrappedToken(vLog)
	if err != nil {
		return err
	}
	var tokenWrapped TokenWrapped
	tokenWrapped.OriginalNetwork = uint(tw.OriginNetwork)
	tokenWrapped.OriginalTokenAddress = tw.OriginTokenAddress
	tokenWrapped.WrappedTokenAddress = tw.WrappedTokenAddress
	tokenWrapped.BlockNumber = vLog.BlockNumber

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

func (etherMan *Client) sequencedBatchesEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("SequenceBatches event detected")
	sb, err := etherMan.PoE.ParseSequenceBatches(vLog)
	if err != nil {
		return err
	}
	// Read the tx for this event.
	tx, isPending, err := etherMan.EtherClient.TransactionByHash(ctx, vLog.TxHash)
	if err != nil {
		return err
	} else if isPending {
		return fmt.Errorf("error tx is still pending. TxHash: %s", tx.Hash().String())
	}
	msg, err := tx.AsMessage(types.NewLondonSigner(tx.ChainId()), big.NewInt(0))
	if err != nil {
		log.Error(err)
		return err
	}
	sequences, err := decodeSequences(tx.Data(), sb.NumBatch, msg.From(), vLog.TxHash)
	if err != nil {
		return fmt.Errorf("error decoding the sequences: %v", err)
	}

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.SequencedBatches = append(block.SequencedBatches, sequences)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].SequencedBatches = append((*blocks)[len(*blocks)-1].SequencedBatches, sequences)
	} else {
		log.Error("Error processing SequencedBatches event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing SequencedBatches event")
	}
	or := Order{
		Name: SequenceBatchesOrder,
		Pos:  len((*blocks)[len(*blocks)-1].SequencedBatches) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func decodeSequences(txData []byte, lastBatchNumber uint64, sequencer common.Address, txHash common.Hash) ([]SequencedBatch, error) {
	// Extract coded txs.
	// Load contract ABI
	abi, err := abi.JSON(strings.NewReader(polygonzkevm.PolygonzkevmABI))
	if err != nil {
		return nil, err
	}

	// Recover Method from signature and ABI
	method, err := abi.MethodById(txData[:4])
	if err != nil {
		return nil, err
	}

	// Unpack method inputs
	data, err := method.Inputs.Unpack(txData[4:])
	if err != nil {
		return nil, err
	}
	var sequences []polygonzkevm.PolygonZkEVMBatchData
	bytedata, err := json.Marshal(data[0])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytedata, &sequences)
	if err != nil {
		return nil, err
	}

	sequencedBatches := make([]SequencedBatch, len(sequences))
	for i, seq := range sequences {
		bn := lastBatchNumber - uint64(len(sequences)-(i+1))
		sequencedBatches[i] = SequencedBatch{
			BatchNumber:           bn,
			Sequencer:             sequencer,
			TxHash:                txHash,
			PolygonZkEVMBatchData: seq,
		}
	}

	return sequencedBatches, nil
}

func (etherMan *Client) verifyBatchesTrustedAggregator(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("trustedVerifyBatches event detected")
	vb, err := etherMan.PoE.ParseVerifyBatchesTrustedAggregator(vLog)
	if err != nil {
		return err
	}
	var trustedVerifyBatch VerifiedBatch
	trustedVerifyBatch.BatchNumber = vb.NumBatch
	trustedVerifyBatch.TxHash = vLog.TxHash
	trustedVerifyBatch.Aggregator = vb.Aggregator
	trustedVerifyBatch.StateRoot = vb.StateRoot

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.VerifiedBatches = append(block.VerifiedBatches, trustedVerifyBatch)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].VerifiedBatches = append((*blocks)[len(*blocks)-1].VerifiedBatches, trustedVerifyBatch)
	} else {
		log.Error("Error processing trustedVerifyBatch event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing trustedVerifyBatch event")
	}
	or := Order{
		Name: TrustedVerifyBatchOrder,
		Pos:  len((*blocks)[len(*blocks)-1].VerifiedBatches) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) forceSequencedBatchesEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("SequenceForceBatches event detect")
	fsb, err := etherMan.PoE.ParseSequenceForceBatches(vLog)
	if err != nil {
		return err
	}

	// Read the tx for this batch.
	tx, isPending, err := etherMan.EtherClient.TransactionByHash(ctx, vLog.TxHash)
	if err != nil {
		return err
	} else if isPending {
		return fmt.Errorf("error: tx is still pending. TxHash: %s", tx.Hash().String())
	}
	msg, err := tx.AsMessage(types.NewLondonSigner(tx.ChainId()), big.NewInt(0))
	if err != nil {
		log.Error(err)
		return err
	}
	fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
	if err != nil {
		return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
	}
	sequencedForceBatch, err := decodeSequencedForceBatches(tx.Data(), fsb.NumBatch, msg.From(), vLog.TxHash, fullBlock)
	if err != nil {
		return err
	}

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time()), 0), fullBlock)
		block.SequencedForceBatches = append(block.SequencedForceBatches, sequencedForceBatch)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].SequencedForceBatches = append((*blocks)[len(*blocks)-1].SequencedForceBatches, sequencedForceBatch)
	} else {
		log.Error("Error processing ForceSequencedBatches event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing ForceSequencedBatches event")
	}
	or := Order{
		Name: SequenceForceBatchesOrder,
		Pos:  len((*blocks)[len(*blocks)-1].SequencedForceBatches) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)

	return nil
}

func (etherMan *Client) forcedBatchEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("ForceBatch event detected")
	fb, err := etherMan.PoE.ParseForceBatch(vLog)
	if err != nil {
		return err
	}
	var forcedBatch ForcedBatch
	forcedBatch.ForcedBatchNumber = fb.ForceBatchNum
	forcedBatch.GlobalExitRoot = fb.LastGlobalExitRoot
	forcedBatch.BlockNumber = vLog.BlockNumber
	// Read the tx for this batch.
	tx, isPending, err := etherMan.EtherClient.TransactionByHash(ctx, vLog.TxHash)
	if err != nil {
		return err
	} else if isPending {
		return fmt.Errorf("error: tx is still pending. TxHash: %s", tx.Hash().String())
	}
	msg, err := tx.AsMessage(types.NewLondonSigner(tx.ChainId()), big.NewInt(0))
	if err != nil {
		log.Error(err)
		return err
	}
	if fb.Sequencer == msg.From() {
		txData := tx.Data()
		// Extract coded txs.
		// Load contract ABI
		abi, err := abi.JSON(strings.NewReader(polygonzkevm.PolygonzkevmABI))
		if err != nil {
			return err
		}

		// Recover Method from signature and ABI
		method, err := abi.MethodById(txData[:4])
		if err != nil {
			return err
		}

		// Unpack method inputs
		data, err := method.Inputs.Unpack(txData[4:])
		if err != nil {
			return err
		}
		bytedata := data[0].([]byte)
		forcedBatch.RawTxsData = bytedata
	} else {
		forcedBatch.RawTxsData = fb.Transactions
	}
	forcedBatch.Sequencer = fb.Sequencer
	fullBlock, err := etherMan.EtherClient.BlockByHash(ctx, vLog.BlockHash)
	if err != nil {
		return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %w", vLog.BlockNumber, err)
	}
	t := time.Unix(int64(fullBlock.Time()), 0)
	forcedBatch.ForcedAt = t

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		block := prepareBlock(vLog, t, fullBlock)
		block.ForcedBatches = append(block.ForcedBatches, forcedBatch)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].ForcedBatches = append((*blocks)[len(*blocks)-1].ForcedBatches, forcedBatch)
	} else {
		log.Error("Error processing ForceBatch event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing ForceBatch event")
	}
	or := Order{
		Name: ForcedBatchesOrder,
		Pos:  len((*blocks)[len(*blocks)-1].ForcedBatches) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func decodeSequencedForceBatches(txData []byte, lastBatchNumber uint64, sequencer common.Address, txHash common.Hash, block *types.Block) ([]SequencedForceBatch, error) {
	// Extract coded txs.
	// Load contract ABI
	abi, err := abi.JSON(strings.NewReader(polygonzkevm.PolygonzkevmABI))
	if err != nil {
		return nil, err
	}

	// Recover Method from signature and ABI
	method, err := abi.MethodById(txData[:4])
	if err != nil {
		return nil, err
	}

	// Unpack method inputs
	data, err := method.Inputs.Unpack(txData[4:])
	if err != nil {
		return nil, err
	}

	var forceBatches []polygonzkevm.PolygonZkEVMForcedBatchData
	bytedata, err := json.Marshal(data[0])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytedata, &forceBatches)
	if err != nil {
		return nil, err
	}

	sequencedForcedBatches := make([]SequencedForceBatch, len(forceBatches))
	for i, force := range forceBatches {
		bn := lastBatchNumber - uint64(len(forceBatches)-(i+1))
		sequencedForcedBatches[i] = SequencedForceBatch{
			BatchNumber:                 bn,
			Sequencer:                   sequencer,
			TxHash:                      txHash,
			Timestamp:                   time.Unix(int64(block.Time()), 0),
			PolygonZkEVMForcedBatchData: force,
		}
	}
	return sequencedForcedBatches, nil
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

// GetLatestBatchNumber function allows to retrieve the latest proposed batch in the smc
func (etherMan *Client) GetLatestBatchNumber() (uint64, error) {
	latestBatch, err := etherMan.PoE.LastBatchSequenced(&bind.CallOpts{Pending: false})
	return uint64(latestBatch), err
}

// GetNetworkID gets the network ID of the dedicated chain.
func (etherMan *Client) GetNetworkID(ctx context.Context) (uint, error) {
	networkID, err := etherMan.Bridge.NetworkID(&bind.CallOpts{Pending: false})
	if err != nil {
		return 0, err
	}
	return uint(networkID), nil
}

// GetTrustedSequencerURL Gets the trusted sequencer url from rollup smc
func (etherMan *Client) GetTrustedSequencerURL() (string, error) {
	return etherMan.PoE.TrustedSequencerURL(&bind.CallOpts{Pending: false})
}
