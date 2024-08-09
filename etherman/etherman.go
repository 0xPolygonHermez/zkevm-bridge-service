package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/oldpolygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonrollupmanager"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmglobalexitroot"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

var (
	// New Ger event
	updateL1InfoTreeSignatureHash = crypto.Keccak256Hash([]byte("UpdateL1InfoTree(bytes32,bytes32)"))

	// New Bridge events
	depositEventSignatureHash         = crypto.Keccak256Hash([]byte("BridgeEvent(uint8,uint32,address,uint32,address,uint256,bytes,uint32)")) // Used in oldBridge as well
	claimEventSignatureHash           = crypto.Keccak256Hash([]byte("ClaimEvent(uint256,uint32,address,address,uint256)"))
	newWrappedTokenEventSignatureHash = crypto.Keccak256Hash([]byte("NewWrappedToken(uint32,address,address,bytes)")) // Used in oldBridge as well

	// Old Bridge events
	oldClaimEventSignatureHash = crypto.Keccak256Hash([]byte("ClaimEvent(uint32,uint32,address,address,uint256)"))

	// Proxy events
	initializedProxySignatureHash = crypto.Keccak256Hash([]byte("Initialized(uint8)"))
	adminChangedSignatureHash     = crypto.Keccak256Hash([]byte("AdminChanged(address,address)"))
	beaconUpgradedSignatureHash   = crypto.Keccak256Hash([]byte("BeaconUpgraded(address)"))
	upgradedSignatureHash         = crypto.Keccak256Hash([]byte("Upgraded(address)"))

	// Events RollupManager
	setBatchFeeSignatureHash                       = crypto.Keccak256Hash([]byte("SetBatchFee(uint256)"))
	setTrustedAggregatorSignatureHash              = crypto.Keccak256Hash([]byte("SetTrustedAggregator(address)"))       // Used in oldZkEvm as well
	setVerifyBatchTimeTargetSignatureHash          = crypto.Keccak256Hash([]byte("SetVerifyBatchTimeTarget(uint64)"))    // Used in oldZkEvm as well
	setMultiplierBatchFeeSignatureHash             = crypto.Keccak256Hash([]byte("SetMultiplierBatchFee(uint16)"))       // Used in oldZkEvm as well
	setPendingStateTimeoutSignatureHash            = crypto.Keccak256Hash([]byte("SetPendingStateTimeout(uint64)"))      // Used in oldZkEvm as well
	setTrustedAggregatorTimeoutSignatureHash       = crypto.Keccak256Hash([]byte("SetTrustedAggregatorTimeout(uint64)")) // Used in oldZkEvm as well
	overridePendingStateSignatureHash              = crypto.Keccak256Hash([]byte("OverridePendingState(uint32,uint64,bytes32,bytes32,address)"))
	proveNonDeterministicPendingStateSignatureHash = crypto.Keccak256Hash([]byte("ProveNonDeterministicPendingState(bytes32,bytes32)")) // Used in oldZkEvm as well
	consolidatePendingStateSignatureHash           = crypto.Keccak256Hash([]byte("ConsolidatePendingState(uint32,uint64,bytes32,bytes32,uint64)"))
	verifyBatchesTrustedAggregatorSignatureHash    = crypto.Keccak256Hash([]byte("VerifyBatchesTrustedAggregator(uint32,uint64,bytes32,bytes32,address)"))
	rollupManagerVerifyBatchesSignatureHash        = crypto.Keccak256Hash([]byte("VerifyBatches(uint32,uint64,bytes32,bytes32,address)"))
	onSequenceBatchesSignatureHash                 = crypto.Keccak256Hash([]byte("OnSequenceBatches(uint32,uint64)"))
	updateRollupSignatureHash                      = crypto.Keccak256Hash([]byte("UpdateRollup(uint32,uint32,uint64)"))
	addExistingRollupSignatureHash                 = crypto.Keccak256Hash([]byte("AddExistingRollup(uint32,uint64,address,uint64,uint8,uint64)"))
	createNewRollupSignatureHash                   = crypto.Keccak256Hash([]byte("CreateNewRollup(uint32,uint32,address,uint64,address)"))
	obsoleteRollupTypeSignatureHash                = crypto.Keccak256Hash([]byte("ObsoleteRollupType(uint32)"))
	addNewRollupTypeSignatureHash                  = crypto.Keccak256Hash([]byte("AddNewRollupType(uint32,address,address,uint64,uint8,bytes32,string)"))

	// Extra RollupManager
	initializedSignatureHash               = crypto.Keccak256Hash([]byte("Initialized(uint64)"))                       // Initializable. Used in RollupBase as well
	roleAdminChangedSignatureHash          = crypto.Keccak256Hash([]byte("RoleAdminChanged(bytes32,bytes32,bytes32)")) // IAccessControlUpgradeable
	roleGrantedSignatureHash               = crypto.Keccak256Hash([]byte("RoleGranted(bytes32,address,address)"))      // IAccessControlUpgradeable
	roleRevokedSignatureHash               = crypto.Keccak256Hash([]byte("RoleRevoked(bytes32,address,address)"))      // IAccessControlUpgradeable
	emergencyStateActivatedSignatureHash   = crypto.Keccak256Hash([]byte("EmergencyStateActivated()"))                 // EmergencyManager. Used in oldZkEvm as well
	emergencyStateDeactivatedSignatureHash = crypto.Keccak256Hash([]byte("EmergencyStateDeactivated()"))               // EmergencyManager. Used in oldZkEvm as well

	// PreLxLy events
	updateGlobalExitRootSignatureHash              = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	oldVerifyBatchesTrustedAggregatorSignatureHash = crypto.Keccak256Hash([]byte("VerifyBatchesTrustedAggregator(uint64,bytes32,address)"))
	transferOwnershipSignatureHash                 = crypto.Keccak256Hash([]byte("OwnershipTransferred(address,address)"))
	updateZkEVMVersionSignatureHash                = crypto.Keccak256Hash([]byte("UpdateZkEVMVersion(uint64,uint64,string)"))
	oldConsolidatePendingStateSignatureHash        = crypto.Keccak256Hash([]byte("ConsolidatePendingState(uint64,bytes32,uint64)"))
	oldOverridePendingStateSignatureHash           = crypto.Keccak256Hash([]byte("OverridePendingState(uint64,bytes32,address)"))
	sequenceBatchesPreEtrogSignatureHash           = crypto.Keccak256Hash([]byte("SequenceBatches(uint64)"))

	setForceBatchTimeoutSignatureHash   = crypto.Keccak256Hash([]byte("SetForceBatchTimeout(uint64)"))             // Used in oldZkEvm as well
	setTrustedSequencerURLSignatureHash = crypto.Keccak256Hash([]byte("SetTrustedSequencerURL(string)"))           // Used in oldZkEvm as well
	setTrustedSequencerSignatureHash    = crypto.Keccak256Hash([]byte("SetTrustedSequencer(address)"))             // Used in oldZkEvm as well
	verifyBatchesSignatureHash          = crypto.Keccak256Hash([]byte("VerifyBatches(uint64,bytes32,address)"))    // Used in oldZkEvm as well
	sequenceForceBatchesSignatureHash   = crypto.Keccak256Hash([]byte("SequenceForceBatches(uint64)"))             // Used in oldZkEvm as well
	forceBatchSignatureHash             = crypto.Keccak256Hash([]byte("ForceBatch(uint64,bytes32,address,bytes)")) // Used in oldZkEvm as well
	sequenceBatchesSignatureHash        = crypto.Keccak256Hash([]byte("SequenceBatches(uint64,bytes32)"))          // Used in oldZkEvm as well
	acceptAdminRoleSignatureHash        = crypto.Keccak256Hash([]byte("AcceptAdminRole(address)"))                 // Used in oldZkEvm as well
	transferAdminRoleSignatureHash      = crypto.Keccak256Hash([]byte("TransferAdminRole(address)"))               // Used in oldZkEvm as well

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
	// VerifyBatchOrder identifies a VerifyBatch event
	VerifyBatchOrder EventOrder = "VerifyBatch"
	// ActivateEtrogOrder identifies the event to activate etrog
	ActivateEtrogOrder EventOrder = "etrog"
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
	OldPolygonBridge           *oldpolygonzkevmbridge.Oldpolygonzkevmbridge
	PolygonZkEVMGlobalExitRoot *polygonzkevmglobalexitroot.Polygonzkevmglobalexitroot
	PolygonRollupManager       *polygonrollupmanager.Polygonrollupmanager
	ClaimCompressor            *claimcompressor.Claimcompressor
	RollupID                   uint32
	SCAddresses                []common.Address
}

