package apolloconfig

// Config is the configs to connect to remote Apollo server
type Config struct {
	Enabled              bool     `mapstructure:"Enabled"`
	AppID                string   `mapstructure:"AppID"`
	Cluster              string   `mapstructure:"Cluster"`
	MetaAddress          string   `mapstructure:"MetaAddress"`
	Namespaces           []string `mapstructure:"Namespaces"` // List of namespaces to be loaded to cache, separated by commas
	Secret               string   `mapstructure:"Secret"`
	IsBackupConfig       bool     `mapstructure:"IsBackupConfig"`
	DisableEntryDebugLog bool     `mapstructure:"DisableEntryDebugLog"`
}
