package client

// Config is a client config
type Config struct {
	L1NodeURL    string `mapstructure:"L1NodeURL"`
	L2NodeURL    string `mapstructure:"L2NodeURL"`
	BridgeURL    string `mapstructure:"BridgeURL"`
	L1BridgeAddr string `mapstructure:"L1BridgeAddr"`
	L2BridgeAddr string `mapstructure:"L2BridgeAddr"`
}
