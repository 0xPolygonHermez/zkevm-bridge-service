package claimtxman

import "github.com/0xPolygonHermez/zkevm-node/config/types"

// Config is configuration for L2 claim transaction manager
type Config struct {
	// FrequencyToMonitorTxs frequency of the resending failed txs
	FrequencyToMonitorTxs types.Duration `mapstructure:"FrequencyToMonitorTxs"`
	// WaitTxToBeMined time to wait after transaction was sent to the ethereum
	WaitTxToBeMined types.Duration `mapstructure:"WaitTxToBeMined"`
	// PrivateKey defines the key store file that is going
	// to be read in order to provide the private key to sign the claim txs
	PrivateKey types.KeystoreFileConfig `mapstructure:"PrivateKey"`
}
