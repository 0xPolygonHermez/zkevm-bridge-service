package synchronizer

import (
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
)

// Config represents the configuration of the synchronizer
type Config struct {
	// SyncInterval is the delay interval between reading new rollup information
	SyncInterval types.Duration `mapstructure:"SyncInterval"`

	// SyncChunkSize is the number of blocks to sync on each chunk
	SyncChunkSize uint64 `mapstructure:"SyncChunkSize"`

	// USDCContractAddresses is the list of contract addresses for USDC LxLy feature
	// For these address, the deposit needs to be modified to extract the user address from the metadata
	USDCContractAddresses []common.Address `mapstructure:"USDCContractAddresses"`
}
