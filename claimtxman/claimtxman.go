package claimtxman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
)

const (
	failureIntervalInSeconds = 5
	maxHistorySize           = 10
)

var (
	// ErrNotFound when the object is not found
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists when the object already exists
	ErrAlreadyExists = errors.New("already exists")

	// ErrExecutionReverted returned when trying to get the revert message
	// but the call fails without revealing the revert reason
	ErrExecutionReverted = errors.New("execution reverted")
)

// ClaimTxManager is the claim transaction manager
type ClaimTxManager struct {
	ctx    context.Context
	cancel context.CancelFunc

	// client is the ethereum client
	l2Node          *utils.Client
	bridgeService   pb.BridgeServiceServer
	cfg             Config
	chExitRootEvent chan *etherman.GlobalExitRoot
	storage         storageInterface
	auth            *bind.TransactOpts
}

// NewClaimTxManager creates a new claim transaction manager
func NewClaimTxManager(cfg Config, chExitRootEvent chan *etherman.GlobalExitRoot, l2NodeURL, l2BridgeAddr string, bridgeService pb.BridgeServiceServer, storage storageInterface) (*ClaimTxManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client, err := utils.NewClient(ctx, l2NodeURL, l2BridgeAddr)
	if err != nil {
		return nil, err
	}
	auth, err := client.GetSignerFromKeystore(ctx, cfg.PrivateKey)
	return &ClaimTxManager{
		ctx:             ctx,
		cancel:          cancel,
		l2Node:          client,
		bridgeService:   bridgeService,
		cfg:             cfg,
		chExitRootEvent: chExitRootEvent,
		storage:         storage,
		auth:            auth,
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
			tm.updateDepositsStatus(ger)
		case <-time.After(tm.cfg.FrequencyToMonitorTxs.Duration):
			err := tm.monitorTxs(context.Background())
			if err != nil {
				log.Errorf("failed to monitor txs: %v", err)
			}
		}
	}
}

func (tm *ClaimTxManager) updateDepositsStatus(ger *etherman.GlobalExitRoot) error {
	if ger.BlockID != 0 { // L2 exit root is updated
		if err := tm.storage.UpdateL2DepositsStatus(tm.ctx, ger.ExitRoots[1][:]); err != nil {
			return err
		}
	} else { // L1 exit root is updated in virtual state
		deposits, err := tm.storage.UpdateL1DepositsStatus(tm.ctx, ger.ExitRoots[0][:])
		if err != nil {
			return err
		}
		dbTx, err := tm.storage.BeginDBTransaction(tm.ctx)
		if err != nil {
			return err
		}
		for _, deposit := range deposits {
			res, err := tm.bridgeService.GetProof(tm.ctx, &pb.GetProofRequest{
				NetId:      deposit.NetworkId,
				DepositCnt: deposit.DepositCnt,
			})
			if err != nil {
				return err
			}
			proves := [32][bridgectrl.KeyLen]byte{}
			for i, p := range res.Proof.MerkleProof {
				var proof [bridgectrl.KeyLen]byte
				copy(proof[:], common.FromHex(p))
				proves[i] = proof
			}
			to, data, err := tm.l2Node.BuildSendClaim(tm.ctx, deposit, proves,
				&etherman.GlobalExitRoot{
					ExitRoots: []common.Hash{
						common.HexToHash(res.Proof.MainExitRoot),
						common.HexToHash(res.Proof.RollupExitRoot),
					}},
				tm.auth)
			if err != nil {
				return err
			}
			tm.addClaimTx(strconv.FormatUint(deposit.DepositCnt, 10), tm.auth.From, to, nil, data, dbTx)
		}
		err = tm.storage.Commit(tm.ctx, dbTx)
		if err != nil {
			rollbackErr := tm.storage.Rollback(tm.ctx, dbTx)
			if rollbackErr != nil {
				log.Fatalf("claimtxman error rolling back state. RollbackErr: %s, err: %s", rollbackErr.Error(), err.Error())
			}
			log.Fatalf("AddClaimTx committing dbTx, err: %s", err.Error())
		}
	}
	return nil
}

