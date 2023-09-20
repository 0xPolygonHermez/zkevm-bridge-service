package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryHashSlice(t *testing.T) {
	mTx := MonitoredTx{
		History: make(map[common.Hash]bool),
	}
	tx1 := types.NewTransaction(0, common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"), big.NewInt(10), 100000, big.NewInt(1000000000), []byte{})
	tx2 := types.NewTransaction(1, common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"), big.NewInt(11), 100001, big.NewInt(1000000010), []byte{})
	txs := []*types.Transaction{tx1, tx2}
	err := mTx.AddHistory(tx1)
	require.NoError(t, err)
	history := mTx.HistoryHashSlice()
	for i := range history {
		t.Logf("history %d: %s", i, common.Bytes2Hex(history[i]))
		assert.Equal(t, txs[i].Hash(), common.BytesToHash(history[i]))
		t.Log("TEST1: ", txs[i].Hash(), common.BytesToHash(history[i]))
	}

	err = mTx.AddHistory(tx2)
	require.NoError(t, err)
	history = mTx.HistoryHashSlice()
	for i := range history {
		t.Logf("history %d: %s", i, common.Bytes2Hex(history[i]))
		assert.Equal(t, txs[i].Hash(), common.BytesToHash(history[i]))
		t.Log("TEST2: ", txs[i].Hash(), common.BytesToHash(history[i]))
	}

	mTx.RemoveHistory(tx1)
	history = mTx.HistoryHashSlice()
	t.Logf("history %d: %s", 0, common.Bytes2Hex(history[0]))
	assert.Equal(t, txs[1].Hash(), common.BytesToHash(history[0]))
	t.Log("TEST3: ", txs[1].Hash(), common.BytesToHash(history[0]))
}
