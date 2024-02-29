package metrics

type Config struct {
	Enabled bool   `mapstructure:"Enabled"`
	Port    string `mapstructure:"Port"`
}
