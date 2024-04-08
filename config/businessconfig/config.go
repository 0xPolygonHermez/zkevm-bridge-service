package businessconfig

import (
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	StandardChainIds []uint64 `mapstructure:"StandardChainIds"`
	InnerChainIds    []uint64 `mapstructure:"InnerChainIds"`

	USDCContractAddresses []common.Address `mapstructure:"USDCContractAddresses"`
	USDCTokenAddresses    []common.Address `mapstructure:"USDCTokenAddresses"`
}
