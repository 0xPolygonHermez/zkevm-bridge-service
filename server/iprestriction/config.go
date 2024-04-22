package iprestriction

import (
	"github.com/0xPolygonHermez/zkevm-node/config/types"
)

type Config struct {
	Enabled  bool           `mapstructure:"Enabled"`
	UseNacos bool           `mapstructure:"UseNacos"`
	Host     string         `mapstructure:"Host"` // If UseNacos, Host is the nacos service name
	Timeout  types.Duration `mapstructure:"TimeoutSeconds"`

	// Additional restricted IP list, this should be used for testing only
	IPBlocklist []string `mapstructure:"IPBlockList"`
}
