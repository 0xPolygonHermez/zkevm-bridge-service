package claimtxman

import (
	"github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/ethereum/go-ethereum/common"
)

// Config is configuration for L2 claim transaction manager
type Config struct {
	//Enabled whether to enable this module
	Enabled bool `mapstructure:"Enabled"`
	// FrequencyToMonitorTxs frequency of the resending failed txs
	FrequencyToMonitorTxs types.Duration `mapstructure:"FrequencyToMonitorTxs"`
	// PrivateKey defines the key store file that is going
	// to be read in order to provide the private key to sign the claim txs
	PrivateKey types.KeystoreFileConfig `mapstructure:"PrivateKey" apollo:"keystoreFileConfig"`
	// RetryInterval is time between each retry
	RetryInterval types.Duration `mapstructure:"RetryInterval"`
	// RetryNumber is the number of retries before giving up
	RetryNumber int `mapstructure:"RetryNumber"`
	// AuthorizedClaimMessageAddresses are the allowed address to bridge message with autoClaim
	AuthorizedClaimMessageAddresses []common.Address `mapstructure:"AuthorizedClaimMessageAddresses"`

	// X1
	// FreeGas enabled whether gas price is 0
	FreeGas bool `mapstructure:"FreeGas"`
	// OptClaim enabled store claimTx into storage every deposit
	OptClaim bool `mapstructure:"OptClaim"`
}
