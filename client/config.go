package client

type Config struct {
	L1NodeURL string `mapstructure:"L1NodeURL"`
	L2NodeURL string `mapstructure:"L2NodeURL"`
	BridgeURL string `mapstructure:"BridgeURL"`
}
