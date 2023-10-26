package types

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// MonitoredTxStatusCreated mean the tx was just added to the storage
	MonitoredTxStatusCreated = MonitoredTxStatus("created")

	// MonitoredTxStatusFailed means the tx was already mined and failed with an
	// error that can't be recovered automatically, ex: the data in the tx is invalid
	// and the tx gets reverted
	MonitoredTxStatusFailed = MonitoredTxStatus("failed")

	// MonitoredTxStatusConfirmed means the tx was already mined and the receipt
	// status is Successful
	MonitoredTxStatusConfirmed = MonitoredTxStatus("confirmed")
)

var (
	// ErrAlreadyExists when the object already exists
	ErrAlreadyExists = errors.New("already exists")
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
	// DepositID is the tx identifier controller by the caller
	DepositID uint

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

	// GasPrice is the tx gas price
	GasPrice *big.Int

	// Status of this monitoring
	Status MonitoredTxStatus

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

// RemoveHistory removes a transaction from the monitoring history
func (mTx MonitoredTx) RemoveHistory(tx *types.Transaction) {
	delete(mTx.History, tx.Hash())
}

// HistoryHashSlice returns the current history field as a string slice
func (mTx *MonitoredTx) HistoryHashSlice() [][]byte {
	history := make([][]byte, 0, len(mTx.History))
	for h := range mTx.History {
		history = append(history, h.Bytes())
	}
	return history
}
