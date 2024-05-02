package claimtxman

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/state/runtime"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v4"
)

const (
	maxHistorySize  = 10
	keyLen          = 32
	mtHeight        = 32
	LeafTypeMessage = uint8(1)
)

// ClaimTxManager is the claim transaction manager for L2.
type ClaimTxManager struct {
	ctx    context.Context
	cancel context.CancelFunc

	// client is the ethereum client
	l2Node          *utils.Client
	l2NetworkID     uint
	bridgeService   bridgeServiceInterface
	cfg             Config
	chExitRootEvent chan *etherman.GlobalExitRoot
	chSynced        chan uint
	storage         StorageInterface
	auth            *bind.TransactOpts
	rollupID        uint
	synced          bool
	nonceCache      *NonceCache
	monitorTxs      types.TxMonitorer
}

// NewClaimTxManager creates a new claim transaction manager.
func NewClaimTxManager(ctx context.Context, cfg Config, chExitRootEvent chan *etherman.GlobalExitRoot,
	chSynced chan uint,
	l2NodeURL string,
	l2NetworkID uint,
	l2BridgeAddr common.Address,
	bridgeService bridgeServiceInterface,
	storage interface{},
	rollupID uint,
	etherMan EthermanI,
	nonceCache *NonceCache,
	auth *bind.TransactOpts) (*ClaimTxManager, error) {
	client, err := utils.NewClient(ctx, l2NodeURL, l2BridgeAddr)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)

	var monitorTx ctmtypes.TxMonitorer
	if cfg.GroupingClaims.Enabled {
		log.Info("ClaimTxManager working in compressor mode to group claim txs")
		monitorTx = NewMonitorCompressedTxs(ctx, storage.(StorageCompressedInterface), client, cfg, nonceCache, auth, etherMan, utils.NewTimeProviderSystemLocalTime())
	} else {
		log.Info("ClaimTxManager working in regular mode to send claim txs individually")
		monitorTx = NewMonitorTxs(ctx, storage.(StorageInterface), client, cfg, nonceCache, auth)
	}
	return &ClaimTxManager{
		ctx:             ctx,
		cancel:          cancel,
		l2Node:          client,
		l2NetworkID:     l2NetworkID,
		bridgeService:   bridgeService,
		cfg:             cfg,
		chExitRootEvent: chExitRootEvent,
		chSynced:        chSynced,
		storage:         storage.(StorageInterface),
		auth:            auth,
		rollupID:        rollupID,
		nonceCache:      nonceCache,
		monitorTxs:      monitorTx,
	}, err
}

// Start will start the tx management, reading txs from storage,
// send then to the blockchain and keep monitoring them until they
// get mined
func (tm *ClaimTxManager) Start() {
	ticker := time.NewTicker(tm.cfg.FrequencyToMonitorTxs.Duration)
	compressorTicker := time.NewTicker(tm.cfg.GroupingClaims.FrequencyToProcessCompressedClaims.Duration)
	var ger = &etherman.GlobalExitRoot{}
	var latestProcessedGer common.Hash
	for {
		select {
		case <-tm.ctx.Done():
			return
		case netID := <-tm.chSynced:
			if netID == tm.l2NetworkID && !tm.synced {
				log.Info("NetworkID synced: ", netID)
				tm.synced = true
			}
		case ger = <-tm.chExitRootEvent:
			if tm.synced {
				log.Debug("UpdateDepositsStatus for ger: ", ger.GlobalExitRoot)
				if tm.cfg.GroupingClaims.Enabled {
					log.Debug("Ger value updated and ready to be processed...")
					continue
				}
				go func() {
					err := tm.updateDepositsStatus(ger)
					if err != nil {
						log.Errorf("failed to update deposits status: %v", err)
					}
				}()
			} else {
				log.Infof("Waiting for networkID %d to be synced before processing deposits", tm.l2NetworkID)
			}
		case <-compressorTicker.C:
			if tm.synced && tm.cfg.GroupingClaims.Enabled && ger.GlobalExitRoot != latestProcessedGer {
				log.Info("Processing deposits for ger: ", ger.GlobalExitRoot)
				go func() {
					err := tm.updateDepositsStatus(ger)
					if err != nil {
						log.Errorf("failed to update deposits status: %v", err)
					}
				}()
				latestProcessedGer = ger.GlobalExitRoot
			}
		case <-ticker.C:
			err := tm.monitorTxs.MonitorTxs(tm.ctx)
			if err != nil {
				log.Errorf("failed to monitor txs: %v", err)
			}
		}
	}
}

