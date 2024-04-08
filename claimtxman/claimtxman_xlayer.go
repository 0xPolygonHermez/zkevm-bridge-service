package claimtxman

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/pushtask"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils/gerror"
	"github.com/0xPolygonHermez/zkevm-node/pool"
	"github.com/0xPolygonHermez/zkevm-node/state/runtime"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
)

// StartXLayer will start the tx management, reading txs from storage,
// send then to the blockchain and keep monitoring them until they
// get mined
func (tm *ClaimTxManager) StartXLayer() {
	go tm.startMonitorTxs()
	for {
		select {
		case <-tm.ctx.Done():
			tm.isDone = true
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
					err := tm.updateDepositsStatusXLayer(ger)
					if err != nil {
						log.Errorf("failed to update deposits status: %v", err)
					}
				}()
			} else {
				log.Infof("Waiting for networkID %d to be synced before processing deposits", tm.l2NetworkID)
			}
		}
	}
}

func (tm *ClaimTxManager) startMonitorTxs() {
	ticker := time.NewTicker(tm.cfg.FrequencyToMonitorTxs.Duration)
	for range ticker.C {
		if tm.isDone {
			return
		}
		traceID := utils.GenerateTraceID()
		ctx := context.WithValue(tm.ctx, utils.CtxTraceID, traceID)
		logger := log.WithFields(utils.TraceID, traceID)
		logger.Infof("MonitorTxs begin %d", tm.l2NetworkID)
		err := tm.monitorTxsXLayer(ctx)
		if err != nil {
			logger.Errorf("failed to monitor txs: %v", err)
		}
		logger.Infof("MonitorTxs end %d", tm.l2NetworkID)
	}
}

func (tm *ClaimTxManager) updateDepositsStatusXLayer(ger *etherman.GlobalExitRoot) error {
	if tm.cfg.OptClaim {
		if ger.BlockID != 0 {
			tm.updateDepositsL2Mutex.Lock()
			defer tm.updateDepositsL2Mutex.Unlock()
			return tm.processDepositStatusL2(ger)
		} else {
			tm.updateDepositsL1Mutex.Lock()
			defer tm.updateDepositsL1Mutex.Unlock()
			return tm.processDepositStatusL1(ger)
		}
	}

	dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
	if err != nil {
		return err
	}
	err = tm.processDepositStatusXLayer(ger, dbTx)
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
	log.Debugf("updateDepositsStatus done")

	return nil
}

func (tm *ClaimTxManager) processDepositStatusL2(ger *etherman.GlobalExitRoot) error {
	dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
	if err != nil {
		return err
	}
	log.Infof("Rollup exitroot %v is updated", ger.ExitRoots[1])
	deposits, err := tm.storage.UpdateL2DepositsStatusXLayer(tm.ctx, ger.ExitRoots[1][:], tm.rollupID, tm.l2NetworkID, dbTx)
	if err != nil {
		log.Errorf("error getting and updating L2DepositsStatus. Error: %v", err)
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
	log.Debugf("begin send deposits for l1 ready_claim, blockId: %v, blockNumber: %v, deposit size: %v", ger.BlockID, ger.BlockNumber,
		len(deposits))
	for _, deposit := range deposits {
		// Notify FE that tx is pending auto claim
		go tm.pushTransactionUpdate(deposit, uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM))
	}
	return nil
}

func (tm *ClaimTxManager) getDeposits(ger *etherman.GlobalExitRoot) ([]*etherman.Deposit, error) {
	log.Infof("Mainnet exitroot %v is updated", ger.ExitRoots[0])
	deposits, err := tm.storage.GetL1Deposits(tm.ctx, ger.ExitRoots[0][:], nil)
	if err != nil {
		log.Errorf("error processing ger. Error: %v", err)
		return nil, err
	}
	return deposits, nil
}

