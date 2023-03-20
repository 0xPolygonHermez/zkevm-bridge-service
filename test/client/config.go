package client

import "github.com/ethereum/go-ethereum/common"

// Config is a client config
type Config struct {
	L1NodeURL    string         `mapstructure:"L1NodeURL"`
	L2NodeURL    string         `mapstructure:"L2NodeURL"`
	BridgeURL    string         `mapstructure:"BridgeURL"`
	L1BridgeAddr common.Address `mapstructure:"L1BridgeAddr"`
	L2BridgeAddr common.Address `mapstructure:"L2BridgeAddr"`
}