// NewClient creates a new etherman.
func NewClient(cfg Config, polygonBridgeAddr, polygonZkEVMGlobalExitRootAddress, polygonRollupManagerAddress, polygonZkEvmAddress common.Address) (*Client, error) {
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
	oldpolygonBridge, err := oldpolygonzkevmbridge.NewOldpolygonzkevmbridge(polygonBridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	polygonZkEVMGlobalExitRoot, err := polygonzkevmglobalexitroot.NewPolygonzkevmglobalexitroot(polygonZkEVMGlobalExitRootAddress, ethClient)
	if err != nil {
		return nil, err
	}
	polygonRollupManager, err := polygonrollupmanager.NewPolygonrollupmanager(polygonRollupManagerAddress, ethClient)
	if err != nil {
		return nil, err
	}

	// Get RollupID
	rollupID, err := polygonRollupManager.RollupAddressToID(&bind.CallOpts{Pending: false}, polygonZkEvmAddress)
	if err != nil {
		return nil, err
	}
	log.Debug("rollupID: ", rollupID)
	var scAddresses []common.Address
	scAddresses = append(scAddresses, polygonZkEVMGlobalExitRootAddress, polygonBridgeAddr, polygonRollupManagerAddress)

	return &Client{
		EtherClient:                ethClient,
		PolygonBridge:              polygonBridge,
		OldPolygonBridge:           oldpolygonBridge,
		PolygonZkEVMGlobalExitRoot: polygonZkEVMGlobalExitRoot,
		PolygonRollupManager:       polygonRollupManager,
		RollupID:                   rollupID,
		SCAddresses:                scAddresses}, nil
}

// NewL2Client creates a new etherman for L2.
func NewL2Client(url string, polygonBridgeAddr, claimCompressorAddress common.Address) (*Client, error) {
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", url, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := polygonzkevmbridge.NewPolygonzkevmbridge(polygonBridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	oldpolygonBridge, err := oldpolygonzkevmbridge.NewOldpolygonzkevmbridge(polygonBridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	var claimCompressor *claimcompressor.Claimcompressor
	if claimCompressorAddress == (common.Address{}) {
		log.Warn("Claim compressor Address not configured")
	} else {
		log.Infof("Grouping claims allowed, claimCompressor=%s", claimCompressorAddress.String())
		claimCompressor, err = claimcompressor.NewClaimcompressor(claimCompressorAddress, ethClient)
		if err != nil {
			log.Errorf("error creating claimCompressor: %+v", err)
			return nil, err
		}
	}

	scAddresses := []common.Address{polygonBridgeAddr}

	return &Client{
		EtherClient:      ethClient,
		PolygonBridge:    bridge,
		OldPolygonBridge: oldpolygonBridge,
		SCAddresses:      scAddresses,
		ClaimCompressor:  claimCompressor,
	}, nil
}

// GetRollupInfoByBlockRange function retrieves the Rollup information that are included in all this ethereum blocks
// from block x to block y.
func (etherMan *Client) GetRollupInfoByBlockRange(ctx context.Context, fromBlock uint64, toBlock *uint64) ([]Block, map[common.Hash][]Order, error) {
	// Filter query
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		Addresses: etherMan.SCAddresses,
		Topics:    [][]common.Hash{{updateGlobalExitRootSignatureHash, updateL1InfoTreeSignatureHash, depositEventSignatureHash, claimEventSignatureHash, oldClaimEventSignatureHash, newWrappedTokenEventSignatureHash, verifyBatchesTrustedAggregatorSignatureHash, rollupManagerVerifyBatchesSignatureHash, addExistingRollupSignatureHash, createNewRollupSignatureHash}},
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
	case updateL1InfoTreeSignatureHash:
		return etherMan.updateL1InfoTreeEvent(ctx, vLog, blocks, blocksOrder)
	case depositEventSignatureHash:
		return etherMan.depositEvent(ctx, vLog, blocks, blocksOrder)
	case claimEventSignatureHash:
		return etherMan.newClaimEvent(ctx, vLog, blocks, blocksOrder)
	case oldClaimEventSignatureHash:
		return etherMan.oldClaimEvent(ctx, vLog, blocks, blocksOrder)
	case newWrappedTokenEventSignatureHash:
		return etherMan.tokenWrappedEvent(ctx, vLog, blocks, blocksOrder)
	case initializedProxySignatureHash:
		log.Debug("Initialized proxy event detected. Ignoring...")
		return nil
	case adminChangedSignatureHash:
		log.Debug("AdminChanged event detected. Ignoring...")
		return nil
	case beaconUpgradedSignatureHash:
		log.Debug("BeaconUpgraded event detected. Ignoring...")
		return nil
	case upgradedSignatureHash:
		log.Debug("Upgraded event detected. Ignoring...")
		return nil
	case transferOwnershipSignatureHash:
		log.Debug("TransferOwnership event detected. Ignoring...")
		return nil
	case setBatchFeeSignatureHash:
		log.Debug("SetBatchFee event detected. Ignoring...")
		return nil
	case setTrustedAggregatorSignatureHash:
		log.Debug("SetTrustedAggregator event detected. Ignoring...")
		return nil
	case setVerifyBatchTimeTargetSignatureHash:
		log.Debug("SetVerifyBatchTimeTarget event detected. Ignoring...")
		return nil
	case setMultiplierBatchFeeSignatureHash:
		log.Debug("SetMultiplierBatchFee event detected. Ignoring...")
		return nil
	case setPendingStateTimeoutSignatureHash:
		log.Debug("SetPendingStateTimeout event detected. Ignoring...")
		return nil
	case setTrustedAggregatorTimeoutSignatureHash:
		log.Debug("SetTrustedAggregatorTimeout event detected. Ignoring...")
		return nil
	case overridePendingStateSignatureHash:
		log.Debug("OverridePendingState event detected. Ignoring...")
		return nil
	case proveNonDeterministicPendingStateSignatureHash:
		log.Debug("ProveNonDeterministicPendingState event detected. Ignoring...")
		return nil
	case consolidatePendingStateSignatureHash:
		log.Debug("ConsolidatePendingState event detected. Ignoring...")
		return nil
	case verifyBatchesTrustedAggregatorSignatureHash:
		return etherMan.verifyBatchesTrustedAggregatorEvent(ctx, vLog, blocks, blocksOrder)
	case rollupManagerVerifyBatchesSignatureHash:
		return etherMan.verifyBatchesEvent(ctx, vLog, blocks, blocksOrder)
	case onSequenceBatchesSignatureHash:
		log.Debug("OnSequenceBatches event detected. Ignoring...")
		return nil
	case updateRollupSignatureHash:
		log.Debug("UpdateRollup event detected. Ignoring...")
		return nil
	case addExistingRollupSignatureHash:
		return etherMan.AddExistingRollupEvent(ctx, vLog, blocks, blocksOrder)
	case createNewRollupSignatureHash:
		return etherMan.createNewRollupEvent(ctx, vLog, blocks, blocksOrder)
	case obsoleteRollupTypeSignatureHash:
		log.Debug("ObsoleteRollupType event detected. Ignoring...")
		return nil
	case addNewRollupTypeSignatureHash:
		log.Debug("AddNewRollupType event detected. Ignoring...")
		return nil
	case initializedSignatureHash:
		log.Debug("Initialized event detected. Ignoring...")
		return nil
	case roleAdminChangedSignatureHash:
		log.Debug("RoleAdminChanged event detected. Ignoring...")
		return nil
	case roleGrantedSignatureHash:
		log.Debug("RoleGranted event detected. Ignoring...")
		return nil
	case roleRevokedSignatureHash:
		log.Debug("RoleRevoked event detected. Ignoring...")
		return nil
	case emergencyStateActivatedSignatureHash:
		log.Debug("EmergencyStateActivated event detected. Ignoring...")
		return nil
	case emergencyStateDeactivatedSignatureHash:
		log.Debug("EmergencyStateDeactivated event detected. Ignoring...")
		return nil
	case oldVerifyBatchesTrustedAggregatorSignatureHash:
		log.Debug("OldVerifyBatchesTrustedAggregator event detected. Ignoring...")
		return nil
	case updateZkEVMVersionSignatureHash:
		log.Debug("UpdateZkEVMVersion event detected. Ignoring...")
		return nil
	case oldConsolidatePendingStateSignatureHash:
		log.Debug("OldConsolidatePendingState event detected. Ignoring...")
		return nil
	case oldOverridePendingStateSignatureHash:
		log.Debug("OldOverridePendingState event detected. Ignoring...")
		return nil
	case sequenceBatchesPreEtrogSignatureHash:
		log.Debug("SequenceBatchesPreEtrog event detected. Ignoring...")
		return nil
	case setForceBatchTimeoutSignatureHash:
		log.Debug("SetForceBatchTimeout event detected. Ignoring...")
		return nil
	case setTrustedSequencerURLSignatureHash:
		log.Debug("SetTrustedSequencerURL event detected. Ignoring...")
		return nil
	case setTrustedSequencerSignatureHash:
		log.Debug("SetTrustedSequencer event detected. Ignoring...")
		return nil
	case verifyBatchesSignatureHash:
		log.Debug("VerifyBatches event detected. Ignoring...")
		return nil
	case sequenceForceBatchesSignatureHash:
		log.Debug("SequenceForceBatches event detected. Ignoring...")
		return nil
	case forceBatchSignatureHash:
		log.Debug("ForceBatch event detected. Ignoring...")
		return nil
	case sequenceBatchesSignatureHash:
		log.Debug("SequenceBatches event detected. Ignoring...")
		return nil
	case acceptAdminRoleSignatureHash:
		log.Debug("AcceptAdminRole event detected. Ignoring...")
		return nil
	case transferAdminRoleSignatureHash:
		log.Debug("TransferAdminRole event detected. Ignoring...")
		return nil
	}
	log.Warnf("Event not registered: %+v", vLog)
	return nil
}

