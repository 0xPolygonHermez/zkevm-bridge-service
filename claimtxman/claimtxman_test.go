package claimtxman

import (
	"context"
	"math/big"
	"testing"
	"time"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Test monitored txs storage apis
func TestMonitoredTxStorage(t *testing.T) {
	ctx := context.Background()
	dbCfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)
	pg, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	var _ storageInterface = pg
	tx, err := pg.BeginDBTransaction(ctx)
	require.NoError(t, err)

	block := &etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}
	blockID, err := pg.AddBlock(ctx, block, tx)
	require.NoError(t, err)

	deposit := &etherman.Deposit{
		NetworkID:          0,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		BlockNumber:        1,
		BlockID:            blockID,
		DepositCount:       1,
		Metadata:           common.FromHex("0x0"),
	}
	_, err = pg.AddDeposit(ctx, deposit, tx)
	require.NoError(t, err)

	toAdr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	mTx := ctmtypes.MonitoredTx{
		ID:      1,
		BlockID: blockID,
		From:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		To:      &toAdr,
		Nonce:   1,
		Value:   big.NewInt(1000000),
		Data:    common.FromHex("0x0"),
		Gas:     1000000,
		Status:  ctmtypes.MonitoredTxStatusCreated,
		History: make(map[common.Hash]bool),
	}
	err = pg.AddClaimTx(ctx, mTx, tx)
	require.NoError(t, err)

	require.NoError(t, mTx.AddHistory(types.NewTx(&types.LegacyTx{
		To:    mTx.To,
		Nonce: mTx.Nonce,
		Value: mTx.Value,
		Data:  mTx.Data,
		Gas:   mTx.Gas,
	})))
	err = pg.UpdateClaimTx(ctx, mTx, tx)
	require.NoError(t, err)

	mTx = ctmtypes.MonitoredTx{
		ID:      2,
		BlockID: blockID,
		From:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		To:      &toAdr,
		Nonce:   1,
		Value:   big.NewInt(1000000),
		Data:    common.FromHex("0x0"),
		Gas:     1000000,
		Status:  ctmtypes.MonitoredTxStatusConfirmed,
		History: make(map[common.Hash]bool),
	}
	err = pg.AddClaimTx(ctx, mTx, tx)
	require.NoError(t, err)

	mTxs, err := pg.GetClaimTxsByStatus(ctx, []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatusCreated}, tx)
	require.NoError(t, err)
	require.Len(t, mTxs, 1)

	mTxs, err = pg.GetClaimTxsByStatus(ctx, []ctmtypes.MonitoredTxStatus{ctmtypes.MonitoredTxStatusCreated, ctmtypes.MonitoredTxStatusConfirmed}, tx)
	require.NoError(t, err)
	require.Len(t, mTxs, 2)

	require.NoError(t, tx.Commit(ctx))
}

// Test the update deposit status logic
func TestUpdateDepositStatus(t *testing.T) {
	ctx := context.Background()
	dbCfg := pgstorage.NewConfigFromEnv()
	err := pgstorage.InitOrReset(dbCfg)
	require.NoError(t, err)
	pg, err := pgstorage.NewPostgresStorage(dbCfg)
	require.NoError(t, err)

	block := &etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f1"),
		ParentHash:  common.HexToHash("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9f2"),
		NetworkID:   0,
		ReceivedAt:  time.Now(),
	}
	blockID, err := pg.AddBlock(ctx, block, nil)
	require.NoError(t, err)

	deposit := &etherman.Deposit{
		NetworkID:          0,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		BlockNumber:        1,
		BlockID:            blockID,
		DepositCount:       1,
		Metadata:           common.FromHex("0x0"),
	}
	depositID, err := pg.AddDeposit(ctx, deposit, nil)
	require.NoError(t, err)
	l1Root := common.FromHex("0x838c5655cb21c6cb83313b5a631175dff4963772cce9108188b34ac87c81c41e")
	require.NoError(t, pg.SetRoot(ctx, l1Root, depositID, deposit.DepositCount, deposit.NetworkID, nil))

	block = &etherman.Block{
		BlockNumber: 1,
		BlockHash:   common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		ParentHash:  common.HexToHash("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30"),
		NetworkID:   1,
		ReceivedAt:  time.Now(),
	}
	blockID, err = pg.AddBlock(ctx, block, nil)
	require.NoError(t, err)

	destAdr := "0x4d5Cf5032B2a844602278b01199ED191A86c93ff"
	deposit = &etherman.Deposit{
		NetworkID:          1,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress(destAdr),
		BlockNumber:        1,
		BlockID:            blockID,
		DepositCount:       1,
		Metadata:           common.FromHex("0x0"),
	}
	depositID, err = pg.AddDeposit(ctx, deposit, nil)
	require.NoError(t, err)
	l2Root := common.FromHex("0xb4c11951957c6f8f642c4af61cd6b24640fec6dc7fc607ee8206a99e92410d30")
	require.NoError(t, pg.SetRoot(ctx, l2Root, depositID, deposit.DepositCount, deposit.NetworkID, nil))

	deposit = &etherman.Deposit{
		NetworkID:          1,
		OriginalNetwork:    0,
		OriginalAddress:    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
		Amount:             big.NewInt(1000000),
		DestinationNetwork: 1,
		DestinationAddress: common.HexToAddress(destAdr),
		BlockNumber:        1,
		BlockID:            blockID,
		DepositCount:       2,
		Metadata:           common.FromHex("0x0"),
	}
	depositID, err = pg.AddDeposit(ctx, deposit, nil)
	require.NoError(t, err)
	l2Root1 := common.FromHex("0xda7bce9f4e8618b6bd2f4132ce798cdc7a60e7e1460a7299e3c6342a579626d2")
	require.NoError(t, pg.SetRoot(ctx, l2Root1, depositID, deposit.DepositCount, deposit.NetworkID, nil))

	deposits, err := pg.UpdateL1DepositsStatus(ctx, l1Root, nil)
	require.NoError(t, err)
	require.Len(t, deposits, 1)
	require.True(t, deposits[0].ReadyForClaim)
	require.Equal(t, deposits[0].DepositCount, uint(1))
	require.Equal(t, deposits[0].NetworkID, uint(0))

	require.NoError(t, pg.UpdateL2DepositsStatus(ctx, l2Root, nil))
	deposits, err = pg.GetDeposits(ctx, destAdr, 10, 0, nil)
	require.NoError(t, err)
	require.Len(t, deposits, 2)
	require.True(t, deposits[1].ReadyForClaim)
	require.False(t, deposits[0].ReadyForClaim)
}
