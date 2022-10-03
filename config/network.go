package config

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

//NetworkConfig is the configuration struct for the different environments
type NetworkConfig struct {
	Arity                 uint8
	GenBlockNumber        uint64
	PoEAddr               common.Address
	BridgeAddr            common.Address
	GlobalExitRootManAddr common.Address
	MaticAddr             common.Address
	L2BridgeAddrs         []common.Address
	L1ChainID             uint64
}

const (
	testnet         = "testnet"
	internalTestnet = "internaltestnet"
	local           = "local"
)

//nolint:gomnd
var (
	mainnetConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        13808430,
		PoEAddr:               common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		BridgeAddr:            common.HexToAddress("0x11D0Dc8E2Ce3a93EB2b32f4C7c3fD9dDAf1211FA"),
		GlobalExitRootManAddr: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaticAddr:             common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:             1, //Mainnet
	}
	testnetConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        9817974,
		PoEAddr:               common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		BridgeAddr:            common.HexToAddress("0x21D0Dc8E2Ce3a93EB2b32f4C7c3fD9dDAf1211FA"),
		GlobalExitRootManAddr: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaticAddr:             common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:             4, //Rinkeby
	}
	internalTestnetConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        7674349,
		PoEAddr:               common.HexToAddress("0x159113e5560c9CC2d8c4e716228CCf92c72E9603"),
		BridgeAddr:            common.HexToAddress("0x47c1090bc966280000Fe4356a501f1D0887Ce840"),
		GlobalExitRootManAddr: common.HexToAddress("0xA379Dd55Eb12e8FCdb467A814A15DE2b29677066"),
		MaticAddr:             common.HexToAddress("0x94Ca2BbE1b469f25D3B22BDf17Fc80ad09E7F662"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")},
		L1ChainID:             5, //Goerli
	}
	localConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        1,
		PoEAddr:               common.HexToAddress("0x2279B7A0a67DB372996a5FaB50D91eAA73d2eBe6"),
		BridgeAddr:            common.HexToAddress("0x0165878A594ca255338adfa4d48449f69242Eb8F"),
		GlobalExitRootManAddr: common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		MaticAddr:             common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")},
		L1ChainID:             1337,
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