func (etherMan *Client) updateGlobalExitRootEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("UpdateGlobalExitRoot event detected. Processing...")
	return etherMan.processUpdateGlobalExitRootEvent(ctx, vLog.Topics[1], vLog.Topics[2], vLog, blocks, blocksOrder)
}

func (etherMan *Client) updateL1InfoTreeEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("UpdateL1InfoTree event detected")
	globalExitRoot, err := etherMan.PolygonZkEVMGlobalExitRoot.ParseUpdateL1InfoTree(vLog)
	if err != nil {
		return err
	}
	return etherMan.processUpdateGlobalExitRootEvent(ctx, globalExitRoot.MainnetExitRoot, globalExitRoot.RollupExitRoot, vLog, blocks, blocksOrder)
}

func (etherMan *Client) processUpdateGlobalExitRootEvent(ctx context.Context, mainnetExitRoot, rollupExitRoot common.Hash, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	var gExitRoot GlobalExitRoot
	gExitRoot.ExitRoots = make([]common.Hash, 0)
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, mainnetExitRoot)
	gExitRoot.ExitRoots = append(gExitRoot.ExitRoots, rollupExitRoot)
	gExitRoot.GlobalExitRoot = hash(mainnetExitRoot, rollupExitRoot)
	gExitRoot.BlockNumber = vLog.BlockNumber

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		t := time.Unix(int64(fullBlock.Time), 0)
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
	log.Debug("Deposit event detected. Processing...")
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

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
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

