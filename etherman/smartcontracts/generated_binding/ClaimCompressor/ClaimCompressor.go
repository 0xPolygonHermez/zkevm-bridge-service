// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ClaimCompressor

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// ClaimCompressorCompressClaimCallData is an auto generated low-level Go binding around an user-defined struct.
type ClaimCompressorCompressClaimCallData struct {
	SmtProofLocalExitRoot  [32][32]byte
	SmtProofRollupExitRoot [32][32]byte
	GlobalIndex            *big.Int
	OriginNetwork          uint32
	OriginAddress          common.Address
	DestinationAddress     common.Address
	Amount                 *big.Int
	Metadata               []byte
	IsMessage              bool
}

// ClaimCompressorMetaData contains all meta data concerning the ClaimCompressor contract.
var ClaimCompressorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"__bridgeAddress\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"__networkID\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"mainnetExitRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"rollupExitRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32[32]\",\"name\":\"smtProofLocalExitRoot\",\"type\":\"bytes32[32]\"},{\"internalType\":\"bytes32[32]\",\"name\":\"smtProofRollupExitRoot\",\"type\":\"bytes32[32]\"},{\"internalType\":\"uint256\",\"name\":\"globalIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"originNetwork\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"originAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"destinationAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"isMessage\",\"type\":\"bool\"}],\"internalType\":\"structClaimCompressor.CompressClaimCallData[]\",\"name\":\"compressClaimCalldata\",\"type\":\"tuple[]\"}],\"name\":\"compressClaimCall\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"compressedClaimCalls\",\"type\":\"bytes\"}],\"name\":\"sendCompressedClaims\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// ClaimCompressorABI is the input ABI used to generate the binding from.
// Deprecated: Use ClaimCompressorMetaData.ABI instead.
var ClaimCompressorABI = ClaimCompressorMetaData.ABI

// ClaimCompressor is an auto generated Go binding around an Ethereum contract.
type ClaimCompressor struct {
	ClaimCompressorCaller     // Read-only binding to the contract
	ClaimCompressorTransactor // Write-only binding to the contract
	ClaimCompressorFilterer   // Log filterer for contract events
}

// ClaimCompressorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClaimCompressorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimCompressorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClaimCompressorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimCompressorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClaimCompressorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimCompressorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClaimCompressorSession struct {
	Contract     *ClaimCompressor  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClaimCompressorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClaimCompressorCallerSession struct {
	Contract *ClaimCompressorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ClaimCompressorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClaimCompressorTransactorSession struct {
	Contract     *ClaimCompressorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ClaimCompressorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClaimCompressorRaw struct {
	Contract *ClaimCompressor // Generic contract binding to access the raw methods on
}

// ClaimCompressorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClaimCompressorCallerRaw struct {
	Contract *ClaimCompressorCaller // Generic read-only contract binding to access the raw methods on
}

// ClaimCompressorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClaimCompressorTransactorRaw struct {
	Contract *ClaimCompressorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClaimCompressor creates a new instance of ClaimCompressor, bound to a specific deployed contract.
func NewClaimCompressor(address common.Address, backend bind.ContractBackend) (*ClaimCompressor, error) {
	contract, err := bindClaimCompressor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ClaimCompressor{ClaimCompressorCaller: ClaimCompressorCaller{contract: contract}, ClaimCompressorTransactor: ClaimCompressorTransactor{contract: contract}, ClaimCompressorFilterer: ClaimCompressorFilterer{contract: contract}}, nil
}

// NewClaimCompressorCaller creates a new read-only instance of ClaimCompressor, bound to a specific deployed contract.
func NewClaimCompressorCaller(address common.Address, caller bind.ContractCaller) (*ClaimCompressorCaller, error) {
	contract, err := bindClaimCompressor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClaimCompressorCaller{contract: contract}, nil
}

// NewClaimCompressorTransactor creates a new write-only instance of ClaimCompressor, bound to a specific deployed contract.
func NewClaimCompressorTransactor(address common.Address, transactor bind.ContractTransactor) (*ClaimCompressorTransactor, error) {
	contract, err := bindClaimCompressor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClaimCompressorTransactor{contract: contract}, nil
}

// NewClaimCompressorFilterer creates a new log filterer instance of ClaimCompressor, bound to a specific deployed contract.
func NewClaimCompressorFilterer(address common.Address, filterer bind.ContractFilterer) (*ClaimCompressorFilterer, error) {
	contract, err := bindClaimCompressor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClaimCompressorFilterer{contract: contract}, nil
}

// bindClaimCompressor binds a generic wrapper to an already deployed contract.
func bindClaimCompressor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClaimCompressorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClaimCompressor *ClaimCompressorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClaimCompressor.Contract.ClaimCompressorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClaimCompressor *ClaimCompressorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.ClaimCompressorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClaimCompressor *ClaimCompressorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.ClaimCompressorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClaimCompressor *ClaimCompressorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClaimCompressor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClaimCompressor *ClaimCompressorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClaimCompressor *ClaimCompressorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.contract.Transact(opts, method, params...)
}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_ClaimCompressor *ClaimCompressorCaller) CompressClaimCall(opts *bind.CallOpts, mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	var out []interface{}
	err := _ClaimCompressor.contract.Call(opts, &out, "compressClaimCall", mainnetExitRoot, rollupExitRoot, compressClaimCalldata)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_ClaimCompressor *ClaimCompressorSession) CompressClaimCall(mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	return _ClaimCompressor.Contract.CompressClaimCall(&_ClaimCompressor.CallOpts, mainnetExitRoot, rollupExitRoot, compressClaimCalldata)
}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_ClaimCompressor *ClaimCompressorCallerSession) CompressClaimCall(mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	return _ClaimCompressor.Contract.CompressClaimCall(&_ClaimCompressor.CallOpts, mainnetExitRoot, rollupExitRoot, compressClaimCalldata)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_ClaimCompressor *ClaimCompressorTransactor) SendCompressedClaims(opts *bind.TransactOpts, compressedClaimCalls []byte) (*types.Transaction, error) {
	return _ClaimCompressor.contract.Transact(opts, "sendCompressedClaims", compressedClaimCalls)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_ClaimCompressor *ClaimCompressorSession) SendCompressedClaims(compressedClaimCalls []byte) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.SendCompressedClaims(&_ClaimCompressor.TransactOpts, compressedClaimCalls)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_ClaimCompressor *ClaimCompressorTransactorSession) SendCompressedClaims(compressedClaimCalls []byte) (*types.Transaction, error) {
	return _ClaimCompressor.Contract.SendCompressedClaims(&_ClaimCompressor.TransactOpts, compressedClaimCalls)
}
