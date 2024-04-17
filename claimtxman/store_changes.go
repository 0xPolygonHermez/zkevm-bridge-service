package claimtxman

import (
	"context"
	"fmt"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/jackc/pgx/v4"
)

type StoreChanges struct {
	AddGroups    []ctmtypes.MonitoredTxGroupDBEntry
	UpdateGroups []ctmtypes.MonitoredTxGroupDBEntry
	UpdateTxs    []ctmtypes.MonitoredTx
}

func NewStoreChanges() *StoreChanges {
	return &StoreChanges{}
}
func (a *StoreChanges) AddGroup(group ctmtypes.MonitoredTxGroupDBEntry) {
	a.AddGroups = append(a.AddGroups, group)
}
func (a *StoreChanges) UpdateGroup(group ctmtypes.MonitoredTxGroupDBEntry) {
	a.UpdateGroups = append(a.UpdateGroups, group)
}
func (a *StoreChanges) UpdateTx(tx ctmtypes.MonitoredTx) {
	a.UpdateTxs = append(a.UpdateTxs, tx)
}

func (a *StoreChanges) Execute(ctx context.Context, storage StorageCompressedInterface, dbTx pgx.Tx) error {
	for i := range a.AddGroups {
		log.Debugf("Adding group %d ", a.AddGroups[i].GroupID)
		err := storage.AddMonitoredTxsGroup(ctx, &a.AddGroups[i], dbTx)
		if err != nil {
			return fmt.Errorf("storeChanges.Execute error adding MonitoresTxGroup. Err: %w", err)
		}
		log.Infof("Added group %d", a.AddGroups[i].GroupID)
	}

	for i := range a.UpdateGroups {
		a.UpdateGroups[i].UpdatedAt = time.Now()
		err := storage.UpdateMonitoredTxsGroup(ctx, &a.UpdateGroups[i], dbTx)
		if err != nil {
			return err
		}
		log.Infof("Updated group %d", a.UpdateGroups[i].GroupID)
	}

	for i := range a.UpdateTxs {
		log.Debugf("Updating tx deposit_id:%d  group_id:%d", a.UpdateTxs[i].DepositID, a.UpdateTxs[i].GroupID)
		err := storage.UpdateClaimTx(ctx, a.UpdateTxs[i], dbTx)
		if err != nil {
			return fmt.Errorf("storeChanges.Execute error UpdateClaimTx. Err: %w", err)
		}
		groupIDstr := "<nil>"
		if a.UpdateTxs[i].GroupID != nil {
			groupIDstr = fmt.Sprintf("%d", *a.UpdateTxs[i].GroupID)
		}
		log.Infof("Updated tx deposit_id:%d  group_id:%s", a.UpdateTxs[i].DepositID, groupIDstr)
	}
	return nil
}
