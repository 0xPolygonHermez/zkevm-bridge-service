package utils

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUSDCLxLyMapping(t *testing.T) {
	contractAddr1 := common.HexToAddress("0xfe3240995c771f10D2583e8fa95F92ee40E15150")
	contractAddr2 := common.HexToAddress("0x1A8C4999D32F05B63A227517Be0824AeD47e4728")
	contractAddr3 := common.HexToAddress("0xfe3240995c771f10D2583e8fa95F92ee40E15151")
	tokenAddr1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	tokenAddr2 := common.HexToAddress("0x00d69D72a429d4985b34A8E1A6C9e47997F0aFA3")

	InitUSDCLxLyMapping([]common.Address{contractAddr1, contractAddr2}, []common.Address{tokenAddr1, tokenAddr2})

	list := GetUSDCContractAddressList()
	require.Len(t, list, 2)
	require.Contains(t, list, contractAddr1)
	require.Contains(t, list, contractAddr2)

	require.True(t, IsUSDCContractAddress(contractAddr1))
	require.True(t, IsUSDCContractAddress(contractAddr2))
	require.False(t, IsUSDCContractAddress(contractAddr3))

	token, ok := GetUSDCTokenFromContract(contractAddr2)
	require.True(t, ok)
	require.Equal(t, tokenAddr2, token)

	token, ok = GetUSDCTokenFromContract(contractAddr3)
	require.False(t, ok)
}

func TestDecodeUSDCBridgeMetadata(t *testing.T) {
	metadata := common.Hex2Bytes("00000000000000000000000023335657622dcc27bb1914e51cdc30871d6d04d300000000000000000000000000000000000000000000000000000000000f4240")
	addr := common.HexToAddress("0x23335657622dcc27bb1914e51cdc30871d6d04d3")
	amount := new(big.Int).SetUint64(1000000)

	addr1, amount1 := DecodeUSDCBridgeMetadata(metadata)
	require.Equal(t, addr, addr1)
	require.Equal(t, amount, amount1)
}
