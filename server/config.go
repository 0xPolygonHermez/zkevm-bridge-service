package server

// Config struct
type Config struct {
	// GRPCPort is TCP port to listen by gRPC server
	GRPCPort string
	// HTTPPort is TCP port to listen by HTTP/REST gateway
	HTTPPort string
}
