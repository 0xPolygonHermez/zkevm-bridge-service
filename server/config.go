package server

import "github.com/0xPolygonHermez/zkevm-bridge-service/db"

// Config struct
type Config struct {
	// GRPCPort is TCP port to listen by gRPC server
	GRPCPort string `mapstructure:"GRPCPort"`
	// HTTPPort is TCP port to listen by HTTP/REST gateway
	HTTPPort string `mapstructure:"HTTPPort"`
	// CacheSize is the buffer size of the lru-cache
	CacheSize int `mapstructure:"CacheSize"`
	// DefaultPageLimit is the default page limit for pagination
	DefaultPageLimit uint32 `mapstructure:"DefaultPageLimit"`
	// MaxPageLimit is the maximum page limit for pagination
	MaxPageLimit uint32 `mapstructure:"MaxPageLimit"`
	// Version is the version of the bridge service
	BridgeVersion string `mapstructure:"BridgeVersion"`
	// DB is the database config
	DB db.Config `mapstructure:"DB"`
}
