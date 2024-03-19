package txcompressor

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	ctmtypes "github.com/0xPolygonHermez/zkevm-bridge-service/claimtxman/types"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/generated_binding/ClaimCompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/polygonzkevmbridge"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrMethodUnknown  = errors.New("method unknown")
)

type CompressClaimParameters struct {
	MainnetExitRoot common.Hash
	RollupExitRoot  common.Hash
	ClaimDatas      []ClaimCompressor.ClaimCompressorCompressClaimCallData
}

// bridgeClaimXParams is a struct to hold the parameters for the bridge claimAsset and claimMessage methods
type bridgeClaimXParams struct {
	smtProofLocalExitRoot  [32][32]byte
	smtProofRollupExitRoot [32][32]byte
	globalIndex            *big.Int
	mainnetExitRoot        [32]byte
	rollupExitRoot         [32]byte
	originNetwork          uint32
	originTokenAddress     common.Address
	destinationNetwork     uint32
	destinationAddress     common.Address
	amount                 *big.Int
	metadata               []byte
	isMessage              bool
}

type ComposeCompressClaim struct {
	smcAbi              abi.ABI
	methodClaimAssets   abi.Method
	methodClaimMessages abi.Method
}

func NewComposeCompressClaim() (*ComposeCompressClaim, error) {
	smcAbi, err := abi.JSON(strings.NewReader(polygonzkevmbridge.PolygonzkevmbridgeABI))
	if err != nil {
		return nil, errors.New("fails to read abi fom Bridge contract")
	}
	methodClaimAssets, ok := smcAbi.Methods["claimAsset"]
	if !ok {
		return nil, errors.New("method claimAssets not found")
	}
	methodClaimMessages, ok := smcAbi.Methods["claimMessage"]
	if !ok {
		return nil, errors.New("method claimMessages not found")
	}
	return &ComposeCompressClaim{
		smcAbi:              smcAbi,
		methodClaimAssets:   methodClaimAssets,
		methodClaimMessages: methodClaimMessages,
	}, nil
}

func (c *ComposeCompressClaim) GetCompressClaimParametersFromMonitoredTx(txs []ctmtypes.MonitoredTx) (*CompressClaimParameters, error) {
	result := make(map[uint64][]byte, len(txs))
	for i := range txs {
		result[uint64(txs[i].DepositID)] = txs[i].Data
	}
	return c.GetCompressClaimParameters(result)
}

// GetCompressClaimParameters returns the parameters for the compressClaim method
// txData = map [depositID] = txData
func (c *ComposeCompressClaim) GetCompressClaimParameters(txsData map[uint64][]byte) (*CompressClaimParameters, error) {
	params := make(map[uint64]bridgeClaimXParams)
	for depositID := range txsData {
		data := txsData[depositID]
		txParams, err := c.extractParams(data)
		if err != nil {
			return nil, err
		}
		params[uint64(depositID)] = *txParams
	}
	maxDepositID := slices.Max(maps.Keys(params))
	result := &CompressClaimParameters{
		MainnetExitRoot: params[maxDepositID].mainnetExitRoot,
		RollupExitRoot:  params[maxDepositID].rollupExitRoot,
	}
	for _, v := range params {
		claimData := ClaimCompressor.ClaimCompressorCompressClaimCallData{
			SmtProofLocalExitRoot:  v.smtProofLocalExitRoot,
			SmtProofRollupExitRoot: v.smtProofRollupExitRoot,
			GlobalIndex:            v.globalIndex,
			OriginNetwork:          v.originNetwork,
			OriginAddress:          v.originTokenAddress,
			// TODO: destinationNetwork is missing?
			//DestinationNetwork:     v.destinationNetwork,
			DestinationAddress: v.destinationAddress,
			Amount:             v.amount,
			Metadata:           v.metadata,
			IsMessage:          v.isMessage,
		}
		result.ClaimDatas = append(result.ClaimDatas, claimData)
	}
	return result, nil
}

func (c *ComposeCompressClaim) extractParams(data []byte) (*bridgeClaimXParams, error) {
	isMessage := false
	if bytes.Equal(c.methodClaimAssets.ID, data[0:4]) {
		isMessage = false
	} else if bytes.Equal(c.methodClaimMessages.ID, data[0:4]) {
		isMessage = true
	} else {
		return nil, ErrMethodUnknown
	}
	params, err := c.extractParamsClaimX(data)
	if err != nil {
		return nil, err
	}
	params.isMessage = isMessage
	return params, nil
}
func (c *ComposeCompressClaim) extractParamsClaimX(data []byte) (*bridgeClaimXParams, error) {
	// do something
	method, err := c.smcAbi.MethodById(data[:4])
	if err != nil {
		return nil, fmt.Errorf("extracting params, getting method err: %w ", err)
	}

	// Unpack method inputs
	params, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, fmt.Errorf("extracting params, Unpack err: %w ", err)
	}
	result := &bridgeClaimXParams{}
	// Solidity: function claimAsset(bytes32[32] smtProofLocalExitRoot, bytes32[32] smtProofRollupExitRoot, uint256 globalIndex, bytes32 mainnetExitRoot, bytes32 rollupExitRoot, uint32 originNetwork, address originTokenAddress, uint32 destinationNetwork, address destinationAddress, uint256 amount, bytes metadata) returns()
	result.smtProofLocalExitRoot = params[0].([32][32]byte)
	result.smtProofRollupExitRoot = params[1].([32][32]byte)
	result.globalIndex = params[2].(*big.Int)
	result.mainnetExitRoot = params[3].([32]byte)
	result.rollupExitRoot = params[4].([32]byte)
	result.originNetwork = params[5].(uint32)
	result.originTokenAddress = params[6].(common.Address)
	result.destinationNetwork = params[7].(uint32)
	result.destinationAddress = params[8].(common.Address)
	result.amount = params[9].(*big.Int)
	result.metadata = params[10].([]byte)
	return result, nil
}
