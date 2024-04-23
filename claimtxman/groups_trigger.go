package claimtxman

import (
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
)

type GroupsTrigger struct {
	Cfg ConfigGroupingClaims
}

func NewGroupsTrigger(cfg ConfigGroupingClaims) *GroupsTrigger {
	return &GroupsTrigger{Cfg: cfg}
}

func (t *GroupsTrigger) ChooseTxs(now time.Time, TxCandidatesForGroup []ctmtypes.MonitoredTx) []ctmtypes.MonitoredTx {
	if t.isRetainedPeriodSurpassed(now, TxCandidatesForGroup) {
		return t.chooseGroupTx(TxCandidatesForGroup)
	}
	if len(TxCandidatesForGroup) >= t.Cfg.TriggerNumberOfClaims {
		return t.chooseGroupTx(TxCandidatesForGroup)
	}
	return nil
}

func (t *GroupsTrigger) isRetainedPeriodSurpassed(now time.Time, TxCandidatesForGroup []ctmtypes.MonitoredTx) bool {
	for _, tx := range TxCandidatesForGroup {
		elapsed := now.Sub(tx.CreatedAt)
		if elapsed > t.Cfg.TriggerRetainedClaimPeriod.Duration {
			log.Debugf("Claim deposit_id: %d has surpassed the retained period, elapsed: %s", tx.DepositID, elapsed)
			return true
		}
	}
	return false
}

func (t *GroupsTrigger) chooseGroupTx(TxCandidatesForGroup []ctmtypes.MonitoredTx) []ctmtypes.MonitoredTx {
	group := []ctmtypes.MonitoredTx{}
	for _, tx := range TxCandidatesForGroup {
		group = append(group, tx)
		if len(group) == t.Cfg.MaxNumberOfClaimsPerGroup {
			break
		}
	}
	return group
}
