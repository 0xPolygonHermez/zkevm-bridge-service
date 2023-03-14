package claimtxman

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// MonitoredTxStatusCreated mean the tx was just added to the storage
	MonitoredTxStatusCreated = MonitoredTxStatus("created")

	// MonitoredTxStatusSent means that at least a eth tx was sent to the network
	MonitoredTxStatusSent = MonitoredTxStatus("sent")

	// MonitoredTxStatusFailed means the tx was already mined and failed with an
	// error that can't be recovered automatically, ex: the data in the tx is invalid
	// and the tx gets reverted
	MonitoredTxStatusFailed = MonitoredTxStatus("failed")

	// MonitoredTxStatusConfirmed means the tx was already mined and the receipt
	// status is Successful
	MonitoredTxStatusConfirmed = MonitoredTxStatus("confirmed")

	// MonitoredTxStatusReorged is used when a monitored tx was already confirmed but
	// the L1 block where this tx was confirmed has been reorged, in this situation
	// the caller needs to review this information and wait until it gets confirmed
	// again in a future block
	MonitoredTxStatusReorged = MonitoredTxStatus("reorged")

	// MonitoredTxStatusDone means the tx was set by the owner as done
	MonitoredTxStatusDone = MonitoredTxStatus("done")
)

// MonitoredTxStatus represents the status of a monitored tx
type MonitoredTxStatus string

// String returns a string representation of the status
func (s MonitoredTxStatus) String() string {
	return string(s)
}

// MonitoredTx represents a set of information used to build tx
// plus information to monitor if the transactions was sent successfully
type MonitoredTx struct {
	// Id is the tx identifier controller by the caller
	Id string

	// From is a sender of the tx, used to identify which private key should be used to sing the tx
	From common.Address

	// To is a receiver of the tx
	To *common.Address

	// Nonce used to create the tx
	Nonce uint64

	// Value is a tx value
	Value *big.Int

	// Data is a tx data
	Data []byte

	// Gas is a tx gas
	Gas uint64

	// GasPrice is a tx gas price
	GasPrice *big.Int

	// Status of this monitoring
	Status MonitoredTxStatus

	// BlockID represents the block where the tx was identified
	// to be mined, it's the same as the block id found in the
	// tx receipt, this is used to control reorged monitored txs
	BlockID uint64

	// History represent all transaction hashes from
	// transactions created using this struct data and
	// sent to the network
	History map[common.Hash]bool

	// CreatedAt date time it was created
	CreatedAt time.Time

	// UpdatedAt last date time it was updated
	UpdatedAt time.Time
}

// Tx uses the current information to build a tx
func (mTx MonitoredTx) Tx() *types.Transaction {
	tx := types.NewTx(&types.LegacyTx{
		To:       mTx.To,
		Nonce:    mTx.Nonce,
		Value:    mTx.Value,
		Data:     mTx.Data,
		Gas:      mTx.Gas,
		GasPrice: mTx.GasPrice,
	})

	return tx
}

// AddHistory adds a transaction to the monitoring history
func (mTx MonitoredTx) AddHistory(tx *types.Transaction) error {
	if _, found := mTx.History[tx.Hash()]; found {
		return ErrAlreadyExists
	}
	mTx.History[tx.Hash()] = true
	return nil
}

// ToStringPtr returns the current to field as a string pointer
func (mTx *MonitoredTx) ToStringPtr() *string {
	var to *string
	if mTx.To != nil {
		s := mTx.To.String()
		to = &s
	}
	return to
}

// ValueU64Ptr returns the current value field as a uint64 pointer
func (mTx *MonitoredTx) ValueU64Ptr() *uint64 {
	var value *uint64
	if mTx.Value != nil {
		tmp := mTx.Value.Uint64()
		value = &tmp
	}
	return value
}

// DataStringPtr returns the current data field as a string pointer
func (mTx *MonitoredTx) DataStringPtr() *string {
	var data *string
	if mTx.Data != nil {
		tmp := hex.EncodeToString(mTx.Data)
		data = &tmp
	}
	return data
}

// HistoryStringSlice returns the current history field as a string slice
func (mTx *MonitoredTx) HistoryStringSlice() []string {
	history := make([]string, 0, len(mTx.History))
	for h := range mTx.History {
		history = append(history, h.String())
	}
	return history
}

// HistoryHashSlice returns the current history field as a string slice
func (mTx *MonitoredTx) HistoryHashSlice() []common.Hash {
	history := make([]common.Hash, 0, len(mTx.History))
	for h := range mTx.History {
		history = append(history, h)
	}
	return history
}

// MonitoredTxResult represents the result of a execution of a monitored tx
type MonitoredTxResult struct {
	ID     string
	Status MonitoredTxStatus
	Txs    map[common.Hash]TxResult
}

// TxResult represents the result of a execution of a ethereum transaction in the block chain
type TxResult struct {
	Tx            *types.Transaction
	Receipt       *types.Receipt
	RevertMessage string
}
