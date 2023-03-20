package claimtxman

import "github.com/0xPolygonHermez/zkevm-node/config/types"

// Config is configuration for L2 claim transaction manager
type Config struct {
	// FrequencyToMonitorTxs frequency of the resending failed txs
	FrequencyToMonitorTxs types.Duration `mapstructure:"FrequencyToMonitorTxs"`
	// PrivateKey defines the key store file that is going
	// to be read in order to provide the private key to sign the claim txs
	PrivateKey types.KeystoreFileConfig `mapstructure:"PrivateKey"`
}
