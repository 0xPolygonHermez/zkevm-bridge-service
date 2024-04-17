package claimtxman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/state/runtime"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
)

type MonitorTxs struct {
	storage StorageInterface
	ctx     context.Context
	// client is the ethereum client
	l2Node     *utils.Client
	cfg        Config
	nonceCache *NonceCache
	auth       *bind.TransactOpts
}

func NewMonitorTxs(ctx context.Context,
	storage StorageInterface,
	l2Node *utils.Client,
	cfg Config,
	nonceCache *NonceCache,
	auth *bind.TransactOpts) *MonitorTxs {
	return &MonitorTxs{
		storage:    storage.(StorageInterface),
		ctx:        ctx,
		l2Node:     l2Node,
		cfg:        cfg,
		nonceCache: nonceCache,
		auth:       auth,
	}
}

// MonitorTxs process all pending monitored tx
func (tm *MonitorTxs) MonitorTxs(ctx context.Context) error {
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

		// if the tx is not mined yet, check that not all the tx were mined and go to the next
		// check if the tx is in the pending pool
		// Retry if the tx has not appeared in the pool yet.
		// if the tx was mined successfully we can break the loop and proceed
		// update monitored tx changes into storage
		// if the tx was mined but failed, we continue to consider it was not mined
		// and store the failed receipt to be used to check if nonce needs to be reviewed
		hasFailedReceipts, allHistoryTxMined, receiptSuccessful := tm.checkTxHistory(ctx, mTx, mTxLog, dbTx)

		if receiptSuccessful {
			//mTxLog.Infof("tx %s was mined successfully", txHash.String())

			mTx.Status = ctmtypes.MonitoredTxStatusConfirmed

			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx when confirmed: %v", err)
			}
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

// returns hasFailedReceipts, allHistoryTxMined, receiptSuccessful
func (tm *MonitorTxs) checkTxHistory(ctx context.Context, mTx ctmtypes.MonitoredTx, mTxLog *log.Logger, dbTx pgx.Tx) (bool, bool, bool) {
	var receipt *types.Receipt
	hasFailedReceipts := false
	allHistoryTxMined := true
	receiptSuccessful := false
	mined := false
	var err error
	for txHash := range mTx.History {
		mTxLog.Infof("Checking if tx %s is mined", txHash.String())
		mined, receipt, err = tm.l2Node.CheckTxWasMined(ctx, txHash)
		if err != nil {
			mTxLog.Errorf("failed to check if tx %s was mined: %v", txHash.String(), err)
			continue
		}

		if !mined {
			_, _, err = tm.l2Node.TransactionByHash(ctx, txHash)
			if err != nil {
				mTxLog.Errorf("error getting txByHash %s. Error: %v", txHash.String(), err)

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

		if receipt.Status == types.ReceiptStatusSuccessful {
			mTxLog.Infof("tx %s was mined successfully", txHash.String())
			receiptSuccessful = true

			break
		}
		hasFailedReceipts = true
	}
	return hasFailedReceipts, allHistoryTxMined, receiptSuccessful
}

// ReviewMonitoredTx checks if tx needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (tm *MonitorTxs) ReviewMonitoredTx(ctx context.Context, mTx *ctmtypes.MonitoredTx, reviewNonce bool) error {
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
