package claimtxman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
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
	storage         storageInterface
	auth            *bind.TransactOpts
	nonceCache      *lru.Cache[string, uint64]
}

// NewClaimTxManager creates a new claim transaction manager.
func NewClaimTxManager(cfg Config, chExitRootEvent chan *etherman.GlobalExitRoot, l2NodeURL string, l2NetworkID uint, l2BridgeAddr common.Address, bridgeService bridgeServiceInterface, storage interface{}) (*ClaimTxManager, error) {
	client, err := utils.NewClient(context.Background(), l2NodeURL, l2BridgeAddr)
	if err != nil {
		return nil, err
	}
	cache, err := lru.New[string, uint64](int(cacheSize))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	auth, err := client.GetSignerFromKeystore(ctx, cfg.PrivateKey)
	return &ClaimTxManager{
		ctx:             ctx,
		cancel:          cancel,
		l2Node:          client,
		l2NetworkID:     l2NetworkID,
		bridgeService:   bridgeService,
		cfg:             cfg,
		chExitRootEvent: chExitRootEvent,
		storage:         storage.(storageInterface),
		auth:            auth,
		nonceCache:      cache,
	}, err
}

// Start will start the tx management, reading txs from storage,
// send then to the blockchain and keep monitoring them until they
// get mined
func (tm *ClaimTxManager) Start() {
	for {
		select {
		case <-tm.ctx.Done():
			return
		case ger := <-tm.chExitRootEvent:
			go func() {
				err := tm.updateDepositsStatus(ger)
				if err != nil {
					log.Errorf("failed to update deposits status: %v", err)
				}
			}()
		case <-time.After(tm.cfg.FrequencyToMonitorTxs.Duration):
			err := tm.monitorTxs(context.Background())
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
	if ger.BlockID != 0 { // L2 exit root is updated
		log.Infof("Rollup exitroot %v is updated", ger.ExitRoots[1])
		if err := tm.storage.UpdateL2DepositsStatus(tm.ctx, ger.ExitRoots[1][:], dbTx); err != nil {
			return err
		}
	} else { // L1 exit root is updated in the trusted state
		log.Infof("Mainnet exitroot %v is updated", ger.ExitRoots[0])
		deposits, err := tm.storage.UpdateL1DepositsStatus(tm.ctx, ger.ExitRoots[0][:], dbTx)
		if err != nil {
			return err
		}
		for _, deposit := range deposits {
			claimHash, err := tm.bridgeService.GetDepositStatus(tm.ctx, deposit.DepositCount, deposit.DestinationNetwork)
			if err != nil {
				return err
			}
			if len(claimHash) > 0 || deposit.LeafType == LeafTypeMessage {
				continue
			}
			log.Infof("create the claim tx for the deposit %d", deposit.DepositCount)
			ger, proves, err := tm.bridgeService.GetClaimProof(deposit.DepositCount, deposit.NetworkID, dbTx)
			if err != nil {
				return err
			}
			var mtProves [mtHeight][keyLen]byte
			for i := 0; i < mtHeight; i++ {
				mtProves[i] = proves[i]
			}
			to, data, err := tm.l2Node.BuildSendClaim(tm.ctx, deposit, mtProves,
				&etherman.GlobalExitRoot{
					ExitRoots: []common.Hash{
						ger.ExitRoots[0],
						ger.ExitRoots[1],
					}},
				tm.auth)
			if err != nil {
				return err
			}
			if err = tm.addClaimTx(deposit.DepositCount, deposit.BlockID, tm.auth.From, to, nil, data, dbTx); err != nil {
				return err
			}
		}
	}
	err = tm.storage.Commit(tm.ctx, dbTx)
	if err != nil {
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
		}
		log.Fatalf("AddClaimTx committing dbTx, err: %s", err.Error())
	}
	return nil
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

func (tm *ClaimTxManager) addClaimTx(id uint, blockID uint64, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
	// get next nonce
	nonce, err := tm.getNextNonce(from)
	if err != nil {
		err := fmt.Errorf("failed to get current nonce: %w", err)
		log.Errorf(err.Error())
		return err
	}
	// get gas
	gas, err := tm.l2Node.EstimateGas(tm.ctx, ethereum.CallMsg{
		From:  from,
		To:    to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		log.Errorf("failed to estimate gas: %w, data: %v", err, common.Bytes2Hex(data))
		return err
	}

	// create monitored tx
	mTx := ctmtypes.MonitoredTx{
		ID: id, BlockID: blockID, From: from, To: to,
		Nonce: nonce, Value: value, Data: data,
		Gas: gas, Status: ctmtypes.MonitoredTxStatusCreated,
	}

	// add to storage
	err = tm.storage.AddClaimTx(tm.ctx, mTx, dbTx)
	if err != nil {
		err := fmt.Errorf("failed to add tx to get monitored: %w", err)
		log.Errorf(err.Error())
		return err
	}

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
		return fmt.Errorf("failed to get created monitored txs: %v", err)
	}

	log.Infof("found %v monitored tx to process", len(mTxs))
	for _, mTx := range mTxs {
		mTx := mTx // force variable shadowing to avoid pointer conflicts
		mTxLog := log.WithFields("monitoredTx", mTx.ID)
		mTxLog.Info("processing")

		// check if any of the txs in the history was mined
		mined := false
		var receipt *types.Receipt
		hasFailedReceipts := false
		allHistoryTxMined := true
		receiptSuccessful := false

		for txHash := range mTx.History {
			mined, receipt, err = tm.l2Node.CheckTxWasMined(ctx, txHash)
			if err != nil {
				mTxLog.Errorf("failed to check if tx %v was mined: %v", txHash.String(), err)
				continue
			}

			// if the tx is not mined yet, check that not all the tx were mined and go to the next
			if !mined {
				// check if the tx is in the pending pool
				_, _, err = tm.l2Node.TransactionByHash(ctx, txHash)
				if errors.Is(err, ethereum.NotFound) {
					mTxLog.Errorf("tx %v was not found in the pending pool", txHash.String())
					hasFailedReceipts = true
					continue
				} else if err != nil {
					mTxLog.Errorf("failed to get tx %v: %v", txHash.String(), err)
					continue
				}

				allHistoryTxMined = false
				continue
			}

			// if the tx was mined successfully we can break the loop and proceed
			if receipt.Status == types.ReceiptStatusSuccessful {
				mTxLog.Infof("tx %v was mined successfully", txHash.String())
				receiptSuccessful = true
				block, err := tm.l2Node.BlockByNumber(ctx, receipt.BlockNumber)
				if err != nil {
					mTxLog.Errorf("failed to get receipt block: %v", err)
					continue
				}
				mTx.BlockID, err = tm.storage.AddBlock(ctx, &etherman.Block{
					NetworkID:   tm.l2NetworkID,
					BlockNumber: block.Number().Uint64(),
					BlockHash:   block.Hash(),
					ParentHash:  block.ParentHash(),
					ReceivedAt:  block.ReceivedAt,
				}, dbTx)
				if err != nil {
					mTxLog.Errorf("failed to add receipt block: %v", err)
					continue
				}
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
			mTxLog.Infof("marked as failed because reached the history size limit")
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
				err := tm.ReviewMonitoredTx(ctx, &mTx)
				if err != nil {
					mTxLog.Errorf("failed to review monitored tx: %v", err)
					continue
				}
			}

			var signedTx *types.Transaction
			// rebuild transaction
			tx := mTx.Tx()
			mTxLog.Debugf("unsigned tx %v created for monitored tx", tx.Hash().String())

			// sign tx
			signedTx, err = tm.auth.Signer(mTx.From, tx)
			if err != nil {
				mTxLog.Errorf("failed to sign tx %v created from monitored tx: %v", tx.Hash().String(), err)
				continue
			}
			mTxLog.Debugf("signed tx %v created", signedTx.Hash().String())

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
					if strings.Contains(err.Error(), "nonce") {
						mTxLog.Infof("nonce error detected, resetting nonce cache")
						tm.nonceCache.Remove(mTx.From.Hex())
					}
					mTx.RemoveHistory(signedTx)
					mTxLog.Errorf("failed to send tx %v to network: %v", signedTx.Hash().String(), err)
				}
			} else {
				mTxLog.Infof("signed tx %v already found in the network for the monitored tx: %v", signedTx.Hash().String(), err)
			}

			// update monitored tx changes into storage
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx: %v", err)
				continue
			}
			mTxLog.Debugf("signed tx added to the monitored tx history")
		}
	}

	err = tm.storage.Commit(tm.ctx, dbTx)
	if err != nil {
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
		}
		log.Fatalf("UpdateClaimTx committing dbTx, err: %s", err.Error())
	}
	return nil
}

