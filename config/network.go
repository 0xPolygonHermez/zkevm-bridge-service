package config

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-core/log"
)

//NetworkConfig is the configuration struct for the different environments
type NetworkConfig struct {
	Arity                 uint8
	GenBlockNumber        uint64
	PoEAddr               common.Address
	BridgeAddr            common.Address
	GlobalExitRootManAddr common.Address
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
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:             1, //Mainnet
	}
	testnetConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        9817974,
		PoEAddr:               common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		BridgeAddr:            common.HexToAddress("0x21D0Dc8E2Ce3a93EB2b32f4C7c3fD9dDAf1211FA"),
		GlobalExitRootManAddr: common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")},
		L1ChainID:             4, //Rinkeby
	}
	internalTestnetConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        6262429,
		PoEAddr:               common.HexToAddress("0x9a5a6dfD995F53676e41Ea0ec6d77b817C58513E"),
		BridgeAddr:            common.HexToAddress("0x7eDc3cd854f0A5FE720Ebb42e0B39a2F33c2ec37"),
		GlobalExitRootManAddr: common.HexToAddress("0x20B5C093d354b39348eE384dC77eD698Eabbf89f"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")},
		L1ChainID:             5, //Goerli
	}
	localConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        1,
		PoEAddr:               common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BridgeAddr:            common.HexToAddress("0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"),
		GlobalExitRootManAddr: common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"),
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
