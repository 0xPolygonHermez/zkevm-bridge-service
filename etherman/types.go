package etherman

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Block struct
type Block struct {
	ID              uint64
	BlockNumber     uint64
	BlockHash       common.Hash
	ParentHash      common.Hash
	NetworkID       uint
	GlobalExitRoots []GlobalExitRoot
	Deposits        []Deposit
	Claims          []Claim
	Tokens          []TokenWrapped
	VerifiedBatches []VerifiedBatch
	ActivateEtrog   []bool
	ReceivedAt      time.Time
}

// GlobalExitRoot struct
type GlobalExitRoot struct {
	BlockID        uint64
	BlockNumber    uint64
	ExitRoots      []common.Hash
	GlobalExitRoot common.Hash
}

// Deposit struct
type Deposit struct {
	LeafType             uint8
	OriginalTokenNetwork uint
	OriginalTokenAddress common.Address
	Amount               *big.Int
	DestinationNetwork   uint
	DestinationAddress   common.Address
	DepositCount         uint
	BlockID              uint64
	BlockNumber          uint64
	OriginNetwork        uint
	TxHash               common.Hash
	Metadata             []byte
	// it is only used for the bridge service
	ReadyForClaim bool
}

// Claim struct
type Claim struct {
	MainnetFlag          bool
	RollupIndex          uint64
	DepositCount         uint
	OriginalTokenNetwork uint
	OriginalTokenAddress common.Address
	Amount               *big.Int
	DestinationAddress   common.Address
	BlockID              uint64
	BlockNumber          uint64
	DestinationNetwork   uint
	TxHash               common.Hash
}

// OriginNetwork returns the rollup ID of the claim
func (c *Claim) OriginNetwork() uint {
	if c.MainnetFlag {
		return 0
	}
	return uint(c.RollupIndex + 1)
}

// TokenWrapped struct
type TokenWrapped struct {
	TokenMetadata
	OriginalNetwork      uint
	OriginalTokenAddress common.Address
	WrappedTokenAddress  common.Address
	BlockID              uint64
	BlockNumber          uint64
	NetworkID            uint
}

// TokenMetadata is a metadata of ERC20 token.
type TokenMetadata struct {
	Name     string
	Symbol   string
	Decimals uint8
}

type VerifiedBatch struct {
	BlockNumber   uint64
	BatchNumber   uint64
	RollupID      uint
	LocalExitRoot common.Hash
	TxHash        common.Hash
	StateRoot     common.Hash
	Aggregator    common.Address
}

// RollupExitLeaf struct
type RollupExitLeaf struct {
	ID       uint64
	BlockID  uint64
	Leaf     common.Hash
	RollupId uint
	Root     common.Hash
}
