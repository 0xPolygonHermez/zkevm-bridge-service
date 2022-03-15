package etherman

// Config represents the configuration of the etherman
type Config struct {
	L1URL string `mapstructure:"L1URL"`
	L2URL []string `mapstructure:"L2URL"`

	PrivateKeyPath     string `mapstructure:"PrivateKeyPath"`
	PrivateKeyPassword string `mapstructure:"PrivateKeyPassword"`
}