func (etherMan *Client) oldClaimEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("Old claim event detected. Processing...")
	c, err := etherMan.OldPolygonBridge.ParseClaimEvent(vLog)
	if err != nil {
		return err
	}
	return etherMan.claimEvent(ctx, vLog, blocks, blocksOrder, c.Amount, c.DestinationAddress, c.OriginAddress, uint(c.Index), uint(c.OriginNetwork), 0, false)
}

func (etherMan *Client) newClaimEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("New claim event detected. Processing...")
	c, err := etherMan.PolygonBridge.ParseClaimEvent(vLog)
	if err != nil {
		return err
	}
	mainnetFlag, rollupIndex, localExitRootIndex, err := decodeGlobalIndex(c.GlobalIndex)
	if err != nil {
		return err
	}
	return etherMan.claimEvent(ctx, vLog, blocks, blocksOrder, c.Amount, c.DestinationAddress, c.OriginAddress, uint(localExitRootIndex), uint(c.OriginNetwork), rollupIndex, mainnetFlag)
}

func (etherMan *Client) claimEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order, amount *big.Int, destinationAddress, originAddress common.Address, Index uint, originNetwork uint, rollupIndex uint64, mainnetFlag bool) error {
	var claim Claim
	claim.Amount = amount
	claim.DestinationAddress = destinationAddress
	claim.Index = Index
	claim.OriginalNetwork = originNetwork
	claim.OriginalAddress = originAddress
	claim.BlockNumber = vLog.BlockNumber
	claim.TxHash = vLog.TxHash
	claim.RollupIndex = rollupIndex
	claim.MainnetFlag = mainnetFlag

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
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
	log.Debug("TokenWrapped event detected. Processing...")
	tw, err := etherMan.PolygonBridge.ParseNewWrappedToken(vLog)
	if err != nil {
		return err
	}
	var tokenWrapped TokenWrapped
	tokenWrapped.OriginalNetwork = uint(tw.OriginNetwork)
	tokenWrapped.OriginalTokenAddress = tw.OriginTokenAddress
	tokenWrapped.WrappedTokenAddress = tw.WrappedTokenAddress
	tokenWrapped.BlockNumber = vLog.BlockNumber

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
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

func prepareBlock(vLog types.Log, t time.Time, fullBlock *types.Header) Block {
	var block Block
	block.BlockNumber = vLog.BlockNumber
	block.BlockHash = vLog.BlockHash
	block.ParentHash = fullBlock.ParentHash
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

func (etherMan *Client) verifyBatchesTrustedAggregatorEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("VerifyBatchesTrustedAggregator event detected. Processing...")
	vb, err := etherMan.PolygonRollupManager.ParseVerifyBatchesTrustedAggregator(vLog)
	if err != nil {
		log.Error("error parsing verifyBatchesTrustedAggregator event. Error: ", err)
		return err
	}
	return etherMan.verifyBatches(ctx, vLog, blocks, blocksOrder, uint(vb.RollupID), vb.NumBatch, vb.StateRoot, vb.ExitRoot, vb.Aggregator)
}

func (etherMan *Client) verifyBatchesEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("RollupManagerVerifyBatches event detected. Processing...")
	vb, err := etherMan.PolygonRollupManager.ParseVerifyBatches(vLog)
	if err != nil {
		log.Error("error parsing VerifyBatches event. Error: ", err)
		return err
	}
	return etherMan.verifyBatches(ctx, vLog, blocks, blocksOrder, uint(vb.RollupID), vb.NumBatch, vb.StateRoot, vb.ExitRoot, vb.Aggregator)
}

