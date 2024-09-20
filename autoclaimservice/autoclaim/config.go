package autoclaim

import(
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
)
// Config represents the configuration of the AutoClaim package
type Config struct {
	// PrivateKey defines the key store file that is going
	// to be read in order to provide the private key to sign the claim txs
	PrivateKey types.KeystoreFileConfig `mapstructure:"PrivateKey"`
	// AuthorizedClaimMessageAddresses are the allowed address to bridge message with autoClaim
	AuthorizedClaimMessageAddresses []common.Address `mapstructure:"AuthorizedClaimMessageAddresses"`
	// AutoClaimInterval is time between each iteration
	AutoClaimInterval types.Duration `mapstructure:"AutoClaimInterval"`
	// GasOffset is the offset for the gas estimation
	GasOffset uint64 `mapstructure:"GasOffset"`
	// MaxNumberOfClaimsPerGroup is the maximum number of claims per group. 0 means group claims is disabled
	MaxNumberOfClaimsPerGroup int `mapstructure:"MaxNumberOfClaimsPerGroup"`
	// L2RPC is the URL of the L2 node
	L2RPC  string   `mapstructure:"L2RPC"`
	// BridgeURL is the URL of the bridge service
	BridgeURL string `mapstructure:"BridgeURL"`
}

// NetworkConfig is the configuration struct for the different environments.
type NetworkConfig struct {
	// PolygonBridgeAddress is the l2 bridge smc address
	PolygonBridgeAddress   common.Address `mapstructure:"PolygonBridgeAddress"`
	// PolygonBridgeAddress is the l2 claim compressor smc address. If it's not set, then group claims is disabled
	ClaimCompressorAddress common.Address `mapstructure:"ClaimCompressorAddress"`
}