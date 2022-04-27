package etherman

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	// BatchesOrder identifies a batch event
	BatchesOrder EventOrder = "Batches"
	//DepositsOrder identifies a deposit event
	DepositsOrder EventOrder = "Deposits"
	//GlobalExitRootsOrder identifies a gloalExitRoot event
	GlobalExitRootsOrder EventOrder = "GlobalExitRoots"
	//ClaimsOrder identifies a claim event
	ClaimsOrder EventOrder = "Claims"
	//TokensOrder identifies a TokenWrapped event
	TokensOrder EventOrder = "Tokens"
)

// EventOrder is the the type used to identify the events order
type EventOrder string

// Order contains the event order to let the synchronizer store the information following this order
type Order struct {
	Name EventOrder
	Pos  int
}

// Block struct
type Block struct {
	ID              uint64
	BlockNumber     uint64
	BlockHash       common.Hash
	ParentHash      common.Hash
	NetworkID       uint
	Batches         []Batch
	Deposits        []Deposit
	GlobalExitRoots []GlobalExitRoot
	Claims          []Claim
	Tokens          []TokenWrapped

	ReceivedAt time.Time
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
}

// GlobalExitRoot struct
type GlobalExitRoot struct {
	BlockID             uint64
	BlockNumber         uint64
	GlobalExitRootNum   *big.Int
	GlobalExitRootL2Num *big.Int
	ExitRoots           []common.Hash
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

// Batch represents a batch
type Batch struct {
	BlockID            uint64
	BlockNumber        uint64
	NetworkID          uint
	Sequencer          common.Address
	Aggregator         common.Address
	ConsolidatedTxHash common.Hash
	ChainID            *big.Int
	GlobalExitRoot     common.Hash
	Header             *types.Header
	Uncles             []*types.Header
	ReceivedAt         time.Time
	ConsolidatedAt     *time.Time
}

// Number is a helper function to get the batch number from the header
func (b *Batch) Number() *big.Int {
	return b.Header.Number
}

// Hash returns the batch hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (b *Batch) Hash() common.Hash {
	return b.Header.Hash()
}
