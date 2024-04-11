package metrics

type Config struct {
	Enabled bool   `mapstructure:"Enabled"`
	Port    string `mapstructure:"Port"`

	// Environment label for the metrics, to separate mainnet and testnet metrics
	Env string `mapstructure:"Env"`
}
