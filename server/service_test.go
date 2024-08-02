package server

import (
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetClaimProofbyGER(t *testing.T) {
	cfg := Config{
		CacheSize: 32,
	}
	mockStorage := newBridgeServiceStorageMock(t)
	sut := NewBridgeService(cfg, 32, []uint{0, 1}, mockStorage, 1)
	depositCnt := uint(0)
	networkID := uint(0)
	GER := common.Hash{}
	deposit := &etherman.Deposit{}
	mockStorage.EXPECT().GetDeposit(mock.Anything, depositCnt, networkID, mock.Anything).Return(deposit, nil)
	exitRoot := etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.Hash{}, common.Hash{}},
	}
	mockStorage.EXPECT().GetExitRootByGER(mock.Anything, GER, mock.Anything).Return(&exitRoot, nil)
	node := [][]byte{{}, {}}
	mockStorage.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(node, nil)
	smtProof, smtRollupProof, globaExitRoot, err := sut.GetClaimProofbyGER(depositCnt, networkID, GER, nil)
	require.NoError(t, err)
	require.NotNil(t, smtProof)
	require.NotNil(t, smtRollupProof)
	require.NotNil(t, globaExitRoot)
}