func (tm *ClaimTxManager) processDepositStatusL1(newGer *etherman.GlobalExitRoot) error {
	deposits, err := tm.getDeposits(newGer)
	if err != nil {
		return err
	}

	log.Debug("createClaimTx deposits-num:", len(deposits))
	for _, deposit := range deposits {
		dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
		if err != nil {
			return err
		}
		err = tm.storage.UpdateL1DepositStatus(tm.ctx, deposit.DepositCount, dbTx)
		if err != nil {
			log.Errorf("error update deposit %d status. Error: %v", deposit.DepositCount, err)
			tm.rollbackStore(dbTx)
			return err
		}
		ignore := false
		if tm.l2NetworkID != deposit.DestinationNetwork {
			log.Infof("Ignoring deposit: %d: dest_net: %d, we are:%d", deposit.DepositCount, deposit.DestinationNetwork, tm.l2NetworkID)
			ignore = true
		} else {
			claimHash, err := tm.bridgeService.GetDepositStatus(tm.ctx, deposit.DepositCount, deposit.DestinationNetwork)
			if err != nil {
				log.Errorf("error getting deposit status for deposit %d. Error: %v", deposit.DepositCount, err)
				tm.rollbackStore(dbTx)
				return err
			}
			if len(claimHash) > 0 || (deposit.LeafType == LeafTypeMessage && !tm.isDepositMessageAllowed(deposit)) {
				log.Infof("Ignoring deposit: %d, leafType: %d, claimHash: %s", deposit.DepositCount, deposit.LeafType, claimHash)
				ignore = true
			}
		}

		if ignore {
			// todo: optimize it
			err = tm.storage.Commit(tm.ctx, dbTx)
			if err != nil {
				log.Errorf("AddClaimTx committing dbTx. Err: %v", err)
				rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
				if rollbackErr != nil {
					log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
					return rollbackErr
				}
				log.Fatalf("AddClaimTx committing dbTx, err: %s", err.Error())
				return err
			}
			continue
		}
		log.Infof("create the claim tx for the deposit %d", deposit.DepositCount)
		ger, proof, rollupProof, err := tm.bridgeService.GetClaimProof(deposit.DepositCount, deposit.NetworkID, dbTx)
		if err != nil {
			log.Errorf("error getting Claim Proof for deposit %d. Error: %v", deposit.DepositCount, err)
			tm.rollbackStore(dbTx)
			return err
		}
		log.Infof("get the claim proof for the deposit %d successfully", deposit.DepositCount)
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
			log.Errorf("error BuildSendClaimXLayer tx for deposit %d. Error: %v", deposit.DepositCount, err)
			tm.rollbackStore(dbTx)
			return err
		}
		if err = tm.addClaimTxXLayer(deposit.DepositCount, tm.auth.From, tx.To(), nil, tx.Data(), dbTx); err != nil {
			log.Errorf("error adding claim tx for deposit %d. Error: %v", deposit.DepositCount, err)
			tm.rollbackStore(dbTx)
			return err
		}

		// There can be cases that the deposit can be ready for claim (and even claimed) before it reached 64 block confirmations
		// (for example, in devnet where the block confirmations required is lower)
		// To prevent duplicated push in such cases (which can cause unexpected behavior), we need to remove the tx from
		// the block->tx mapping cache
		err = tm.redisStorage.DeleteBlockDeposit(tm.ctx, deposit)
		if err != nil {
			log.Errorf("failed to delete deposit %d from block num cache, error: %v", deposit.DepositCount, err)
			tm.rollbackStore(dbTx)
			return err
		}

		err = tm.storage.Commit(tm.ctx, dbTx)
		if err != nil {
			log.Errorf("AddClaimTx committing dbTx. Err: %v", err)
			rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
				return rollbackErr
			}
			log.Fatalf("AddClaimTx committing dbTx, err: %s", err.Error())
			return err
		}
		log.Infof("add claim tx for the deposit %d blockID %d successfully", deposit.DepositCount, deposit.BlockID)

		// Notify FE that tx is pending auto claim
		go tm.pushTransactionUpdate(deposit, uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM))
	}
	return nil
}

func (tm *ClaimTxManager) rollbackStore(dbTx pgx.Tx) {
	rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
	if rollbackErr != nil {
		log.Errorf("claimtxman error rolling back state. RollbackErr: %v", rollbackErr)
	}
}

