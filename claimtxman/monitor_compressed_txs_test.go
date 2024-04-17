package claimtxman_test

import (
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman"
	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/barkimedes/go-deepcopy"
	"github.com/stretchr/testify/require"
)

func TestDeepCopy(t *testing.T) {
	pendingTx := &claimtxman.PendingTxs{
		GroupTx:              make(map[uint64]*ctmtypes.MonitoredTxGroup),
		TxCandidatesForGroup: make([]ctmtypes.MonitoredTx, 0),
		LastGroupTxID:        0,
	}
	pendingTx.AddGroup(ctmtypes.MonitoredTxGroup{})

	initialStatus, err := deepcopy.Anything(pendingTx)
	require.NoError(t, err)
	copied := *initialStatus.(*claimtxman.PendingTxs)
	require.Equal(t, pendingTx, &copied)
}

func TestDeepCopy2(t *testing.T) {
	mTxs := make([]ctmtypes.MonitoredTx, 0)
	groups := make(map[uint64]ctmtypes.MonitoredTxGroupDBEntry)
	groups[uint64(1)] = ctmtypes.MonitoredTxGroupDBEntry{}

	lastGroupID := uint64(0)

	pendingTx, err := claimtxman.NewPendingTxs(mTxs, groups, lastGroupID)
	require.NoError(t, err)

	initialStatus, err := deepcopy.Anything(&pendingTx)
	require.NoError(t, err)
	copied := *initialStatus.(*claimtxman.PendingTxs)
	require.Equal(t, pendingTx, &copied)
}
