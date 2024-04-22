package iprestriction

type Config struct {
	Enabled        bool   `mapstructure:"Enabled"`
	UseNacos       bool   `mapstructure:"UseNacos"`
	Host           string `mapstructure:"Host"` // If UseNacos, Host is the nacos service name
	TimeoutSeconds int    `mapstructure:"TimeoutSeconds"`

	// Additional restricted IP list, this should be used for testing only
	IPBlocklist []string `mapstructure:"IPBlockList"`
}
