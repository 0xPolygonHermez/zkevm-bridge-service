package metrics

type Config struct {
	Enabled bool   `mapstructure:"Enabled"`
	Port    string `mapstructure:"Port"`

	// Endpoint is the metrics endpoint for prometheus to query the metrics
	Endpoint string `mapstructure:"Endpoint"`

	// Env is the environment label for the metrics, to separate mainnet and testnet metrics
	Env string `mapstructure:"Env"`
}
