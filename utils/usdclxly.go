package utils

import (
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/common"
)

var (
	usdcContractToTokenMapping map[common.Address]common.Address
	emptyAddress               = common.Address{}
)

func InitUSDCLxLyMapping(usdcContractAddresses, usdcTokenAddresses []common.Address) {
	log.Debugf("USDCLxLyMapping: contracts[%v] tokens[%v]", usdcContractAddresses, usdcTokenAddresses)
	if len(usdcContractAddresses) != len(usdcTokenAddresses) {
		log.Errorf("InitUSDCLxLyMapping: contract addresses (%v) and token addresses (%v) have different length", len(usdcContractAddresses), len(usdcTokenAddresses))
	}

	usdcContractToTokenMapping = make(map[common.Address]common.Address)
	l := min(len(usdcContractAddresses), len(usdcTokenAddresses))
	for i := 0; i < l; i++ {
		if usdcTokenAddresses[i] == emptyAddress {
			continue
		}
		usdcContractToTokenMapping[usdcContractAddresses[i]] = usdcTokenAddresses[i]
	}
}

func GetUSDCContractAddressList() []common.Address {
	result := make([]common.Address, 0)
	for addr := range usdcContractToTokenMapping {
		result = append(result, addr)
	}
	return result
}

func IsUSDCContractAddress(address common.Address) bool {
	if _, ok := usdcContractToTokenMapping[address]; ok {
		return true
	}
	return false
}

func GetUSDCTokenFromContract(contractAddress common.Address) (common.Address, bool) {
	if token, ok := usdcContractToTokenMapping[contractAddress]; ok {
		return token, true
	}
	return common.Address{}, false
}

// DecodeUSDCBridgeMetadata extracts the user's account address from the metadata of USDC bridge
// Metadata structure:
// - Destination address: 32 bytes
// - Bridging amount: 32 bytes
func DecodeUSDCBridgeMetadata(metadata []byte) (common.Address, *big.Int) {
	// Convert the first 32 bytes to address, and last 32 bytes to big int
	// Maybe there's a more elegant way?
	return common.BytesToAddress(metadata[:32]), new(big.Int).SetBytes(metadata[32:]) //nolint:gomnd
}

func ReplaceUSDCDepositInfo(deposit *etherman.Deposit) {
	token, ok := GetUSDCTokenFromContract(deposit.OriginalAddress)
	if !ok {
		return
	}
	deposit.OriginalAddress = token
	_, deposit.Amount = DecodeUSDCBridgeMetadata(deposit.Metadata)
}
