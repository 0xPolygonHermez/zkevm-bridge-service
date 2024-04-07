package claimtxman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/state/runtime"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v4"
)

const (
	maxHistorySize  = 10
	keyLen          = 32
	mtHeight        = 32
	cacheSize       = 1000
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
	storage         storageInterface
	auth            *bind.TransactOpts
	rollupID        uint
	nonceCache      *lru.Cache[string, uint64]
	synced          bool

	// XLayer
	updateDepositsL1Mutex sync.Mutex
	updateDepositsL2Mutex sync.Mutex
	isDone                bool
	// Producer to push the transaction status change to front end
	messagePushProducer messagepush.KafkaProducer
	redisStorage        redisstorage.RedisStorage
}

// NewClaimTxManager creates a new claim transaction manager.
func NewClaimTxManager(cfg Config, chExitRootEvent chan *etherman.GlobalExitRoot, chSynced chan uint, l2NodeURL string, l2NetworkID uint,
	l2BridgeAddr common.Address, bridgeService bridgeServiceInterface, storage interface{}, producer messagepush.KafkaProducer,
	redisStorage redisstorage.RedisStorage, rollupID uint) (*ClaimTxManager, error) {
	ctx := context.Background()
	client, err := utils.NewClient(ctx, l2NodeURL, l2BridgeAddr)
	if err != nil {
		return nil, err
	}
	cache, err := lru.New[string, uint64](int(cacheSize))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	auth, err := client.GetSignerFromKeystore(ctx, cfg.PrivateKey)
	return &ClaimTxManager{
		ctx:                 ctx,
		cancel:              cancel,
		l2Node:              client,
		l2NetworkID:         l2NetworkID,
		bridgeService:       bridgeService,
		cfg:                 cfg,
		chExitRootEvent:     chExitRootEvent,
		chSynced:            chSynced,
		storage:             storage.(storageInterface),
		auth:                auth,
		rollupID:            rollupID,
		nonceCache:          cache,
		messagePushProducer: producer,
		redisStorage:        redisStorage,
	}, err
}

