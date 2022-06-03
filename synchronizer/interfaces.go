package synchronizer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hermeznetwork/hermez-bridge/etherman"
)

type localEtherMan interface {
	GetBridgeInfoByBlockRange(ctx context.Context, fromBlock uint64, toBlock *uint64) ([]etherman.Block, map[common.Hash][]etherman.Order, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BlockByNumber(ctx context.Context, blockNumber uint64) (*types.Block, error)
	GetNetworkID(ctx context.Context) (uint, error)
	ForceBatch(ctx context.Context) error
}

// storageInterface gathers the methods required to interact with the state.
type storageInterface interface {
	GetLastBlock(ctx context.Context, networkID uint) (*etherman.Block, error)
	Rollback(ctx context.Context, index uint) error
	BeginDBTransaction(ctx context.Context, index uint) error
	Commit(ctx context.Context, index uint) error
	AddBlock(ctx context.Context, block *etherman.Block) (uint64, error)
	ConsolidateBatch(ctx context.Context, batch *etherman.Batch) error
	AddBatch(ctx context.Context, batch *etherman.Batch) error
	AddExitRoot(ctx context.Context, exitRoot *etherman.GlobalExitRoot) error
	AddDeposit(ctx context.Context, deposit *etherman.Deposit) error
	AddClaim(ctx context.Context, claim *etherman.Claim) error
	AddTokenWrapped(ctx context.Context, tokeWrapped *etherman.TokenWrapped) error
	Reset(ctx context.Context, block *etherman.Block, networkID uint) error
	GetPreviousBlock(ctx context.Context, networkID uint, offset uint64) (*etherman.Block, error)
	GetNumberDeposits(ctx context.Context, origNetworkID uint, blockNumber uint64) (uint64, error)
}
