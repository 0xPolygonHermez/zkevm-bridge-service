package config

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/common"
)

// NetworkConfig is the configuration struct for the different environments.
type NetworkConfig struct {
	GenBlockNumber                    uint64
	PolygonBridgeAddress              common.Address
	PolygonZkEVMGlobalExitRootAddress common.Address
	PolygonRollupManagerAddress       common.Address
	PolygonZkEvmAddress               common.Address
	ClaimCompressorAdress             common.Address
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
			PolygonRollupManagerAddress:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			PolygonZkEvmAddress:               common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe")},
		},

		"testnet": {
			GenBlockNumber:                    8572995,
			PolygonBridgeAddress:              common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x4d9427DCA0406358445bC0a8F88C26b704004f74"),
			PolygonRollupManagerAddress:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			PolygonZkEvmAddress:               common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7")},
		},
		"internaltestnet": {
			GenBlockNumber:                    7674349,
			PolygonBridgeAddress:              common.HexToAddress("0x47c1090bc966280000Fe4356a501f1D0887Ce840"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xA379Dd55Eb12e8FCdb467A814A15DE2b29677066"),
			PolygonRollupManagerAddress:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			PolygonZkEvmAddress:               common.HexToAddress("0x0000000000000000000000000000000000000000"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xfC5b0c5F677a3f3E29DB2e98c9eD455c7ACfCf03")},
		},
		"local": {
			GenBlockNumber:                    1,
			PolygonBridgeAddress:              common.HexToAddress("0xCca6ECD73932e49633B9307e1aa0fC174525F424"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x8A791620dd6260079BF849Dc5567aDC3F2FdC318"),
			PolygonRollupManagerAddress:       common.HexToAddress("0xB7f8BC63BbcaD18155201308C8f3540b07f84F5e"),
			PolygonZkEvmAddress:               common.HexToAddress("0x8dAF17A20c9DBA35f005b6324F493785D239719d"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xCca6ECD73932e49633B9307e1aa0fC174525F424")},
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
