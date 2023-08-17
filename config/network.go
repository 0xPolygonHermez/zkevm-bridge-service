package config

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

// NetworkConfig is the configuration struct for the different environments.
type NetworkConfig struct {
	GenBlockNumber                    uint64
	PolygonBridgeAddress              common.Address
	PolygonZkEVMGlobalExitRootAddress common.Address
	L2PolygonBridgeAddresses          []common.Address
}

const (
	defaultNetwork = "mainnet"
)

//nolint:gomnd
var (
	networkConfigs = map[string]NetworkConfig{
		defaultNetwork: {
			GenBlockNumber:                    16896718,
			PolygonBridgeAddress:              common.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x580bda1e7A0CFAe92Fa7F6c20A3794F169CE3CFb"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe")},
		},

		"testnet": {
			GenBlockNumber:                    8572995,
			PolygonBridgeAddress:              common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x4d9427DCA0406358445bC0a8F88C26b704004f74"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7")},
		},
		"internaltestnet": {
			GenBlockNumber:                    7674349,
			PolygonBridgeAddress:              common.HexToAddress("0x47c1090bc966280000Fe4356a501f1D0887Ce840"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xA379Dd55Eb12e8FCdb467A814A15DE2b29677066"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xfC5b0c5F677a3f3E29DB2e98c9eD455c7ACfCf03")},
		},
		"local": {
			GenBlockNumber:                    1,
			PolygonBridgeAddress:              common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0")},
		},
	}
)

func (cfg *Config) loadNetworkConfig(network string) {
	networkConfig, valid := networkConfigs[network]
	if valid {
		log.Debugf("Network '%v' selected", network)
		cfg.NetworkConfig = networkConfig
	} else {
		log.Debugf("Network '%v' is invalid. Selecting %v instead.", network, defaultNetwork)
		cfg.NetworkConfig = networkConfigs[defaultNetwork]
	}
}