// ReviewMonitoredTx checks if tx needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (tm *ClaimTxManager) ReviewMonitoredTx(ctx context.Context, mTx *ctmtypes.MonitoredTx) error {
	mTxLog := log.WithFields("monitoredTx", mTx.ID)
	mTxLog.Debug("reviewing")
	// get gas
	gas, err := tm.l2Node.EstimateGas(ctx, ethereum.CallMsg{
		From:  mTx.From,
		To:    mTx.To,
		Value: mTx.Value,
		Data:  mTx.Data,
	})
	if err != nil {
		err := fmt.Errorf("failed to estimate gas: %w", err)
		mTxLog.Errorf(err.Error())
		return err
	}

	// check gas
	if gas > mTx.Gas {
		mTxLog.Infof("monitored tx gas updated from %v to %v", mTx.Gas, gas)
		mTx.Gas = gas
	}

	// check nonce
	nonce, err := tm.getNextNonce(mTx.From)
	if err != nil {
		err := fmt.Errorf("failed to get nonce: %w", err)
		mTxLog.Errorf(err.Error())
		return err
	}
	if nonce > mTx.Nonce {
		mTxLog.Infof("monitored tx nonce updated from %v to %v", mTx.Nonce, nonce)
		mTx.Nonce = nonce
	}

	return nil
}
