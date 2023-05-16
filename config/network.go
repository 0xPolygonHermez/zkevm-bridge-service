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
	testnet         = "testnet"
	internalTestnet = "internaltestnet"
	local           = "local"
)

//nolint:gomnd
var (
	mainnetConfig = NetworkConfig{
		GenBlockNumber:                    13808430,
		PolygonZkEVMAddress:               common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		PolygonBridgeAddress:              common.HexToAddress("0x11D0Dc8E2Ce3a93EB2b32f4C7c3fD9dDAf1211FA"),
		PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaticTokenAddress:                 common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:                         1, //Mainnet
	}
	testnetConfig = NetworkConfig{
		GenBlockNumber:                    9817974,
		PolygonZkEVMAddress:               common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		PolygonBridgeAddress:              common.HexToAddress("0x21D0Dc8E2Ce3a93EB2b32f4C7c3fD9dDAf1211FA"),
		PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaticTokenAddress:                 common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:                         4, //Rinkeby
	}
	internalTestnetConfig = NetworkConfig{
		GenBlockNumber:                    7674349,
		PolygonZkEVMAddress:               common.HexToAddress("0x159113e5560c9CC2d8c4e716228CCf92c72E9603"),
		PolygonBridgeAddress:              common.HexToAddress("0x47c1090bc966280000Fe4356a501f1D0887Ce840"),
		PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0xA379Dd55Eb12e8FCdb467A814A15DE2b29677066"),
		MaticTokenAddress:                 common.HexToAddress("0x94Ca2BbE1b469f25D3B22BDf17Fc80ad09E7F662"),
		L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xfC5b0c5F677a3f3E29DB2e98c9eD455c7ACfCf03")},
		L1ChainID:                         5, //Goerli
	}
	localConfig = NetworkConfig{
		GenBlockNumber:                    1,
		PolygonZkEVMAddress:               common.HexToAddress("0x610178dA211FEF7D417bC0e6FeD39F05609AD788"),
		PolygonBridgeAddress:              common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0"),
		PolygonZkEVMGlobalExitRootAddress: common.HexToAddress("0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"),
		MaticTokenAddress:                 common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
		L2PolygonBridgeAddresses:          []common.Address{common.HexToAddress("0xff0EE8ea08cEf5cb4322777F5CC3E8A584B8A4A0")},
		L1ChainID:                         1337,
	}
)

func (cfg *Config) loadNetworkConfig(network string) {
	switch network {
	case testnet:
		log.Debug("Testnet network selected")
		cfg.NetworkConfig = testnetConfig
	case internalTestnet:
		log.Debug("InternalTestnet network selected")
		cfg.NetworkConfig = internalTestnetConfig
	case local:
		log.Debug("Local network selected")
		cfg.NetworkConfig = localConfig
	default:
		log.Debug("Mainnet network selected")
		cfg.NetworkConfig = mainnetConfig
	}
}
