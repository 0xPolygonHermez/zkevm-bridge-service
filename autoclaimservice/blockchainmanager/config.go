package blockchainmanager

import(
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
)
// Config is the configuration struct for the different environments.
type Config struct {
	// L2RPC is the URL of the L2 node
	L2RPC  string   `mapstructure:"L2RPC"`
	// PrivateKey defines the key store file that is going
	// to be read in order to provide the private key to sign the claim txs
	PrivateKey types.KeystoreFileConfig `mapstructure:"PrivateKey"`
	// PolygonBridgeAddress is the l2 bridge smc address
	PolygonBridgeAddress   common.Address `mapstructure:"PolygonBridgeAddress"`
	// ClaimCompressorAddress is the l2 claim compressor smc address. If it's not set, then group claims is disabled
	ClaimCompressorAddress common.Address `mapstructure:"ClaimCompressorAddress"`
}