func (tm *ClaimTxManager) processDepositStatusXLayer(ger *etherman.GlobalExitRoot, dbTx pgx.Tx) error {
	if ger.BlockID != 0 { // L2 exit root is updated
		log.Infof("Rollup exitroot %v is updated", ger.ExitRoots[1])
		deposits, err := tm.storage.UpdateL2DepositsStatusXLayer(tm.ctx, ger.ExitRoots[1][:], tm.rollupID, tm.l2NetworkID, dbTx)
		if err != nil {
			log.Errorf("error getting and updating L2DepositsStatus. Error: %v", err)
			return err
		}
		log.Debugf("begin send deposits for l1 ready_claim, blockId: %v, blockNumber: %v, deposit size: %v", ger.BlockID, ger.BlockNumber,
			len(deposits))
		for _, deposit := range deposits {
			// Notify FE that tx is pending auto claim
			go tm.pushTransactionUpdate(deposit, uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM))
		}
	} else { // L1 exit root is updated in the trusted state
		log.Infof("Mainnet exitroot %v is updated", ger.ExitRoots[0])
		deposits, err := tm.storage.UpdateL1DepositsStatusXLayer(tm.ctx, ger.ExitRoots[0][:], dbTx)
		if err != nil {
			log.Errorf("error getting and updating L1DepositsStatus. Error: %v", err)
			return err
		}
		log.Debugf("Mainnet deposits count %d", len(deposits))
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
			if len(claimHash) > 0 || (deposit.LeafType == LeafTypeMessage && !tm.isDepositMessageAllowed(deposit)) {
				log.Infof("Ignoring deposit: %d, leafType: %d, claimHash: %s, deposit.OriginalAddress: %s", deposit.DepositCount, deposit.LeafType, claimHash, deposit.OriginalAddress.String())
				continue
			}
			log.Infof("create the claim tx for the deposit %d", deposit.DepositCount)
			ger, proof, rollupProof, err := tm.bridgeService.GetClaimProof(deposit.DepositCount, deposit.NetworkID, dbTx)
			if err != nil {
				log.Errorf("error getting Claim Proof for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			log.Debugf("get claim proof done for the deposit %d", deposit.DepositCount)
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
				log.Errorf("error BuildSendClaimXLayer tx for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			log.Debugf("claimTx for deposit %d build successfully %d", deposit.DepositCount)
			if err = tm.addClaimTxXLayer(deposit.DepositCount, tm.auth.From, tx.To(), nil, tx.Data(), dbTx); err != nil {
				log.Errorf("error adding claim tx for deposit %d. Error: %v", deposit.DepositCount, err)
				return err
			}
			log.Debugf("claimTx for deposit %d save successfully %d", deposit.DepositCount)

			// There can be cases that the deposit can be ready for claim (and even claimed) before it reached 64 block confirmations
			// (for example, in devnet where the block confirmations required is lower)
			// To prevent duplicated push in such cases (which can cause unexpected behavior), we need to remove the tx from
			// the block->tx mapping cache
			err = tm.redisStorage.DeleteBlockDeposit(tm.ctx, deposit)
			if err != nil {
				log.Errorf("failed to delete deposit %d from block num cache, error: %v", deposit.DepositCount, err)
				return err
			}

			// Notify FE that tx is pending auto claim
			go tm.pushTransactionUpdate(deposit, uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM))
		}
	}
	return nil
}

