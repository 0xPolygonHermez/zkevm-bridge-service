package coinmiddleware

// Config handles the kafka consumer config
type Config struct {
	// Brokers is the list of address of the kafka brokers
	Brokers []string `mapstructure:"Brokers"`

	// Topics is the topic names to be consumed from
	Topics []string `mapstructure:"Topics"`

	// ConsumerGroupID is the name of the consumer group
	ConsumerGroupID string `mapstructure:"ConsumerGroupID"`

	// InitialOffset is the offset to use if there's no previously committed offset
	// -1: Newest
	// -2: Oldest
	InitialOffset int64 `mapstructure:"InitialOffset"`

	// Username and Password are used for SASL_SSL authentication
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`

	// RootCAPath points to the CA cert used for authentication
	RootCAPath string `mapstructure:"RootCAPath"`
}
