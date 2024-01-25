package db

// Config struct
type Config struct {
	// Database type
	Database string `mapstructure:"Database"`

	// Database name
	Name string `mapstructure:"Name"`

	// User name
	User string `mapstructure:"User"`

	// Password of the user
	Password string `mapstructure:"Password"`

	// Host address
	Host string `mapstructure:"Host"`

	// Port Number
	Port string `mapstructure:"Port"`

	// MaxConns is the maximum number of connections in the pool.
	MaxConns int `mapstructure:"MaxConns"`

	// TableSuffix will be added to all table names, e.g. sync.root -> sync.root_test
	TableSuffix string `mapstructure:"TableSuffix" apollo:"DB.TableSuffix"`
}
