package etherman

// Config represents the configuration of the etherman
type Config struct {
	L1URL  string   `mapstructure:"L1URL"`
	L2URLs []string `mapstructure:"L2URLs"`
}