func (etherMan *Client) verifyBatches(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order, rollupID uint, batchNum uint64, stateRoot, localExitRoot common.Hash, aggregator common.Address) error {
	var verifyBatch VerifiedBatch
	verifyBatch.BlockNumber = vLog.BlockNumber
	verifyBatch.BatchNumber = batchNum
	verifyBatch.RollupID = rollupID
	verifyBatch.LocalExitRoot = localExitRoot
	verifyBatch.TxHash = vLog.TxHash
	verifyBatch.StateRoot = stateRoot
	verifyBatch.Aggregator = aggregator

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
		block.VerifiedBatches = append(block.VerifiedBatches, verifyBatch)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].VerifiedBatches = append((*blocks)[len(*blocks)-1].VerifiedBatches, verifyBatch)
	} else {
		log.Error("Error processing verifyBatch event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing verifyBatch event")
	}
	or := Order{
		Name: VerifyBatchOrder,
		Pos:  len((*blocks)[len(*blocks)-1].VerifiedBatches) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) GetRollupID() uint {
	return uint(etherMan.RollupID)
}

func decodeGlobalIndex(globalIndex *big.Int) (bool, uint64, uint64, error) {
	const lengthGlobalIndexInBytes = 32
	var buf [32]byte
	gIBytes := globalIndex.FillBytes(buf[:])
	if len(gIBytes) != lengthGlobalIndexInBytes {
		return false, 0, 0, fmt.Errorf("invalid globaIndex length. Should be 32. Current length: %d", len(gIBytes))
	}
	mainnetFlag := big.NewInt(0).SetBytes([]byte{gIBytes[23]}).Uint64() == 1
	rollupIndex := big.NewInt(0).SetBytes(gIBytes[24:28])
	localRootIndex := big.NewInt(0).SetBytes(gIBytes[29:32])
	return mainnetFlag, rollupIndex.Uint64(), localRootIndex.Uint64(), nil
}

