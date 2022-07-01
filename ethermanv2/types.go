package ethermanv2

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-core/ethermanv2/smartcontracts/proofofefficiency"
)

// Block struct
type Block struct {
	ID                    uint64
	BlockNumber           uint64
	BlockHash             common.Hash
	ParentHash            common.Hash
	NetworkID             uint
	GlobalExitRoots       []GlobalExitRoot
	ForcedBatches         []ForcedBatch
	SequencedBatches      [][]SequencedBatch
	VerifiedBatches       []VerifiedBatch
	SequencedForceBatches []SequencedForceBatch
	Deposits              []Deposit
	Claims                []Claim
	Tokens                []TokenWrapped
	ReceivedAt            time.Time
}

// GlobalExitRoot struct
type GlobalExitRoot struct {
	BlockID           uint64
	GlobalExitRootNum *big.Int
	MainnetExitRoot   common.Hash
	RollupExitRoot    common.Hash
	GlobalExitRoot    common.Hash
}

// SequencedBatch represents virtual batches
type SequencedBatch struct {
	BatchNumber uint64
	Sequencer   common.Address
	TxHash      common.Hash
	proofofefficiency.ProofOfEfficiencyBatchData
}

// ForcedBatch represents a ForcedBatch
type ForcedBatch struct {
	BlockID           uint64
	BatchNumber       *uint64
	ForcedBatchNumber uint64
	Sequencer         common.Address
	GlobalExitRoot    common.Hash
	RawTxsData        []byte
	ForcedAt          time.Time
}

// SequencedForceBatch is a sturct to track the ForceSequencedBatches event.
type SequencedForceBatch struct {
	LastBatchSequenced uint64
	ForceBatchNumber   uint64
	Sequencer          common.Address
	TxHash             common.Hash
}

// Deposit struct
type Deposit struct {
	OriginalNetwork    uint
	TokenAddress       common.Address
	Amount             *big.Int
	DestinationNetwork uint
	DestinationAddress common.Address
	DepositCount       uint
	BlockID            uint64
	BlockNumber        uint64
	NetworkID          uint
	TxHash             common.Hash
}

// Claim struct
type Claim struct {
	Index              uint
	OriginalNetwork    uint
	Token              common.Address
	Amount             *big.Int
	DestinationAddress common.Address
	BlockID            uint64
	BlockNumber        uint64
	NetworkID          uint
	TxHash             common.Hash
}

// TokenWrapped struct
type TokenWrapped struct {
	OriginalNetwork      uint
	OriginalTokenAddress common.Address
	WrappedTokenAddress  common.Address
	BlockID              uint64
	BlockNumber          uint64
	NetworkID            uint
}

// Batch struct
type Batch struct {
	BatchNumber    uint64
	Coinbase       common.Address
	BatchL2Data    []byte
	Timestamp      time.Time
	GlobalExitRoot common.Hash
}

// VerifyBatch represents a VerifyBatch
type VerifiedBatch struct {
	BatchNumber uint64
	BlockID     uint64
	Aggregator  common.Address
	TxHash      common.Hash
}

// VirtualBatch represents a VirtualBatch
type VirtualBatch struct {
	BatchNumber uint64
	BlockID     uint64
	TxHash      common.Hash
	Sequencer   common.Address
}
