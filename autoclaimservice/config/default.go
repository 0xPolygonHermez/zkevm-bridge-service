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
AuthorizedClaimMessageAddresses = []
AutoClaimInterval = "10m"
MaxNumberOfClaimsPerGroup = 10
BridgeURL = "http://localhost:8080"

[BlockchainManager]
PrivateKey = {Path = "./test/test.keystore", Password = "testonly"}
L2RPC = "http://localhost:8123"
PolygonBridgeAddress = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
ClaimCompressorAddress = "0x0000000000000000000000000000000000000000"

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