func GenerateGlobalIndex(mainnetFlag bool, rollupIndex uint, localExitRootIndex uint) *big.Int {
	var (
		globalIndexBytes []byte
		buf              [4]byte
	)
	if mainnetFlag {
		globalIndexBytes = append(globalIndexBytes, big.NewInt(1).Bytes()...)
		ri := big.NewInt(0).FillBytes(buf[:])
		globalIndexBytes = append(globalIndexBytes, ri...)
	} else {
		ri := big.NewInt(0).SetUint64(uint64(rollupIndex)).FillBytes(buf[:])
		globalIndexBytes = append(globalIndexBytes, ri...)
	}
	leri := big.NewInt(0).SetUint64(uint64(localExitRootIndex)).FillBytes(buf[:])
	globalIndexBytes = append(globalIndexBytes, leri...)
	return big.NewInt(0).SetBytes(globalIndexBytes)
}

func (etherMan *Client) createNewRollupEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("CreateNewRollup event detected. Processing...")
	rollup, err := etherMan.PolygonRollupManager.ParseCreateNewRollup(vLog)
	if err != nil {
		return err
	}
	if rollup.RollupID != etherMan.RollupID {
		return nil
	}

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
		block.ActivateEtrog = append(block.ActivateEtrog, true)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].ActivateEtrog = append((*blocks)[len(*blocks)-1].ActivateEtrog, true)
	} else {
		log.Error("Error processing TokenWrapped event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing TokenWrapped event")
	}
	or := Order{
		Name: ActivateEtrogOrder,
		Pos:  len((*blocks)[len(*blocks)-1].ActivateEtrog) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) AddExistingRollupEvent(ctx context.Context, vLog types.Log, blocks *[]Block, blocksOrder *map[common.Hash][]Order) error {
	log.Debug("AddExistingRollup event detected. Processing...")
	rollup, err := etherMan.PolygonRollupManager.ParseAddExistingRollup(vLog)
	if err != nil {
		return err
	}
	if rollup.RollupID != etherMan.RollupID {
		return nil
	}

	if len(*blocks) == 0 || ((*blocks)[len(*blocks)-1].BlockHash != vLog.BlockHash || (*blocks)[len(*blocks)-1].BlockNumber != vLog.BlockNumber) {
		fullBlock, err := etherMan.EtherClient.HeaderByHash(ctx, vLog.BlockHash)
		if err != nil {
			return fmt.Errorf("error getting hashParent. BlockNumber: %d. Error: %v", vLog.BlockNumber, err)
		}
		block := prepareBlock(vLog, time.Unix(int64(fullBlock.Time), 0), fullBlock)
		block.ActivateEtrog = append(block.ActivateEtrog, true)
		*blocks = append(*blocks, block)
	} else if (*blocks)[len(*blocks)-1].BlockHash == vLog.BlockHash && (*blocks)[len(*blocks)-1].BlockNumber == vLog.BlockNumber {
		(*blocks)[len(*blocks)-1].ActivateEtrog = append((*blocks)[len(*blocks)-1].ActivateEtrog, true)
	} else {
		log.Error("Error processing TokenWrapped event. BlockHash:", vLog.BlockHash, ". BlockNumber: ", vLog.BlockNumber)
		return fmt.Errorf("error processing TokenWrapped event")
	}
	or := Order{
		Name: ActivateEtrogOrder,
		Pos:  len((*blocks)[len(*blocks)-1].ActivateEtrog) - 1,
	}
	(*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash] = append((*blocksOrder)[(*blocks)[len(*blocks)-1].BlockHash], or)
	return nil
}

func (etherMan *Client) SendCompressedClaims(auth *bind.TransactOpts, compressedTxData []byte) (*types.Transaction, error) {
	claimTx, err := etherMan.ClaimCompressor.SendCompressedClaims(auth, compressedTxData)
	if err != nil {
		log.Error("failed to call SMC SendCompressedClaims: %v", err)
		return nil, err
	}
	return claimTx, err
}

func (etherMan *Client) CompressClaimCall(mainnetExitRoot, rollupExitRoot common.Hash, claimData []claimcompressor.ClaimCompressorCompressClaimCallData) ([]byte, error) {
	compressedData, err := etherMan.ClaimCompressor.CompressClaimCall(&bind.CallOpts{Pending: false}, mainnetExitRoot, rollupExitRoot, claimData)
	if err != nil {
		log.Errorf("fails call to claimCompressorSMC. Error: %v", err)
		return []byte{}, nil
	}
	return compressedData, nil
}