func (tm *ClaimTxManager) updateDepositsStatus(ger *etherman.GlobalExitRoot) error {
	dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
	if err != nil {
		return err
	}
	err = tm.processDepositStatus(ger, dbTx)
	if err != nil {
		log.Errorf("error processing ger. Error: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("claimtxman error rolling back state. RollbackErr: %v, err: %s", rollbackErr, err.Error())
			return rollbackErr
		}
		return err
	}
	err = tm.storage.Commit(tm.ctx, dbTx)
	if err != nil {
		log.Errorf("AddClaimTx committing dbTx. Err: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
		}
		log.Fatalf("AddClaimTx committing dbTx, err: %s", err.Error())
	}
	return nil
}

func (tm *ClaimTxManager) processDepositStatus(ger *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	if ger.BlockID != 0 { // L2 exit root is updated
		log.Infof("Rollup exitroot %v is updated", ger.ExitRoots[1])
		if err := tm.storage.UpdateL2DepositsStatus(tm.ctx, ger.ExitRoots[1][:], tm.rollupID, tm.l2NetworkID, dbTx); err != nil {
			log.Errorf("error updating L2DepositsStatus. Error: %v", err)
			return err
		}
	} else { // L1 exit root is updated in the trusted state
		log.Infof("Mainnet exitroot %v is updated", ger.ExitRoots[0])
		deposits, err := tm.storage.UpdateL1DepositsStatus(tm.ctx, ger.ExitRoots[0][:], dbTx)
		if err != nil {
			log.Errorf("error getting and updating L1DepositsStatus. Error: %v", err)
			return err
		}
		for _, deposit := range deposits {
			if tm.l2NetworkID != deposit.DestinationNetwork {
				log.Infof("Ignoring deposit: %d: dest_net: %d, we are:%d", deposit.DepositCount, deposit.DestinationNetwork, tm.l2NetworkID)
				continue
			}

			claimHash, err := tm.bridgeService.GetDepositStatus(tm.ctx, deposit.DepositCount, deposit.DestinationNetwork)
			if err != nil {
				log.Errorf("error getting deposit status for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			if len(claimHash) > 0 || deposit.LeafType == LeafTypeMessage && !tm.isDepositMessageAllowed(deposit) {
				log.Infof("Ignoring deposit: %d, leafType: %d, claimHash: %s, deposit.OriginalAddress: %s", deposit.DepositCount, deposit.LeafType, claimHash, deposit.OriginalAddress.String())
				continue
			}

			log.Infof("create the claim tx for the deposit %d", deposit.DepositCount)
			ger, proof, rollupProof, err := tm.bridgeService.GetClaimProofForCompressed(ger.GlobalExitRoot, deposit.DepositCount, deposit.NetworkID, dbTx)
			if err != nil {
				log.Errorf("error getting Claim Proof for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			var (
				mtProof       [mtHeight][keyLen]byte
				mtRollupProof [mtHeight][keyLen]byte
			)
			for i := 0; i < mtHeight; i++ {
				mtProof[i] = proof[i]
				mtRollupProof[i] = rollupProof[i]
			}
			tx, err := tm.l2Node.BuildSendClaim(tm.ctx, deposit, mtProof, mtRollupProof,
				&etherman.GlobalExitRoot{
					ExitRoots: []common.Hash{
						ger.ExitRoots[0],
						ger.ExitRoots[1],
					}}, 1, 1, 1, tm.rollupID,
				tm.auth)
			if err != nil {
				log.Errorf("error BuildSendClaim tx for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			if err = tm.addClaimTx(deposit.DepositCount, tm.auth.From, tx.To(), nil, tx.Data(), ger.GlobalExitRoot, dbTx); err != nil {
				log.Errorf("error adding claim tx for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
		}
	}
	return nil
}

func (tm *ClaimTxManager) isDepositMessageAllowed(deposit *etherman.Deposit) bool {
	for _, addr := range tm.cfg.AuthorizedClaimMessageAddresses {
		if deposit.OriginalAddress == addr {
			log.Infof("MessageBridge from authorized account detected: %+v, account: %s", deposit, addr.String())
			return true
		}
	}
	log.Infof("MessageBridge Not authorized. DepositCount: %d", deposit.DepositCount)
	return false
}

func (tm *ClaimTxManager) addClaimTx(depositCount uint, from common.Address, to *common.Address, value *big.Int, data []byte, ger common.Hash, dbTx pgx.Tx) error {
	// get gas
	tx := ethereum.CallMsg{
		From:  from,
		To:    to,
		Value: value,
		Data:  data,
	}
	gas, err := tm.l2Node.EstimateGas(tm.ctx, tx)
	for i := 1; err != nil && err.Error() != runtime.ErrExecutionReverted.Error() && i < tm.cfg.RetryNumber; i++ {
		log.Warnf("error while doing gas estimation. Retrying... Error: %v, Data: %s", err, common.Bytes2Hex(data))
		time.Sleep(tm.cfg.RetryInterval.Duration)
		gas, err = tm.l2Node.EstimateGas(tm.ctx, tx)
	}
	if err != nil {
		log.Errorf("failed to estimate gas. Ignoring tx... Error: %v, data: %s", err, common.Bytes2Hex(data))
		return nil
	}
	// get next nonce
	nonce, err := tm.nonceCache.GetNextNonce(from)
	if err != nil {
		err := fmt.Errorf("failed to get current nonce: %v", err)
		log.Errorf("error getting next nonce. Error: %s", err.Error())
		return err
	}

	// create monitored tx
	mTx := ctmtypes.MonitoredTx{
		DepositID: depositCount, From: from, To: to,
		Nonce: nonce, Value: value, Data: data,
		Gas: gas, Status: ctmtypes.MonitoredTxStatusCreated,
		GlobalExitRoot: ger,
	}

	// add to storage
	err = tm.storage.AddClaimTx(tm.ctx, mTx, dbTx)
	if err != nil {
		err := fmt.Errorf("failed to add tx to get monitored: %v", err)
		log.Errorf("error adding claim tx to db. Error: %s", err.Error())
		return err
	}

	return nil
}

// ReviewMonitoredTx checks if tx needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (tm *ClaimTxManager) ReviewMonitoredTx(ctx context.Context, mTx *ctmtypes.MonitoredTx, reviewNonce bool) error {
	mTxLog := log.WithFields("monitoredTx", mTx.DepositID)
	mTxLog.Debug("reviewing")
	// get gas
	tx := ethereum.CallMsg{
		From:  mTx.From,
		To:    mTx.To,
		Value: mTx.Value,
		Data:  mTx.Data,
	}
	gas, err := tm.l2Node.EstimateGas(ctx, tx)
	for i := 1; err != nil && err.Error() != runtime.ErrExecutionReverted.Error() && i < tm.cfg.RetryNumber; i++ {
		mTxLog.Warnf("error during gas estimation. Retrying... Error: %v, Data: %s", err, common.Bytes2Hex(tx.Data))
		time.Sleep(tm.cfg.RetryInterval.Duration)
		gas, err = tm.l2Node.EstimateGas(tm.ctx, tx)
	}
	if err != nil {
		err := fmt.Errorf("failed to estimate gas. Error: %v, Data: %s", err, common.Bytes2Hex(tx.Data))
		mTxLog.Errorf("error: %s", err.Error())
		return err
	}

	// check gas
	if gas > mTx.Gas {
		mTxLog.Infof("monitored tx gas updated from %v to %v", mTx.Gas, gas)
		mTx.Gas = gas
	}

	if reviewNonce {
		// check nonce
		nonce, err := tm.nonceCache.GetNextNonce(mTx.From)
		if err != nil {
			err := fmt.Errorf("failed to get nonce: %v", err)
			mTxLog.Errorf(err.Error())
			return err
		}
		if nonce > mTx.Nonce {
			mTxLog.Infof("monitored tx nonce updated from %v to %v", mTx.Nonce, nonce)
			mTx.Nonce = nonce
		}
	}

	return nil
}
