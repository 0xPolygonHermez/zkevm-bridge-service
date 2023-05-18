package config

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

// NetworkConfig is the configuration struct for the different environments.
type NetworkConfig struct {
	GenBlockNumber                    uint64
	PolygonZkEVMAddress               common.Address
	PolygonBridgeAddress              common.Address
	PolygonZkEVMGlobalExitRootAddress common.Address
	MaticTokenAddress                 common.Address
	L2PolygonBridgeAddresses          []common.Address
	L1ChainID                         uint64
}

const (
	defaultNetwork = "mainnet"
)

//nolint:gomnd
var (
	networkConfigs = map[string]NetworkConfig{
		defaultNetwork: {
			GenBlockNumber:                    16896718,
			PolygonZkEVMAddress:               common.HexToAddress("0x5132A183E9F3CB7C848b0AAC5Ae0c4f0491B7aB2"),
			PolygonBridgeAddress:              common.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x580bda1e7A0CFAe92Fa7F6c20A3794F169CE3CFb"),
			MaticTokenAddress:                 common.HexToAddress("0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe")},
			L1ChainID:                         1, //Mainnet
		},

		"testnet": {
			GenBlockNumber:                    8572995,
			PolygonZkEVMAddress:               common.HexToAddress("0xa997cfD539E703921fD1e3Cf25b4c241a27a4c7A"),
			PolygonBridgeAddress:              common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x4d9427DCA0406358445bC0a8F88C26b704004f74"),
			MaticTokenAddress:                 common.HexToAddress("0x1319D23c2F7034F52Eb07399702B040bA278Ca49"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xF6BEEeBB578e214CA9E23B0e9683454Ff88Ed2A7")},
			L1ChainID:                         5, //Goerli
		},
		"internaltestnet": {
			GenBlockNumber:                    7674349,
			PolygonZkEVMAddress:               common.HexToAddress("0x159113e5560c9CC2d8c4e716228CCf92c72E9603"),
			PolygonBridgeAddress:              common.HexToAddress("0x47c1090bc966280000Fe4356a501f1D0887Ce840"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xA379Dd55Eb12e8FCdb467A814A15DE2b29677066"),
			MaticTokenAddress:                 common.HexToAddress("0x94Ca2BbE1b469f25D3B22BDf17Fc80ad09E7F662"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xfC5b0c5F677a3f3E29DB2e98c9eD455c7ACfCf03")},
			L1ChainID:                         5, //Goerli
		},
		"local": {
			GenBlockNumber:                    1,
			PolygonZkEVMAddress:               common.HexToAddress("0x610178dA211FEF7D417bC0e6FeD39F05609AD788"),
			PolygonBridgeAddress:              common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0"),
			PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"),
			MaticTokenAddress:                 common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
			L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0")},
			L1ChainID:                         1337,
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