// Start will start the tx management, reading txs from storage,
// send then to the blockchain and keep monitoring them until they
// get mined
func (tm *ClaimTxManager) Start() {
	ticker := time.NewTicker(tm.cfg.FrequencyToMonitorTxs.Duration)
	for {
		select {
		case <-tm.ctx.Done():
			return
		case netID := <-tm.chSynced:
			if netID == tm.l2NetworkID && !tm.synced {
				log.Info("NetworkID synced: ", netID)
				tm.synced = true
			}
		case ger := <-tm.chExitRootEvent:
			if tm.synced {
				log.Debug("UpdateDepositsStatus for ger: ", ger.GlobalExitRoot)
				go func() {
					err := tm.updateDepositsStatus(ger)
					if err != nil {
						log.Errorf("failed to update deposits status: %v", err)
					}
				}()
			} else {
				log.Infof("Waiting for networkID %d to be synced before processing deposits", tm.l2NetworkID)
			}
		case <-ticker.C:
			err := tm.monitorTxs(tm.ctx)
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
			ger, proof, rollupProof, err := tm.bridgeService.GetClaimProof(deposit.DepositCount, deposit.NetworkID, dbTx)
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
			tx, err := tm.l2Node.BuildSendClaimXLayer(tm.ctx, deposit, mtProof, mtRollupProof,
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
			if err = tm.addClaimTx(deposit.DepositCount, tm.auth.From, tx.To(), nil, tx.Data(), dbTx); err != nil {
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

func (tm *ClaimTxManager) getNextNonce(from common.Address) (uint64, error) {
	nonce, err := tm.l2Node.NonceAt(tm.ctx, from, nil)
	if err != nil {
		return 0, err
	}
	if tempNonce, found := tm.nonceCache.Get(from.Hex()); found {
		if tempNonce >= nonce {
			nonce = tempNonce + 1
		}
	}
	tm.nonceCache.Add(from.Hex(), nonce)
	return nonce, nil
}

func (tm *ClaimTxManager) addClaimTx(depositCount uint, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
	// get gas
	tx := ethereum.CallMsg{
		From:  from,
		To:    to,
		Value: value,
		Data:  data,
	}
	log.Debugf("addClaimTx deposit: %d", depositCount)
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
	nonce, err := tm.getNextNonce(from)
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
	}

	// add to storage
	err = tm.storage.AddClaimTx(tm.ctx, mTx, dbTx)
	if err != nil {
		err := fmt.Errorf("failed to add tx to get monitored: %v", err)
		log.Errorf("error adding claim tx to db. Error: %s", err.Error())
		return err
	}
	log.Debugf("addClaimTx successfully depositCount: %d", depositCount)

	return nil
}

// monitorTxs process all pending monitored tx
func (tm *ClaimTxManager) monitorTxs(ctx context.Context) error {
	dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
	if err != nil {
		return err
	}

	statusesFilter := []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatusCreated}
	mTxs, err := tm.storage.GetClaimTxsByStatus(ctx, statusesFilter, dbTx)
	if err != nil {
		log.Errorf("failed to get created monitored txs: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("claimtxman error rolling back state. RollbackErr: %s, err: %v", rollbackErr.Error(), err)
			return rollbackErr
		}
		return fmt.Errorf("failed to get created monitored txs: %v", err)
	}

	isResetNonce := false // it will reset the nonce in one cycle
	log.Infof("found %v monitored tx to process", len(mTxs))
	for _, mTx := range mTxs {
		mTx := mTx // force variable shadowing to avoid pointer conflicts
		mTxLog := log.WithFields("monitoredTx", mTx.DepositID)
		mTxLog.Infof("processing tx with nonce %d", mTx.Nonce)

		// check if any of the txs in the history was mined
		mined := false
		var receipt *types.Receipt
		hasFailedReceipts := false
		allHistoryTxMined := true
		receiptSuccessful := false

		for txHash := range mTx.History {
			mTxLog.Infof("Checking if tx %s is mined", txHash.String())
			mined, receipt, err = tm.l2Node.CheckTxWasMined(ctx, txHash)
			if err != nil {
				mTxLog.Errorf("failed to check if tx %s was mined: %v", txHash.String(), err)
				continue
			}

			// if the tx is not mined yet, check that not all the tx were mined and go to the next
			if !mined {
				// check if the tx is in the pending pool
				_, _, err = tm.l2Node.TransactionByHash(ctx, txHash)
				if err != nil {
					mTxLog.Errorf("error getting txByHash %s. Error: %v", txHash.String(), err)
					// Retry if the tx has not appeared in the pool yet.
					for i := 0; i < tm.cfg.RetryNumber && err != nil; i++ {
						mTxLog.Warn("waiting and retrying to find the tx in the pool. TxHash: %s. Error: %v", txHash.String(), err)
						time.Sleep(tm.cfg.RetryInterval.Duration)
						_, _, err = tm.l2Node.TransactionByHash(ctx, txHash)
					}
					if errors.Is(err, ethereum.NotFound) {
						mTxLog.Error("maximum retries and the tx is still missing in the pool. TxHash: ", txHash.String())
						hasFailedReceipts = true
						continue
					} else if err != nil {
						mTxLog.Errorf("failed to retry to get tx %s: %v", txHash.String(), err)
						continue
					}
				}
				log.Infof("tx: %s not mined yet", txHash.String())

				allHistoryTxMined = false
				continue
			}

			// if the tx was mined successfully we can break the loop and proceed
			if receipt.Status == types.ReceiptStatusSuccessful {
				mTxLog.Infof("tx %s was mined successfully", txHash.String())
				receiptSuccessful = true
				mTx.Status = ctmtypes.MonitoredTxStatusConfirmed
				// update monitored tx changes into storage
				err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
				if err != nil {
					mTxLog.Errorf("failed to update monitored tx when confirmed: %v", err)
				}
				break
			}

			// if the tx was mined but failed, we continue to consider it was not mined
			// and store the failed receipt to be used to check if nonce needs to be reviewed
			mined = false
			hasFailedReceipts = true
		}

		if receiptSuccessful {
			continue
		}

		// if the history size reaches the max history size, this means something is really wrong with
		// this Tx and we are not able to identify automatically, so we mark this as failed to let the
		// caller know something is not right and needs to be review and to avoid to monitor this
		// tx infinitely
		if allHistoryTxMined && len(mTx.History) >= maxHistorySize {
			mTx.Status = ctmtypes.MonitoredTxStatusFailed
			mTxLog.Infof("marked as failed because reached the history size limit (%d)", maxHistorySize)
			// update monitored tx changes into storage
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx when max history size limit reached: %v", err)
			}
			continue
		}

		// if we have failed receipts, this means at least one of the generated txs was mined
		// so maybe the current nonce was already consumed, then we need to check if there are
		// tx that were not mined yet, if so, we just need to wait, because maybe one of them
		// will get mined successfully
		if allHistoryTxMined {
			// in case of all tx were mined and none of them were mined successfully, we need to
			// review the tx information
			if hasFailedReceipts {
				mTxLog.Infof("monitored tx needs to be updated")
				err := tm.ReviewMonitoredTx(ctx, &mTx, true)
				if err != nil {
					mTxLog.Errorf("failed to review monitored tx: %v", err)
					continue
				}
			}

			// GasPrice is set here to use always the proper and most accurate value right before sending it to L2
			gasPrice, err := tm.l2Node.SuggestGasPrice(ctx)
			if err != nil {
				mTxLog.Errorf("failed to get suggested gasPrice. Error: %v", err)
				continue
			}
			//Multiply gasPrice by 10 to increase the efficiency of the tx in the sequence
			mTx.GasPrice = big.NewInt(0).Mul(gasPrice, big.NewInt(10)) //nolint:gomnd
			log.Infof("Using gasPrice: %s. The gasPrice suggested by the network is %s", mTx.GasPrice.String(), gasPrice.String())

			// rebuild transaction
			tx := mTx.Tx()
			mTxLog.Debugf("unsigned tx created for monitored tx")

			var signedTx *types.Transaction
			// sign tx
			signedTx, err = tm.auth.Signer(mTx.From, tx)
			if err != nil {
				mTxLog.Errorf("failed to sign tx %v created from monitored tx: %v", tx.Hash().String(), err)
				continue
			}
			mTxLog.Debugf("signed tx %v created using gasPrice: %s", signedTx.Hash().String(), signedTx.GasPrice().String())

			// add tx to monitored tx history
			err = mTx.AddHistory(signedTx)
			if errors.Is(err, ctmtypes.ErrAlreadyExists) {
				mTxLog.Infof("signed tx already existed in the history")
			} else if err != nil {
				mTxLog.Errorf("failed to add signed tx to monitored tx history: %v", err)
				continue
			}

			// check if the tx is already in the network, if not, send it
			_, _, err = tm.l2Node.TransactionByHash(ctx, signedTx.Hash())
			if errors.Is(err, ethereum.NotFound) {
				err := tm.l2Node.SendTransaction(ctx, signedTx)
				if err != nil {
					mTxLog.Errorf("failed to send tx %s to network: %v", signedTx.Hash().String(), err)
					var reviewNonce bool
					if strings.Contains(err.Error(), "nonce") {
						mTxLog.Infof("nonce error detected, Nonce used: %d", signedTx.Nonce())
						if !isResetNonce {
							isResetNonce = true
							tm.nonceCache.Remove(mTx.From.Hex())
							mTxLog.Infof("nonce cache cleared for address %v", mTx.From.Hex())
						}
						reviewNonce = true
					}
					mTx.RemoveHistory(signedTx)
					// we should rebuild the monitored tx to fix the nonce
					err := tm.ReviewMonitoredTx(ctx, &mTx, reviewNonce)
					if err != nil {
						mTxLog.Errorf("failed to review monitored tx: %v", err)
					}
				}
			} else if err != nil && !errors.Is(err, ethereum.NotFound) {
				mTxLog.Error("unexpected error getting TransactionByHash. Error: ", err)
			} else {
				mTxLog.Infof("signed tx %v already found in the network for the monitored tx.", signedTx.Hash().String())
			}

			// update monitored tx changes into storage
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx: %v", err)
				continue
			}
			mTxLog.Infof("signed tx %s added to the monitored tx history", signedTx.Hash().String())
		}
	}

	err = tm.storage.Commit(tm.ctx, dbTx)
	if err != nil {
		log.Errorf("UpdateClaimTx committing dbTx, err: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Errorf("claimtxman error rolling back state. RollbackErr: %s, err: %v", rollbackErr.Error(), err)
			return rollbackErr
		}
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
		nonce, err := tm.getNextNonce(mTx.From)
		if err != nil {
			err := fmt.Errorf("failed to get nonce: %v", err)
			mTxLog.Errorf(err.Error())
			return err
		}
		mTxLog.Infof("monitored tx nonce from %v to %v", mTx.Nonce, nonce)
		if nonce != mTx.Nonce {
			mTxLog.Infof("monitored tx nonce updated from %v to %v", mTx.Nonce, nonce)
			mTx.Nonce = nonce
		}
	}

	return nil
}
