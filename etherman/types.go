package etherman

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	//DepositsOrder identifies a deposit event
	DepositsOrder EventOrder = "Deposits"
	//GlobalExitRootsOrder identifies a gloalExitRoot event
	GlobalExitRootsOrder EventOrder = "GlobalExitRoots"
	//ClaimsOrder identifies a claim event
	ClaimsOrder EventOrder = "Claims"
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
	BlockNumber     uint64
	BlockHash       common.Hash
	ParentHash      common.Hash
	Deposits        []Deposit
	GlobalExitRoots []GlobalExitRoot
	Claims          []Claim

	ReceivedAt time.Time
}

// Deposit struct
type Deposit struct {
	OriginalNetwork    uint
	TokenAddress       common.Address
	Amount             *big.Int
	DestinationNetwork uint
	DestinationAddress common.Address
	BlockNumber        uint64
	DepositCount       uint64
}

// GlobalExitRoot struct
type GlobalExitRoot struct {
	MainnetExitRoot [32]byte
	RollupExitRoot  [32]byte
}

// Claim struct
type Claim struct {
	Index              uint64
	OriginalNetwork    uint
	Token              common.Address
	Amount             *big.Int
	DestinationAddress common.Address
	BlockNumber        uint64
}
