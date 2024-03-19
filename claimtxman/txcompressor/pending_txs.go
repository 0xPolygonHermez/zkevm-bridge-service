package txcompressor

import (
	"fmt"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
)

type GrupedTxs struct {
	// Status can be Compressing or Claiming
	Status ctmtypes.MonitoredTxStatus
	// Txs is a list of monitored txs
	Txs []ctmtypes.MonitoredTx
	// CompressingTxHash is the hash of the tx that will be used to compress the txs
	CompressingTxHash *common.Hash
	// ClaimingTxHash is the hash of the tx that will be used to claim the txs
	ClaimingTxHash *common.Hash
}

type PendingTxs struct {
	GroupTx              map[uint64]*ctmtypes.MonitoredTxGroup
	TxCandidatesForGroup []ctmtypes.MonitoredTx
	LastGroupTxID        uint64
}

func (m *PendingTxs) IsEmpty() bool {
	return len(m.GroupTx) == 0 && len(m.TxCandidatesForGroup) == 0
}

func (m *PendingTxs) GenerateNewGroupID() uint64 {
	m.LastGroupTxID++
	return m.LastGroupTxID
}

func (m *PendingTxs) AddTxCandidatesForGroup(tx ctmtypes.MonitoredTx) {
	m.TxCandidatesForGroup = append(m.TxCandidatesForGroup, tx)
}
func (m *PendingTxs) AddTxToGroup(group uint64, tx ctmtypes.MonitoredTx) {
	if _, ok := m.GroupTx[group]; !ok {
		m.GroupTx[group] = &ctmtypes.MonitoredTxGroup{}
	}
	m.GroupTx[group].AddTx(tx)
}

func (m *PendingTxs) AddGroup(group ctmtypes.MonitoredTxGroup) {
	if _, ok := m.GroupTx[group.DbEntry.GroupID]; ok {
		// Is already added
		return
	}
	m.GroupTx[group.DbEntry.GroupID] = &group
}

func (m *PendingTxs) SetGroupDBEntry(group ctmtypes.MonitoredTxGroupDBEntry) {
	if g, ok := m.GroupTx[group.GroupID]; ok {
		g.DbEntry = group
		m.GroupTx[group.GroupID] = g
	} else {
		m.GroupTx[group.GroupID] = &ctmtypes.MonitoredTxGroup{
			DbEntry: group,
		}
	}
	m.GroupTx[group.GroupID].AssignTxsGroupID()
}

func NewPendingTxs(mTxs []ctmtypes.MonitoredTx, groups map[uint64]ctmtypes.MonitoredTxGroupDBEntry, lastGroupID uint64) (PendingTxs, error) {
	result := PendingTxs{
		GroupTx:       make(map[uint64]*ctmtypes.MonitoredTxGroup),
		LastGroupTxID: lastGroupID,
	}
	for _, tx := range mTxs {
		// just Created are not grouped yet
		if tx.IsCandidateToBeGrouped() {
			result.AddTxCandidatesForGroup(tx)
			continue
		}
		if tx.GroupID != nil {
			groupEntry, ok := groups[*tx.GroupID]
			if !ok {
				return PendingTxs{}, fmt.Errorf("group %d not found", *tx.GroupID)
			}
			result.SetGroupDBEntry(groupEntry)
			result.AddTxToGroup(*tx.GroupID, tx)
		}
	}
	return result, nil
}

// GenerateStoreUpdate generates the changes that need to be done in the storage
func GenerateStoreUpdate(oldState, newState PendingTxs, timeProvider utils.TimeProvider) (*StoreChanges, error) {
	result := NewStoreChanges()
	// Update groups
	for groupID, group := range newState.GroupTx {
		if oldGroup, ok := oldState.GroupTx[groupID]; !ok {
			// Add group
			group.DbEntry.UpdatedAt = timeProvider.Now()
			group.DbEntry.DepositIDs = group.GetTxsDepositID()
			result.AddGroup(group.DbEntry)
		} else {
			if cmp.Equal(oldGroup.DbEntry, group.DbEntry) {
				continue
			}
			group.DbEntry.UpdatedAt = timeProvider.Now()
			result.UpdateGroup(group.DbEntry)
		}
	}
	// Update txs
	for groupID, group := range newState.GroupTx {
		if oldGroup, ok := oldState.GroupTx[groupID]; ok {
			// Check if the txs are the same, if not add the txs to the update list
			for _, tx := range group.Txs {
				oldTx := oldGroup.GetTxByDipositID(tx.DepositID)
				if oldTx == nil {
					return nil, fmt.Errorf("tx with depositID %d not found in old state", tx.DepositID)
				}
				newTx := tx
				if !cmp.Equal(*oldTx, newTx) {
					result.UpdateTx(tx)
				}
			}
		} else {
			// The group is new, so update all the txs
			for _, tx := range group.Txs {
				result.UpdateTx(tx)
			}
		}
	}
	return result, nil
}