func (tm *ClaimTxManager) addClaimTx(id string, from common.Address, to *common.Address, value *big.Int, data []byte, dbTx pgx.Tx) error {
	// get next nonce
	nonce, err := tm.l2Node.NonceAt(tm.ctx, from, nil)
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
	mTx := MonitoredTx{
		Id: id, From: from, To: to,
		Nonce: nonce, Value: value, Data: data,
		Gas: gas, GasPrice: big.NewInt(0),
		Status: MonitoredTxStatusCreated,
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

	statusesFilter := []MonitoredTxStatus{MonitoredTxStatusCreated, MonitoredTxStatusSent}
	mTxs, err := tm.storage.GetByStatus(ctx, nil, statusesFilter, dbTx)
	if err != nil {
		return fmt.Errorf("failed to get created monitored txs: %v", err)
	}

	log.Infof("found %v monitored tx to process", len(mTxs))

	for _, mTx := range mTxs {
		mTx := mTx // force variable shadowing to avoid pointer conflicts
		mTxLog := log.WithFields("monitoredTx", mTx.Id)
		mTxLog.Info("processing")

		// check if any of the txs in the history was mined
		mined := false
		var receipt *types.Receipt
		hasFailedReceipts := false
		allHistoryTxMined := true
		for txHash := range mTx.History {
			mined, receipt, err = tm.l2Node.CheckTxWasMined(ctx, txHash)
			if err != nil {
				mTxLog.Errorf("failed to check if tx %v was mined: %v", txHash.String(), err)
				continue
			}

			// if the tx is not mined yet, check that not all the tx were mined and go to the next
			if !mined {
				allHistoryTxMined = false
				continue
			}

			// if the tx was mined successfully we can break the loop and proceed
			if receipt.Status == types.ReceiptStatusSuccessful {
				break
			}

			// if the tx was mined but failed, we continue to consider it was not mined
			// and store the failed receipt to be used to check if nonce needs to be reviewed
			mined = false
			hasFailedReceipts = true
		}

		// we need to check if we need to review the nonce carefully, to avoid sending
		// duplicated data to the block chain.
		//
		// if we have failed receipts, this means at least one of the generated txs was mined
		// so maybe the current nonce was already consumed, then we need to check if there are
		// tx that were not mined yet, if so, we just need to wait, because maybe one of them
		// will get mined successfully
		//
		// in case of all tx were mined and none of them were mined successfully, we need to
		// review the nonce
		if hasFailedReceipts && allHistoryTxMined {
			mTxLog.Infof("nonce needs to be updated")
			err := tm.ReviewMonitoredTxNonce(ctx, &mTx)
			if err != nil {
				mTxLog.Errorf("failed to review monitored tx nonce: %v", err)
				continue
			}
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx nonce change: %v", err)
				continue
			}
		}

		// if the history size reaches the max history size, this means something is really wrong with
		// this Tx and we are not able to identify automatically, so we mark this as failed to let the
		// caller know something is not right and needs to be review and to avoid to monitor this
		// tx infinitely
		if len(mTx.History) == maxHistorySize {
			mTx.Status = MonitoredTxStatusFailed
			mTxLog.Infof("marked as failed because reached the history size limit: %v", err)
			// update monitored tx changes into storage
			err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to update monitored tx when max history size limit reached: %v", err)
				continue
			}
		}

		var signedTx *types.Transaction
		if !mined {
			// review tx and increase gas and gas price if needed
			if mTx.Status == MonitoredTxStatusSent { // TODO check the interval of UpdatedAt
				err := tm.ReviewMonitoredTx(ctx, &mTx)
				if err != nil {
					mTxLog.Errorf("failed to review monitored tx: %v", err)
					continue
				}
				err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
				if err != nil {
					mTxLog.Errorf("failed to update monitored tx review change: %v", err)
					continue
				}
			}

			// rebuild transaction
			tx := mTx.Tx()
			mTxLog.Debugf("unsigned tx %v created", tx.Hash().String(), mTx.Id)

			// sign tx
			signedTx, err = tm.auth.Signer(mTx.From, tx)
			if err != nil {
				mTxLog.Errorf("failed to sign tx %v created from monitored tx %v: %v", tx.Hash().String(), mTx.Id, err)
				continue
			}
			mTxLog.Debugf("signed tx %v created", signedTx.Hash().String())

			// add tx to monitored tx history
			err = mTx.AddHistory(signedTx)
			if errors.Is(err, ErrAlreadyExists) {
				mTxLog.Infof("signed tx already existed in the history")
			} else if err != nil {
				mTxLog.Errorf("failed to add signed tx to monitored tx %v history: %v", mTx.Id, err)
				continue
			} else {
				// update monitored tx changes into storage
				err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
				if err != nil {
					mTxLog.Errorf("failed to update monitored tx: %v", err)
					continue
				}
				mTxLog.Debugf("signed tx added to the monitored tx history")
			}

			// check if the tx is already in the network, if not, send it
			_, _, err = tm.l2Node.TransactionByHash(ctx, signedTx.Hash())
			// if not found, send it tx to the network
			if errors.Is(err, ethereum.NotFound) {
				mTxLog.Debugf("signed tx not found in the network")
				err := tm.l2Node.SendTransaction(ctx, signedTx)
				if err != nil {
					mTxLog.Errorf("failed to send tx %v to network: %v", signedTx.Hash().String(), err)
					continue
				}
				mTxLog.Infof("signed tx sent to the network: %v", signedTx.Hash().String())
				if mTx.Status == MonitoredTxStatusCreated {
					// update tx status to sent
					mTx.Status = MonitoredTxStatusSent
					mTxLog.Debugf("status changed to %v", string(mTx.Status))
					// update monitored tx changes into storage
					err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
					if err != nil {
						mTxLog.Errorf("failed to update monitored tx changes: %v", err)
						continue
					}
				}
			} else {
				mTxLog.Infof("signed tx already found in the network")
			}
		}

		// if mined, check receipt and mark as Failed or Confirmed
		if receipt.Status == types.ReceiptStatusSuccessful {
			block, err := tm.l2Node.BlockByNumber(ctx, receipt.BlockNumber)
			if err != nil {
				mTxLog.Errorf("failed to get receipt block: %v", err)
				continue
			}
			mTx.BlockID, err = tm.storage.AddBlock(ctx, &etherman.Block{
				BlockNumber: block.Number().Uint64(),
				BlockHash:   block.Hash(),
				ParentHash:  block.ParentHash(),
				ReceivedAt:  block.ReceivedAt,
			}, dbTx)
			if err != nil {
				mTxLog.Errorf("failed to add receipt block: %v", err)
				continue
			}
		} else {
			mTxLog.Info("failed")
			// otherwise we understand this monitored tx has failed
			mTx.Status = MonitoredTxStatusFailed
		}

		// update monitored tx changes into storage
		err = tm.storage.UpdateClaimTx(ctx, mTx, dbTx)
		if err != nil {
			mTxLog.Errorf("failed to update monitored tx: %v", err)
			continue
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

// ReviewMonitoredTx checks if some field needs to be updated
// accordingly to the current information stored and the current
// state of the blockchain
func (tm *ClaimTxManager) ReviewMonitoredTx(ctx context.Context, mTx *MonitoredTx) error {
	mTxLog := log.WithFields("monitoredTx", mTx.Id)
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
	return nil
}

// ReviewMonitoredTxNonce checks if the nonce needs to be updated accordingly to
// the current nonce of the sender account.
//
// IMPORTANT: Nonce is reviewed apart from the other fields because it is a very
// sensible information and can make duplicated data to be sent to the blockchain,
// causing possible side effects and wasting resources on taxes.
func (tm *ClaimTxManager) ReviewMonitoredTxNonce(ctx context.Context, mTx *MonitoredTx) error {
	mTxLog := log.WithFields("monitoredTx", mTx.Id)
	mTxLog.Debug("reviewing nonce")
	nonce, err := tm.l2Node.NonceAt(ctx, mTx.From, nil)
	if err != nil {
		err := fmt.Errorf("failed to estimate gas: %w", err)
		mTxLog.Errorf(err.Error())
		return err
	}

	if nonce > mTx.Nonce {
		mTxLog.Infof("monitored tx nonce updated from %v to %v", mTx.Nonce, nonce)
		mTx.Nonce = nonce
	}

	return nil
}
