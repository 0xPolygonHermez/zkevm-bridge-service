package messagepush

// Config is the config for the Kafka producer
type Config struct {
	Enabled bool `mapstructure:"Enabled"`

	// Due to some restriction in dev environment, bridge service cannot push message to okc-basic kafka
	// We need to implement a "fake producer" flow to let okc-basic get the events from bridge-service
	UseFakeProducer bool `mapstructure:"UseFakeProducer"`

	// Brokers is the list of address of the kafka brokers
	Brokers []string `mapstructure:"Brokers"`

	// Topic is the default topic name to send message to
	Topic   string `mapstructure:"Topic" apollo:"MessagePushProducer.Topic"`
	PushKey string `mapstructure:"PushKey" apollo:"MessagePushProducer.PushKey"`

	// Username and Password are used for SASL_SSL authentication
	Username string `mapstructure:"Username"`
	Password string `mapstructure:"Password"`

	// RootCAPath points to the CA cert used for authentication
	RootCAPath string `mapstructure:"RootCAPath"`
}
