package claimtxman

import (
	"context"
	"fmt"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4"
)

const (
	MinTxPerGroup = 2
)

type StorageCompressedInterface interface {
	GetClaimTxsByStatus(ctx context.Context, statuses []ctmtypes.MonitoredTxStatus, dbTx pgx.Tx) ([]ctmtypes.MonitoredTx, error)
	GetMonitoredTxsGroups(ctx context.Context, groupIds []uint64, dbTx pgx.Tx) (map[uint64]ctmtypes.MonitoredTxGroupDBEntry, error)

	AddMonitoredTxsGroup(ctx context.Context, mTxGroup *ctmtypes.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error
	UpdateClaimTx(ctx context.Context, mTx ctmtypes.MonitoredTx, dbTx pgx.Tx) error
	GetLatestMonitoredTxGroupID(ctx context.Context, dbTx pgx.Tx) (uint64, error)
	UpdateMonitoredTxsGroup(ctx context.Context, mTxGroup *ctmtypes.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error
	// atomic
	Rollback(ctx context.Context, dbTx pgx.Tx) error
	BeginDBTransaction(ctx context.Context) (pgx.Tx, error)
	Commit(ctx context.Context, dbTx pgx.Tx) error
}

type EthermanI interface {
	CompressClaimCall(mainnetExitRoot, rollupExitRoot common.Hash, claimData []claimcompressor.ClaimCompressorCompressClaimCallData) ([]byte, error)
	SendCompressedClaims(auth *bind.TransactOpts, compressedTxData []byte) (*types.Transaction, error)
}
type MonitorCompressedTxs struct {
	storage StorageCompressedInterface
	ctx     context.Context
	// client is the ethereum client
	l2Node                *utils.Client
	cfg                   Config
	nonceCache            *NonceCache
	auth                  *bind.TransactOpts
	etherMan              EthermanI
	compressClaimComposer *ComposeCompressClaim
	timeProvider          utils.TimeProvider
	triggerGroups         *GroupsTrigger
	gasOffset             uint64
}

func NewMonitorCompressedTxs(ctx context.Context,
	storage StorageCompressedInterface,
	l2Node *utils.Client,
	cfg Config,
	nonceCache *NonceCache,
	auth *bind.TransactOpts,
	etherMan EthermanI,
	timeProvider utils.TimeProvider,
	gasOffset uint64) *MonitorCompressedTxs {
	composer, err := NewComposeCompressClaim()
	if err != nil {
		log.Fatal("failed to create ComposeCompressClaim: %v", err)
	}
	return &MonitorCompressedTxs{
		storage:               storage,
		ctx:                   ctx,
		l2Node:                l2Node,
		cfg:                   cfg,
		nonceCache:            nonceCache,
		auth:                  auth,
		etherMan:              etherMan,
		compressClaimComposer: composer,
		timeProvider:          timeProvider,
		triggerGroups:         NewGroupsTrigger(cfg.GroupingClaims),
		gasOffset:             gasOffset,
	}
}

func getGroupsIds(txs []ctmtypes.MonitoredTx) []uint64 {
	tmp := make(map[uint64]struct{})
	for _, tx := range txs {
		if tx.GroupID != nil {
			tmp[*tx.GroupID] = struct{}{}
		}
	}
	keys := make([]uint64, len(tmp))
	i := 0
	for k := range tmp {
		keys[i] = k
		i++
	}
	return keys
}

func (tm *MonitorCompressedTxs) getPendingTxs(ctx context.Context, dbTx pgx.Tx) (PendingTxs, error) {
	statusesFilter := []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatusCreated,
		ctmtypes.MonitoredTxStatusCompressing,
		ctmtypes.MonitoredTxStatusClaiming}
	mTxs, err := tm.storage.GetClaimTxsByStatus(ctx, statusesFilter, dbTx)
	if err != nil {
		return PendingTxs{}, fmt.Errorf("failed to get get monitored txs: %v", err)
	}
	groupIds := getGroupsIds(mTxs)
	groups, err := tm.storage.GetMonitoredTxsGroups(ctx, groupIds, dbTx)
	if err != nil {
		return PendingTxs{}, fmt.Errorf("failed to get get monitoredGroups txs: %v", err)
	}
	lastGroupID, err := tm.storage.GetLatestMonitoredTxGroupID(ctx, dbTx)
	if err != nil {
		return PendingTxs{}, fmt.Errorf("failed to get latest group id: %v", err)
	}
	return NewPendingTxs(mTxs, groups, lastGroupID)
}

// monitorTxs process all pending monitored tx
func (tm *MonitorCompressedTxs) MonitorTxs(ctx context.Context) error {
	dbTx, err := tm.storage.BeginDBTransaction(ctx)
	if err != nil {
		return err
	}
	err = tm.internalMonitorTxs(ctx, dbTx)
	if err != nil {
		_ = tm.storage.Rollback(ctx, dbTx)
		return err
	}
	return tm.storage.Commit(ctx, dbTx)
}

// monitorTxs process all pending monitored tx
func (tm *MonitorCompressedTxs) internalMonitorTxs(ctx context.Context, dbTx pgx.Tx) error {
	pendingTx, err := tm.getPendingTxs(ctx, dbTx)
	if err != nil {
		return err
	}
	if pendingTx.IsEmpty() {
		log.Debug("no pending txs to process, skipping...")
		return nil
	}
	// TODO: Change for deepcopy but is launching a panic
	// tmpCopy, err := deepcopy.Anything(pendingTx)
	// initialStatus := tmpCopy.(*PendingTxs)
	// if err != nil {
	// 	return fmt.Errorf("failed to deepcopy pendingTx: %v", err)
	// }
	initialStatus, err := tm.getPendingTxs(ctx, dbTx)
	if err != nil {
		return err
	}

	err = tm.Process(ctx, &pendingTx)
	if err != nil {
		return err
	}
	storeUpdate, err := GenerateStoreUpdate(initialStatus, pendingTx, tm.timeProvider)
	if err != nil {
		return err
	}

	err = storeUpdate.Execute(ctx, tm.storage, dbTx)
	if err != nil {
		return err
	}

	return nil
}

func (tm *MonitorCompressedTxs) Process(ctx context.Context, pendingTxs *PendingTxs) error {
	// SendCompressClaims for each group
	err := tm.SendClaims(pendingTxs, false)
	if err != nil {
		return err
	}
	err = tm.CheckReceipts(ctx, pendingTxs)
	if err != nil {
		return err
	}
	// Outdate claimsCall that have reach outdate duration
	err = tm.OutdateClaims(pendingTxs)
	if err != nil {
		return err
	}
	// Try to create new groups to claims
	err = tm.createNewGroups(pendingTxs)
	if err != nil {
		return err
	}
	// Send claim txs for each group
	return nil
}

// OnFinishClaimGroupTx is called when a claim tx is mined successful
func (tm *MonitorCompressedTxs) OnFinishClaimGroupTxSuccessful(group *ctmtypes.MonitoredTxGroup, txIndex int) {
	txHash := group.DbEntry.ClaimTxHistory.TxHashes[txIndex].TxHash
	msg := fmt.Sprintf("group_id:%d , tx %s was mined successfully", group.DbEntry.GroupID, txHash.String())
	log.Infof(msg)
	group.DbEntry.LastLog = msg
	group.DbEntry.ClaimTxHistory.TxHashes[txIndex].ReceiptSuccessful()
	group.DbEntry.Status = ctmtypes.MonitoredTxGroupStatusConfirmed
	for i := 0; i < len(group.Txs); i++ {
		log.Infof("group_id:%d , confirmed deposit_id:%d", group.DbEntry.GroupID, group.Txs[i].DepositID)
		group.Txs[i].Status = ctmtypes.MonitoredTxStatusConfirmed
	}
}

func (tm *MonitorCompressedTxs) OnFailGroup(group *ctmtypes.MonitoredTxGroup) {
	group.DbEntry.Status = ctmtypes.MonitoredTxGroupStatussFailed
	group.DbEntry.LastLog = fmt.Sprintf("group_id:%d , reached maximum retries", group.DbEntry.GroupID)
	for i := 0; i < len(group.Txs); i++ {
		log.Infof("group_id:%d , failed deposit_id:%d", group.DbEntry.GroupID, group.Txs[i].DepositID)
		group.Txs[i].Status = ctmtypes.MonitoredTxStatusFailed
	}
}

func (tm *MonitorCompressedTxs) OnFinishClaimGroupTxFailed(group *ctmtypes.MonitoredTxGroup, txIndex int, msg string) {
	txHash := group.DbEntry.ClaimTxHistory.TxHashes[txIndex].TxHash
	msg2 := fmt.Sprintf("tx %s. %s", txHash.String(), msg)
	log.Warn(msg2)
	group.DbEntry.LastLog = msg2
	group.DbEntry.ClaimTxHistory.TxHashes[txIndex].ReceiptFailed()
}

func (tm *MonitorCompressedTxs) CheckReceipts(ctx context.Context, pendingTx *PendingTxs) error {
	for idx := range pendingTx.GroupTx {
		group := pendingTx.GroupTx[idx]
		if group.DbEntry.Status == ctmtypes.MonitoredTxGroupStatusClaiming {
			for txIndex, tx := range group.DbEntry.ClaimTxHistory.TxHashes {
				if tx.IsPending() {
					mined, receipt, err := tm.l2Node.CheckTxWasMined(ctx, tx.TxHash)
					if err != nil {
						msg := fmt.Sprintf("failed to check if tx[%d] %s was mined: %v", txIndex, tx.TxHash.String(), err)
						log.Debug(msg)
						group.DbEntry.LastLog = msg
						continue
					}
					if !mined {
						msg := fmt.Sprintf("tx %s not mined yet", tx.TxHash.String())
						log.Debug(msg)
						group.DbEntry.LastLog = msg
						continue
					}
					log.Infof("tx_id:%d tx_hash:%s mined:%v receipt_status:%v len_logs:%d", txIndex, tx.TxHash.String(), mined, receipt.Status, len(receipt.Logs))

					if receipt.Status == types.ReceiptStatusSuccessful && len(receipt.Logs) > 0 {
						tm.OnFinishClaimGroupTxSuccessful(group, txIndex)
						break
					} else {
						tm.OnFinishClaimGroupTxFailed(group, txIndex, "receipt status failed or no logs")
					}
				}
			}
		}
	}
	return nil
}

func (tm *MonitorCompressedTxs) OutdateClaims(pendingTx *PendingTxs) error {
	for idx := range pendingTx.GroupTx {
		group := pendingTx.GroupTx[idx]
		if group.DbEntry.Status == ctmtypes.MonitoredTxGroupStatusClaiming {
			for txIndex, txHash := range group.DbEntry.ClaimTxHistory.TxHashes {
				if txHash.IsPending() {
					now := tm.timeProvider.Now()
					if txHash.IsExhaustedTimeWaitingForReceipt(now, tm.cfg.GroupingClaims.RetryTimeout.Duration) {
						tm.OnFinishClaimGroupTxFailed(group, txIndex, "reached maximum wait time")
					}
				}
			}
		}
		if group.DbEntry.NumRetries >= int32(tm.cfg.GroupingClaims.MaxRetries) {
			log.Infof("group_id:%d reached maximum retries (%d>=%d)", group.DbEntry.GroupID,
				group.DbEntry.NumRetries, tm.cfg.GroupingClaims.MaxRetries)
			tm.OnFailGroup(group)
		}
	}
	return nil
}

func (tm *MonitorCompressedTxs) CanSendNewClaimCall(group *ctmtypes.MonitoredTxGroup) bool {
	if group.DbEntry.Status == ctmtypes.MonitoredTxGroupStatusCreated {
		return true
	}
	if group.DbEntry.ClaimTxHistory == nil {
		return true
	}
	if len(group.DbEntry.ClaimTxHistory.TxHashes) == 0 {
		return true
	}
	// Check history, if there are any txs pending to be mined we can't send a new one
	for _, txHash := range group.DbEntry.ClaimTxHistory.TxHashes {
		if txHash.IsPending() && !txHash.IsOutdated() {
			return false
		}
	}
	// If all txs have fail: could I send a new one? (check retryInterval and maxRetries)
	if group.DbEntry.NumRetries >= int32(tm.cfg.GroupingClaims.MaxRetries) {
		return false
	}
	// If all txs have fail: could I send a new one? (check retryInterval and maxRetries)
	moreRecentTx := group.DbEntry.ClaimTxHistory.GetMoreRecentTx()
	if moreRecentTx == nil {
		return true
	}
	elapsed := tm.timeProvider.Now().Sub(moreRecentTx.CreatedAt)
	return elapsed >= tm.cfg.GroupingClaims.RetryInterval.Duration
}

func (tm *MonitorCompressedTxs) SendClaims(pendingTx *PendingTxs, onlyFirstOne bool) error {
	for _, group := range pendingTx.GroupTx {
		if group.DbEntry.CompressedTxData == nil {
			log.Warnf("group %d has no compressed data", group.DbEntry.GroupID)
			continue
		}
		if !tm.CanSendNewClaimCall(group) {
			continue
		}
		if onlyFirstOne && !group.DbEntry.IsClaimTxHistoryEmpty() {
			continue
		}

		// Estimating Gas
		auth := *tm.auth
		auth.NoSend = true
		estimatedTx, err := tm.etherMan.SendCompressedClaims(&auth, group.DbEntry.CompressedTxData)
		if err != nil {
			msg := fmt.Sprintf("failed to call SMC SendCompressedClaims for group %d: %v", group.DbEntry.GroupID, err)
			log.Warn(msg)
			group.DbEntry.LastLog = msg
			continue
		}
		auth.NoSend = false
		log.Debug("estimatedGAS: ", estimatedTx.Gas())
		auth.GasLimit = estimatedTx.Gas() + tm.gasOffset
		log.Debug("New GAS: ", auth.GasLimit)
		// Send claim tx
		tx, err := tm.etherMan.SendCompressedClaims(&auth, group.DbEntry.CompressedTxData)
		if err != nil {
			msg := fmt.Sprintf("failed to call SMC SendCompressedClaims for group %d: %v", group.DbEntry.GroupID, err)
			log.Warn(msg)
			group.DbEntry.LastLog = msg
			continue
		}
		log.Debug("Gas used: ", tx.Gas())
		log.Infof("Send claim tx try: %d for group_id:%d  deposits_id:%s txHash:%s", group.DbEntry.NumRetries, group.DbEntry.GroupID, group.GetTxsDepositIDString(), tx.Hash().String())
		group.DbEntry.Status = ctmtypes.MonitoredTxGroupStatusClaiming
		group.DbEntry.AddPendingTx(tx.Hash())
		group.DbEntry.NumRetries++
	}
	return nil
}

func (tm *MonitorCompressedTxs) createNewGroups(pendingTx *PendingTxs) error {
	var err error
	if pendingTx == nil || pendingTx.TxCandidatesForGroup == nil {
		return nil
	}
	groupTxs := tm.triggerGroups.ChooseTxs(tm.timeProvider.Now(), pendingTx.TxCandidatesForGroup)
	if len(groupTxs) == 0 {
		log.Infof("pending claims: %d, not yet trigged", len(pendingTx.TxCandidatesForGroup))
		return nil
	}
	group := ctmtypes.NewMonitoredTxGroup(
		ctmtypes.MonitoredTxGroupDBEntry{
			GroupID: pendingTx.GenerateNewGroupID(),
			Status:  ctmtypes.MonitoredTxGroupStatusCreated,
			//To:      pendingTx.TxCandidatesForGroup[0].To,
			//From:    pendingTx.TxCandidatesForGroup[0].From,
			CreatedAt: tm.timeProvider.Now(),
		}, groupTxs)

	// Group createdTx and generate compress calls
	compressClaimParams, err := tm.compressClaimComposer.GetCompressClaimParametersFromMonitoredTx(group.Txs)
	if err != nil {
		return err
	}
	log.Debugf("calling smc.compressClaimParams for deposits_id=%s", group.GetTxsDepositIDString())
	compressedData, err := tm.etherMan.CompressClaimCall(compressClaimParams.MainnetExitRoot, compressClaimParams.RollupExitRoot, compressClaimParams.ClaimData)
	if err != nil {
		return fmt.Errorf("fails call to claimCompressorSMC for new group_id  with deposits %s err: %v", group.GetTxsDepositIDString(), err)
	}
	log.Infof("Compressed data len %d", len(compressedData))
	group.DbEntry.CompressedTxData = compressedData

	pendingTx.AddGroup(group)
	return nil
}
