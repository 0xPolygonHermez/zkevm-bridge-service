package pgstorage

import (
	"context"
	"errors"
	"fmt"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/jackc/pgx/v4"
	"github.com/lib/pq"
)

// GetLatestGroupID
func (p *PostgresStorage) GetLatestMonitoredTxGroupID(ctx context.Context, dbTx pgx.Tx) (uint64, error) {
	const sql = `SELECT group_id FROM sync.monitored_txs_group ORDER BY group_id DESC LIMIT 1`
	var groupID uint64
	err := p.getExecQuerier(dbTx).QueryRow(ctx, sql).Scan(&groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return groupID, nil
}

// AddMonitoredTxsGroup
func (p *PostgresStorage) AddMonitoredTxsGroup(ctx context.Context, mTxGroup *ctmtypes.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error {
	const sql = `INSERT INTO sync.monitored_txs_group 
		(group_id,status, deposit_ids, num_retries, compressed_tx_data,claim_tx_history, created_at, updated_at) 
		VALUES($1, $2, $3, $4,$5, $6, $7, $8)
		`
	if mTxGroup == nil {
		return errors.New("nil monitored tx group")
	}
	claimTxHistoryStr, err := mTxGroup.ClaimTxHistory.ToJson()
	if err != nil {
		return fmt.Errorf("fails to convert claimTxHistory to json. Err: %w", err)
	}
	_, err = p.getExecQuerier(dbTx).Exec(ctx,
		sql,
		mTxGroup.GroupID,
		mTxGroup.Status,
		pq.Array(mTxGroup.DepositIDs),
		mTxGroup.NumRetries,
		mTxGroup.CompressedTxData,
		claimTxHistoryStr,
		mTxGroup.CreatedAt,
		mTxGroup.UpdatedAt)

	if err != nil {
		return err
	}

	return err
}

func (p *PostgresStorage) UpdateMonitoredTxsGroup(ctx context.Context, mTxGroup *ctmtypes.MonitoredTxGroupDBEntry, dbTx pgx.Tx) error {
	const sql = `UPDATE sync.monitored_txs_group 
		SET num_retries = $2, compressed_tx_data = $3, claim_tx_history = $4, updated_at = $5, status = $6, last_log = $7
		WHERE group_id = $1
		`
	if mTxGroup == nil {
		return errors.New("nil monitored tx group")
	}
	claimTxHistoryStr, err := mTxGroup.ClaimTxHistory.ToJson()
	if err != nil {
		return fmt.Errorf("fails to convert claimTxHistory to json. Err: %w", err)
	}
	_, err = p.getExecQuerier(dbTx).Exec(ctx,
		sql,
		mTxGroup.GroupID,
		mTxGroup.NumRetries,
		mTxGroup.CompressedTxData,
		claimTxHistoryStr,
		mTxGroup.UpdatedAt,
		mTxGroup.Status,
		mTxGroup.LastLog)

	if err != nil {
		return err
	}

	return err
}

func (p *PostgresStorage) GetMonitoredTxsGroups(ctx context.Context, groupIds []uint64, dbTx pgx.Tx) (map[uint64]ctmtypes.MonitoredTxGroupDBEntry, error) {
	const sql = "SELECT group_id, status, num_retries, compressed_tx_data,claim_tx_history, created_at, updated_at FROM sync.monitored_txs_group WHERE group_id = ANY($1) ORDER BY created_at ASC"
	groups := make(map[uint64]ctmtypes.MonitoredTxGroupDBEntry)
	rows, err := p.getExecQuerier(dbTx).Query(ctx, sql, pq.Array(groupIds))
	if errors.Is(err, pgx.ErrNoRows) {
		return groups, nil
	} else if err != nil {
		return groups, err
	}

	for rows.Next() {
		var group ctmtypes.MonitoredTxGroupDBEntry
		var claimTxHistoryStr string
		err = rows.Scan(&group.GroupID, &group.Status, &group.NumRetries, &group.CompressedTxData, &claimTxHistoryStr, &group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			return nil, err
		}
		group.ClaimTxHistory, err = ctmtypes.NewTxHistoryV2FromJson(claimTxHistoryStr)
		if err != nil {
			return nil, fmt.Errorf("fails to convert claimTxHistory from json. Err: %w", err)
		}
		groups[group.GroupID] = group
	}

	return groups, nil
}
