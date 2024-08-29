package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	MonitoredTxStatusCompressing = MonitoredTxStatus("compressing")

	MonitoredTxStatusClaiming = MonitoredTxStatus("claimiming")
)

type TxMonitorer interface {
	MonitorTxs(ctx context.Context) error
}

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
	DepositID uint64

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

	// GroupID is the group id of the tx if have it (could be null)
	GroupID *uint64

	// GlobalExitRoot is the ger used to get the merkle proof
	GlobalExitRoot common.Hash
}

// MonitoredTxGroupStatus represents the status of a monitored tx
type MonitoredTxGroupStatus string

const (
	// MonitoredTxStatusCreated means compress txs data but no send any claim yet
	MonitoredTxGroupStatusCreated = MonitoredTxGroupStatus("created")

	// MonitoredTxGroupStatusClaiming means it have send a claim tx and is waiting for response
	MonitoredTxGroupStatusClaiming = MonitoredTxGroupStatus("claiming")

	// MonitoredTxGroupStatussFailed reach maximum retries and failed
	MonitoredTxGroupStatussFailed = MonitoredTxGroupStatus("failed")

	// MonitoredTxStatusConfirmed means the tx was already mined and the receipt status is Successful
	MonitoredTxGroupStatusConfirmed = MonitoredTxGroupStatus("confirmed")
)

// String returns a string representation of the status
func (s MonitoredTxGroupStatus) String() string {
	return string(s)
}

type MonitoredTxGroupDBEntry struct {
	// GroupID is the group id of the tx if have it (could be null)
	GroupID uint64

	Status MonitoredTxGroupStatus

	DepositIDs []uint64

	// Result of compressing all claimAsset and claimMessage txs of the group
	CompressedTxData []byte

	ClaimTxHistory *TxHistoryV2
	// CreatedAt date time it was created
	CreatedAt time.Time

	// UpdatedAt last date time it was updated
	UpdatedAt time.Time

	NumRetries int32
	// LastLog is a textual status of the last action that taken and it's relevant
	LastLog string
}

func (m *MonitoredTxGroupDBEntry) AddPendingTx(txHash common.Hash) uint64 {
	if m.ClaimTxHistory == nil {
		m.ClaimTxHistory = &TxHistoryV2{
			Version: 1,
		}
	}
	m.ClaimTxHistory.AddPendingTx(txHash)
	return m.GroupID
}

func (m *MonitoredTxGroupDBEntry) IsClaimTxHistoryEmpty() bool {
	if m.ClaimTxHistory == nil {
		return true
	}
	return len(m.ClaimTxHistory.TxHashes) == 0
}

type MonitoredTxGroup struct {
	DbEntry MonitoredTxGroupDBEntry
	Txs     []MonitoredTx
}

func NewMonitoredTxGroup(entry MonitoredTxGroupDBEntry, txs []MonitoredTx) MonitoredTxGroup {
	res := MonitoredTxGroup{
		DbEntry: entry,
		Txs:     txs,
	}
	res.AssignTxsGroupID()
	return res
}

func (m MonitoredTxGroup) GetTxByDepositID(depositID uint64) *MonitoredTx {
	for idx := range m.Txs {
		if m.Txs[idx].DepositID == depositID {
			return &m.Txs[idx]
		}
	}
	return nil
}

func (m *MonitoredTxGroup) AddTx(tx MonitoredTx) {
	tx.GroupID = &m.DbEntry.GroupID
	m.Txs = append(m.Txs, tx)
}

func (m MonitoredTxGroup) GetTxsDepositIDString() string {
	result := ""
	for idx := range m.Txs {
		result += fmt.Sprintf("%d ", m.Txs[idx].DepositID)
	}
	return result
}

func (m MonitoredTxGroup) GetTxsDepositID() []uint64 {
	result := []uint64{}
	for idx := range m.Txs {
		result = append(result, uint64(m.Txs[idx].DepositID))
	}
	return result
}

func (m *MonitoredTxGroup) AssignTxsGroupID() {
	for idx := range m.Txs {
		m.Txs[idx].GroupID = &m.DbEntry.GroupID
	}
}

type TxHistoryV2 struct {
	Version  uint64
	TxHashes []TxHashHistoryEntry
}

func NewTxHistoryV2FromJson(jsonStr string) (*TxHistoryV2, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var txHistory TxHistoryV2
	err := json.Unmarshal([]byte(jsonStr), &txHistory)
	if err != nil {
		return nil, err
	}
	return &txHistory, nil
}

func (t *TxHistoryV2) GetMoreRecentTx() *TxHashHistoryEntry {
	if len(t.TxHashes) == 0 {
		return nil
	}
	maxTime := t.TxHashes[0].CreatedAt
	maxIdx := 0
	for idx := range t.TxHashes {
		if t.TxHashes[idx].CreatedAt.After(maxTime) {
			maxTime = t.TxHashes[idx].CreatedAt
			maxIdx = idx
		}
	}
	return &t.TxHashes[maxIdx]
}

func (t *TxHistoryV2) ToJson() (string, error) {
	if t == nil {
		return "", nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *TxHistoryV2) AddPendingTx(txHash common.Hash) {
	t.TxHashes = append(t.TxHashes, TxHashHistoryEntry{
		TxHash:        txHash,
		ReceiptStatus: nil,
		CreatedAt:     time.Now(),
	})
}

const (
	ReceiptStatusFailed     = 0
	ReceiptStatusSuccessful = 1
	ReceiptStatusOutdated   = 2
)

type TxHashHistoryEntry struct {
	TxHash common.Hash
	// null, ReceiptStatusFailed or ReceiptStatusSuccessful
	ReceiptStatus *uint64
	// CreatedAt date time it was created
	CreatedAt time.Time
}

func (t *TxHashHistoryEntry) IsPending() bool {
	return t.ReceiptStatus == nil
}
func (t *TxHashHistoryEntry) IsFailed() bool {
	if t.ReceiptStatus == nil {
		return false
	}
	return *t.ReceiptStatus == ReceiptStatusFailed
}

func (t *TxHashHistoryEntry) IsSuccessful() bool {
	if t.ReceiptStatus == nil {
		return false
	}
	return *t.ReceiptStatus == ReceiptStatusSuccessful
}
func (t *TxHashHistoryEntry) IsOutdated() bool {
	if t.ReceiptStatus == nil {
		return false
	}
	return *t.ReceiptStatus == ReceiptStatusOutdated
}

func (t *TxHashHistoryEntry) Outdate() {
	t.ReceiptStatus = new(uint64)
	*t.ReceiptStatus = ReceiptStatusOutdated
}

func (t *TxHashHistoryEntry) ReceiptFailed() {
	t.ReceiptStatus = new(uint64)
	*t.ReceiptStatus = ReceiptStatusFailed
}

func (t *TxHashHistoryEntry) ReceiptSuccessful() {
	t.ReceiptStatus = new(uint64)
	*t.ReceiptStatus = ReceiptStatusSuccessful
}

func (t *TxHashHistoryEntry) IsExhaustedTimeWaitingForReceipt(now time.Time, maxTime time.Duration) bool {
	return now.Sub(t.CreatedAt) > maxTime
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

// IsCandidateForGroup returns true if the tx is a candidate to be grouped
func (mTx *MonitoredTx) IsCandidateToBeGrouped(ger common.Hash) bool {
	return mTx.Status == MonitoredTxStatusCreated && len(mTx.History) == 0 && mTx.GroupID == nil && mTx.GlobalExitRoot == ger
}