func (tm *ClaimTxManager) addClaimTxXLayer(depositCount uint, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
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

	// create monitored tx
	mTx := ctmtypes.MonitoredTx{
		DepositID: depositCount, From: from, To: to,
		Value: value, Data: data,
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

// monitorTxsXLayer process all pending monitored tx
func (tm *ClaimTxManager) monitorTxsXLayer(ctx context.Context) error {
	mLog := log.WithFields(utils.TraceID, ctx.Value(utils.CtxTraceID))

	dbTx, err := tm.storage.BeginDBTransaction(ctx)
	if err != nil {
		return err
	}
	mLog.Infof("monitorTxs begin")

	statusesFilter := []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatusCreated}
	mTxs, err := tm.storage.GetClaimTxsByStatus(ctx, statusesFilter, dbTx)
	if err != nil {
		mLog.Errorf("failed to get created monitored txs: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			mLog.Errorf("claimtxman error rolling back state. RollbackErr: %s, err: %v", rollbackErr.Error(), err)
			return rollbackErr
		}
		return fmt.Errorf("failed to get created monitored txs: %v", err)
	}
	mLog.Infof("found %v monitored tx to process", len(mTxs))

	isResetNonce := false // it will reset the nonce in one cycle
	for _, mTx := range mTxs {
		mTx := mTx // force variable shadowing to avoid pointer conflicts
		mTxLog := mLog.WithFields("monitoredTx", mTx.DepositID)
		mTxLog.Infof("processing tx with nonce %d", mTx.Nonce)
		// Check the claim table to see whether the transaction has already been claimed by some other methods
		_, err = tm.storage.GetClaim(ctx, mTx.DepositID, tm.l2NetworkID, dbTx)
		if err != nil && err != gerror.ErrStorageNotFound {
			mTxLog.Errorf("failed to get claim tx: %v", err)
			return err
		}
		if err == nil {
			mTxLog.Infof("Tx has already been claimed")
			mTx.Status = ctmtypes.MonitoredTxStatusConfirmed
			// Update monitored txs status to confirmed
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update tx status to confirmed: %v", err)
			}
			continue
		}

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
				mTxLog.Infof("tx: %s not mined yet", txHash.String())

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

			// Notify FE that tx is pending user claim
			go func() {
				// Retrieve L1 transaction info
				deposit, err := tm.storage.GetDeposit(ctx, mTx.DepositID, 0, nil)
				if err != nil {
					mTxLog.Errorf("push message: GetDeposit error: %v", err)
					return
				}
				tm.pushTransactionUpdate(deposit, uint32(pb.TransactionStatus_TX_PENDING_USER_CLAIM))
			}()
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
				err := tm.ReviewMonitoredTxXLayer(ctx, &mTx)
				if err != nil {
					mTxLog.Errorf("failed to review monitored tx: %v", err)
					continue
				}
			}

			// GasPrice is set here to use always the proper and most accurate value right before sending it to L2
			gasPrice := big.NewInt(0)
			if !tm.cfg.FreeGas {
				gasPrice, err = tm.l2Node.SuggestGasPrice(ctx)
				if err != nil {
					mTxLog.Errorf("failed to get suggested gasPrice. Error: %v", err)
					continue
				}
			}

			//Multiply gasPrice by 10 to increase the efficiency of the tx in the sequence
			mTx.GasPrice = big.NewInt(0).Mul(gasPrice, big.NewInt(10)) //nolint:gomnd
			mTxLog.Infof("Using gasPrice: %s. The gasPrice suggested by the network is %s", mTx.GasPrice.String(), gasPrice.String())

			// Calculate nonce before signing
			err = tm.setTxNonce(&mTx)
			if err != nil {
				mTxLog.Errorf("failed to set tx nonce: %v", err)
				continue
			}

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
			mTxLog.Debugf("signed tx %v created using gasPrice: %s, nonce: %v", signedTx.Hash().String(), signedTx.GasPrice().String(), signedTx.Nonce())

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
					if err.Error() == pool.ErrNonceTooLow.Error() {
						mTxLog.Infof("nonce error detected, Nonce used: %d", signedTx.Nonce())
						if !isResetNonce {
							isResetNonce = true
							tm.nonceCache.Remove(mTx.From.Hex())
							mTxLog.Infof("nonce cache cleared for address %v", mTx.From.Hex())
						}
					}
					if err.Error() == pool.ErrNonceTooHigh.Error() {
						mTxLog.Infof("nonce error detected, Nonce used: %d", signedTx.Nonce())
						if !isResetNonce {
							isResetNonce = true
							tm.nonceCache.Remove(mTx.From.Hex())
							mTxLog.Infof("nonce cache cleared for address %v", mTx.From.Hex())
						}
					}
					mTx.RemoveHistory(signedTx)
					// we should rebuild the monitored tx to fix the nonce
					err := tm.ReviewMonitoredTxXLayer(ctx, &mTx)
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
	mLog.Infof("monitorTxs end")

	err = tm.storage.Commit(tm.ctx, dbTx)
	if err != nil {
		mLog.Errorf("UpdateClaimTx committing dbTx, err: %v", err)
		rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
		if rollbackErr != nil {
			mLog.Errorf("claimtxman error rolling back state. RollbackErr: %s, err: %v", rollbackErr.Error(), err)
			return rollbackErr
		}
		return err
	}

	mLog.Infof("monitorTxs committed")
	return nil
}

// setTxNonce get the next nonce from the nonce cache and set it to the tx
func (tm *ClaimTxManager) setTxNonce(mTx *ctmtypes.MonitoredTx) error {
	nonce, err := tm.getNextNonce(mTx.From)
	if err != nil {
		return errors.Wrap(err, "getNextNonce err")
	}
	mTx.Nonce = nonce
	return nil
}

// Push message to FE to notify about tx status change
func (tm *ClaimTxManager) pushTransactionUpdate(deposit *etherman.Deposit, status uint32) {
	if tm.messagePushProducer == nil {
		log.Errorf("kafka push producer is nil, so can't push tx status change msg!")
		return
	}
	if deposit.LeafType != uint8(utils.LeafTypeAsset) && !tm.isDepositMessageAllowed(deposit) {
		log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
		return
	}
	estimateTime := uint64(0)
	if deposit.NetworkID != 0 {
		estimateTime = pushtask.GetAvgVerifyDuration(tm.ctx, tm.redisStorage)
	}
	err := tm.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
		FromChain:    uint32(deposit.NetworkID),
		ToChain:      uint32(deposit.DestinationNetwork),
		TxHash:       deposit.TxHash.String(),
		Index:        uint64(deposit.DepositCount),
		Status:       status,
		DestAddr:     deposit.DestinationAddress.Hex(),
		EstimateTime: uint32(estimateTime),
		GlobalIndex:  etherman.GenerateGlobalIndex(false, tm.rollupID-1, deposit.DepositCount).String(),
	})
	if err != nil {
		log.Errorf("PushTransactionUpdate error: %v", err)
	}
}

// ReviewMonitoredTxXLayer checks if tx needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (tm *ClaimTxManager) ReviewMonitoredTxXLayer(ctx context.Context, mTx *ctmtypes.MonitoredTx) error {
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

	return nil
}
