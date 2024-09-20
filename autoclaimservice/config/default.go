package config

import (
	"bytes"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// DefaultValues is the default configuration
const DefaultValues = `
[Log]
Level = "debug"
Outputs = ["stdout"]

[AutoClaim]
PrivateKey = {Path = "./test/test.keystore", Password = "testonly"}
AuthorizedClaimMessageAddresses = []
AutoClaimInterval = "10m"
GasOffset = 0
DisableGroupClaims = false
MaxNumberOfClaimsPerGroup = 10
L2RPC = "http://localhost:8123"
BridgeURL = "http://localhost:8080"
`

// Default parses the default configuration values.
func Default() (*Config, error) {
	var cfg Config
	viper.SetConfigType("toml")

	err := viper.ReadConfig(bytes.NewBuffer([]byte(DefaultValues)))
	if err != nil {
		return nil, err
	}
	err = viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
