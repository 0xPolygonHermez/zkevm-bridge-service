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
		GenBlockNumber:        6829370,
		PoEAddr:               common.HexToAddress("0x083E10Fc0De5a919Dec514CCD9130cD772D38Bfb"),
		BridgeAddr:            common.HexToAddress("0x1E0D0574616274aBbcB7D5b37630e22DaEf52e1f"),
		GlobalExitRootManAddr: common.HexToAddress("0x9e82ec4Db711Cde4fC452948Ab06fBbd7ad34e8f"),
		MaticAddr:             common.HexToAddress("0x7431FD5ba483f826cAf06B68ae95b2aE738D666D"),
		L2BridgeAddrs:         []common.Address{common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")},
		L1ChainID:             5, //Goerli
	}
	localConfig = NetworkConfig{
		Arity:                 4,
		GenBlockNumber:        1,
		PoEAddr:               common.HexToAddress("0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"),
		BridgeAddr:            common.HexToAddress("0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"),
		GlobalExitRootManAddr: common.HexToAddress("0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"),
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
