package tokenlogoinfo

import (
	"github.com/0xPolygonHermez/zkevm-node/config/types"
)

type Config struct {
	Enabled              bool           `mapstructure:"Enabled"`
	LogoServiceNacosName string         `mapstructure:"LogoServiceNacosName"`
	Timeout              types.Duration `mapstructure:"TimeoutSeconds"`
}
