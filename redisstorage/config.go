package redisstorage

// Config stores the redis connection configs
type Config struct {
	// If this is true, will use ClusterClient
	IsClusterMode bool `mapstructure:"IsClusterMode"`

	// Host:Port address
	Addrs []string `mapstructure:"Addrs"`

	// Username for ACL
	Username string `mapstructure:"Username"`

	// Password for ACL
	Password string `mapstructure:"Password"`

	// DB index
	DB int `mapstructure:"DB"`

	MockPrice bool `mapstructure:"MockPrice"`
}
