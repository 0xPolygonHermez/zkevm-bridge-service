package autoclaim

import(
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
)
// Config represents the configuration of the AutoClaim package
type Config struct {
	// AuthorizedClaimMessageAddresses are the allowed address to bridge message with autoClaim
	AuthorizedClaimMessageAddresses []common.Address `mapstructure:"AuthorizedClaimMessageAddresses"`
	// AutoClaimInterval is time between each iteration
	AutoClaimInterval types.Duration `mapstructure:"AutoClaimInterval"`
	// MaxNumberOfClaimsPerGroup is the maximum number of claims per group. 0 means group claims is disabled
	MaxNumberOfClaimsPerGroup int `mapstructure:"MaxNumberOfClaimsPerGroup"`
	// BridgeURL is the URL of the bridge service
	BridgeURL string `mapstructure:"BridgeURL"`
}
