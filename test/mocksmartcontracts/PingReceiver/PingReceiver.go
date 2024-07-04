// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package PingReceiver

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

// PingReceiverMetaData contains all meta data concerning the PingReceiver contract.
var PingReceiverMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIPolygonZkEVMBridge\",\"name\":\"_polygonZkEVMBridge\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"pingValue\",\"type\":\"uint256\"}],\"name\":\"PingReceived\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"newPingSender\",\"type\":\"address\"}],\"name\":\"SetSender\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"networkID\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"originAddress\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"originNetwork\",\"type\":\"uint32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"name\":\"onMessageReceived\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pingSender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"pingValue\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"polygonZkEVMBridge\",\"outputs\":[{\"internalType\":\"contractIPolygonZkEVMBridge\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newPingSender\",\"type\":\"address\"}],\"name\":\"setSender\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60c060405234801561001057600080fd5b5060405161097b38038061097b83398101604081905261002f91610107565b610038336100b7565b6001600160a01b03811660808190526040805163bab161bf60e01b8152905163bab161bf9160048082019260209290919082900301816000875af1158015610084573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906100a89190610137565b63ffffffff1660a0525061015d565b600080546001600160a01b038381166001600160a01b0319831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b60006020828403121561011957600080fd5b81516001600160a01b038116811461013057600080fd5b9392505050565b60006020828403121561014957600080fd5b815163ffffffff8116811461013057600080fd5b60805160a0516107f3610188600039600061018401526000818160c2015261024001526107f36000f3fe6080604052600436106100965760003560e01c8063ba2b0ff611610069578063ced32b0c1161004e578063ced32b0c146101bb578063f2fde38b146101db578063ffa8d9dc146101fb57600080fd5b8063ba2b0ff61461014e578063bab161bf1461017257600080fd5b80631806b5f21461009b5780635d43792c146100b0578063715018a61461010e5780638da5cb5b14610123575b600080fd5b6100ae6100a9366004610687565b610228565b005b3480156100bc57600080fd5b506100e47f000000000000000000000000000000000000000000000000000000000000000081565b60405173ffffffffffffffffffffffffffffffffffffffff90911681526020015b60405180910390f35b34801561011a57600080fd5b506100ae6103ed565b34801561012f57600080fd5b5060005473ffffffffffffffffffffffffffffffffffffffff166100e4565b34801561015a57600080fd5b5061016460025481565b604051908152602001610105565b34801561017e57600080fd5b506101a67f000000000000000000000000000000000000000000000000000000000000000081565b60405163ffffffff9091168152602001610105565b3480156101c757600080fd5b506100ae6101d6366004610782565b610401565b3480156101e757600080fd5b506100ae6101f6366004610782565b610482565b34801561020757600080fd5b506001546100e49073ffffffffffffffffffffffffffffffffffffffff1681565b3373ffffffffffffffffffffffffffffffffffffffff7f000000000000000000000000000000000000000000000000000000000000000016146102f2576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603760248201527f50696e6752656365697665723a3a6f6e4d65737361676552656365697665643a60448201527f204e6f7420506f6c79676f6e5a6b45564d42726964676500000000000000000060648201526084015b60405180910390fd5b60015473ffffffffffffffffffffffffffffffffffffffff84811691161461039c576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152603060248201527f50696e6752656365697665723a3a6f6e4d65737361676552656365697665643a60448201527f204e6f742070696e672053656e6465720000000000000000000000000000000060648201526084016102e9565b808060200190518101906103b091906107a4565b60028190556040519081527f51c4f05cea43f3d4604f77fd5a656743088090aa726deb5e3a9f670d8da75d659060200160405180910390a1505050565b6103f5610539565b6103ff60006105ba565b565b610409610539565b600180547fffffffffffffffffffffffff00000000000000000000000000000000000000001673ffffffffffffffffffffffffffffffffffffffff83169081179091556040519081527f3f61e183128e8b0775a132afd07eb6b4d74e1e66d7de3c5fbb7a6349b1207b669060200160405180910390a150565b61048a610539565b73ffffffffffffffffffffffffffffffffffffffff811661052d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160448201527f646472657373000000000000000000000000000000000000000000000000000060648201526084016102e9565b610536816105ba565b50565b60005473ffffffffffffffffffffffffffffffffffffffff1633146103ff576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e657260448201526064016102e9565b6000805473ffffffffffffffffffffffffffffffffffffffff8381167fffffffffffffffffffffffff0000000000000000000000000000000000000000831681178455604051919092169283917f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e09190a35050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461065357600080fd5b919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b60008060006060848603121561069c57600080fd5b6106a58461062f565b9250602084013563ffffffff811681146106be57600080fd5b9150604084013567ffffffffffffffff808211156106db57600080fd5b818601915086601f8301126106ef57600080fd5b81358181111561070157610701610658565b604051601f82017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f0116810190838211818310171561074757610747610658565b8160405282815289602084870101111561076057600080fd5b8260208601602083013760006020848301015280955050505050509250925092565b60006020828403121561079457600080fd5b61079d8261062f565b9392505050565b6000602082840312156107b657600080fd5b505191905056fea2646970667358221220f7a6343b3726f685cbf1f60341761a53a0484ae48391492386c86ccaa585ea9964736f6c63430008110033",
}

// PingReceiverABI is the input ABI used to generate the binding from.
// Deprecated: Use PingReceiverMetaData.ABI instead.
var PingReceiverABI = PingReceiverMetaData.ABI

// PingReceiverBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PingReceiverMetaData.Bin instead.
var PingReceiverBin = PingReceiverMetaData.Bin

// DeployPingReceiver deploys a new Ethereum contract, binding an instance of PingReceiver to it.
func DeployPingReceiver(auth *bind.TransactOpts, backend bind.ContractBackend, _polygonZkEVMBridge common.Address) (common.Address, *types.Transaction, *PingReceiver, error) {
	parsed, err := PingReceiverMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PingReceiverBin), backend, _polygonZkEVMBridge)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &PingReceiver{PingReceiverCaller: PingReceiverCaller{contract: contract}, PingReceiverTransactor: PingReceiverTransactor{contract: contract}, PingReceiverFilterer: PingReceiverFilterer{contract: contract}}, nil
}

// PingReceiver is an auto generated Go binding around an Ethereum contract.
type PingReceiver struct {
	PingReceiverCaller     // Read-only binding to the contract
	PingReceiverTransactor // Write-only binding to the contract
	PingReceiverFilterer   // Log filterer for contract events
}

// PingReceiverCaller is an auto generated read-only Go binding around an Ethereum contract.
type PingReceiverCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PingReceiverTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PingReceiverTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PingReceiverFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PingReceiverFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PingReceiverSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PingReceiverSession struct {
	Contract     *PingReceiver     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PingReceiverCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PingReceiverCallerSession struct {
	Contract *PingReceiverCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// PingReceiverTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PingReceiverTransactorSession struct {
	Contract     *PingReceiverTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// PingReceiverRaw is an auto generated low-level Go binding around an Ethereum contract.
type PingReceiverRaw struct {
	Contract *PingReceiver // Generic contract binding to access the raw methods on
}

// PingReceiverCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PingReceiverCallerRaw struct {
	Contract *PingReceiverCaller // Generic read-only contract binding to access the raw methods on
}

// PingReceiverTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PingReceiverTransactorRaw struct {
	Contract *PingReceiverTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPingReceiver creates a new instance of PingReceiver, bound to a specific deployed contract.
func NewPingReceiver(address common.Address, backend bind.ContractBackend) (*PingReceiver, error) {
	contract, err := bindPingReceiver(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PingReceiver{PingReceiverCaller: PingReceiverCaller{contract: contract}, PingReceiverTransactor: PingReceiverTransactor{contract: contract}, PingReceiverFilterer: PingReceiverFilterer{contract: contract}}, nil
}

// NewPingReceiverCaller creates a new read-only instance of PingReceiver, bound to a specific deployed contract.
func NewPingReceiverCaller(address common.Address, caller bind.ContractCaller) (*PingReceiverCaller, error) {
	contract, err := bindPingReceiver(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PingReceiverCaller{contract: contract}, nil
}

// NewPingReceiverTransactor creates a new write-only instance of PingReceiver, bound to a specific deployed contract.
func NewPingReceiverTransactor(address common.Address, transactor bind.ContractTransactor) (*PingReceiverTransactor, error) {
	contract, err := bindPingReceiver(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PingReceiverTransactor{contract: contract}, nil
}

// NewPingReceiverFilterer creates a new log filterer instance of PingReceiver, bound to a specific deployed contract.
func NewPingReceiverFilterer(address common.Address, filterer bind.ContractFilterer) (*PingReceiverFilterer, error) {
	contract, err := bindPingReceiver(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PingReceiverFilterer{contract: contract}, nil
}

// bindPingReceiver binds a generic wrapper to an already deployed contract.
func bindPingReceiver(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PingReceiverMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PingReceiver *PingReceiverRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PingReceiver.Contract.PingReceiverCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PingReceiver *PingReceiverRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PingReceiver.Contract.PingReceiverTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PingReceiver *PingReceiverRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PingReceiver.Contract.PingReceiverTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PingReceiver *PingReceiverCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PingReceiver.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PingReceiver *PingReceiverTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PingReceiver.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PingReceiver *PingReceiverTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PingReceiver.Contract.contract.Transact(opts, method, params...)
}

// NetworkID is a free data retrieval call binding the contract method 0xbab161bf.
//
// Solidity: function networkID() view returns(uint32)
func (_PingReceiver *PingReceiverCaller) NetworkID(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _PingReceiver.contract.Call(opts, &out, "networkID")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// NetworkID is a free data retrieval call binding the contract method 0xbab161bf.
//
// Solidity: function networkID() view returns(uint32)
func (_PingReceiver *PingReceiverSession) NetworkID() (uint32, error) {
	return _PingReceiver.Contract.NetworkID(&_PingReceiver.CallOpts)
}

// NetworkID is a free data retrieval call binding the contract method 0xbab161bf.
//
// Solidity: function networkID() view returns(uint32)
func (_PingReceiver *PingReceiverCallerSession) NetworkID() (uint32, error) {
	return _PingReceiver.Contract.NetworkID(&_PingReceiver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PingReceiver *PingReceiverCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PingReceiver.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PingReceiver *PingReceiverSession) Owner() (common.Address, error) {
	return _PingReceiver.Contract.Owner(&_PingReceiver.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_PingReceiver *PingReceiverCallerSession) Owner() (common.Address, error) {
	return _PingReceiver.Contract.Owner(&_PingReceiver.CallOpts)
}

// PingSender is a free data retrieval call binding the contract method 0xffa8d9dc.
//
// Solidity: function pingSender() view returns(address)
func (_PingReceiver *PingReceiverCaller) PingSender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PingReceiver.contract.Call(opts, &out, "pingSender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PingSender is a free data retrieval call binding the contract method 0xffa8d9dc.
//
// Solidity: function pingSender() view returns(address)
func (_PingReceiver *PingReceiverSession) PingSender() (common.Address, error) {
	return _PingReceiver.Contract.PingSender(&_PingReceiver.CallOpts)
}

// PingSender is a free data retrieval call binding the contract method 0xffa8d9dc.
//
// Solidity: function pingSender() view returns(address)
func (_PingReceiver *PingReceiverCallerSession) PingSender() (common.Address, error) {
	return _PingReceiver.Contract.PingSender(&_PingReceiver.CallOpts)
}

// PingValue is a free data retrieval call binding the contract method 0xba2b0ff6.
//
// Solidity: function pingValue() view returns(uint256)
func (_PingReceiver *PingReceiverCaller) PingValue(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _PingReceiver.contract.Call(opts, &out, "pingValue")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PingValue is a free data retrieval call binding the contract method 0xba2b0ff6.
//
// Solidity: function pingValue() view returns(uint256)
func (_PingReceiver *PingReceiverSession) PingValue() (*big.Int, error) {
	return _PingReceiver.Contract.PingValue(&_PingReceiver.CallOpts)
}

// PingValue is a free data retrieval call binding the contract method 0xba2b0ff6.
//
// Solidity: function pingValue() view returns(uint256)
func (_PingReceiver *PingReceiverCallerSession) PingValue() (*big.Int, error) {
	return _PingReceiver.Contract.PingValue(&_PingReceiver.CallOpts)
}

// PolygonZkEVMBridge is a free data retrieval call binding the contract method 0x5d43792c.
//
// Solidity: function polygonZkEVMBridge() view returns(address)
func (_PingReceiver *PingReceiverCaller) PolygonZkEVMBridge(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _PingReceiver.contract.Call(opts, &out, "polygonZkEVMBridge")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PolygonZkEVMBridge is a free data retrieval call binding the contract method 0x5d43792c.
//
// Solidity: function polygonZkEVMBridge() view returns(address)
func (_PingReceiver *PingReceiverSession) PolygonZkEVMBridge() (common.Address, error) {
	return _PingReceiver.Contract.PolygonZkEVMBridge(&_PingReceiver.CallOpts)
}

// PolygonZkEVMBridge is a free data retrieval call binding the contract method 0x5d43792c.
//
// Solidity: function polygonZkEVMBridge() view returns(address)
func (_PingReceiver *PingReceiverCallerSession) PolygonZkEVMBridge() (common.Address, error) {
	return _PingReceiver.Contract.PolygonZkEVMBridge(&_PingReceiver.CallOpts)
}

// OnMessageReceived is a paid mutator transaction binding the contract method 0x1806b5f2.
//
// Solidity: function onMessageReceived(address originAddress, uint32 originNetwork, bytes data) payable returns()
func (_PingReceiver *PingReceiverTransactor) OnMessageReceived(opts *bind.TransactOpts, originAddress common.Address, originNetwork uint32, data []byte) (*types.Transaction, error) {
	return _PingReceiver.contract.Transact(opts, "onMessageReceived", originAddress, originNetwork, data)
}

// OnMessageReceived is a paid mutator transaction binding the contract method 0x1806b5f2.
//
// Solidity: function onMessageReceived(address originAddress, uint32 originNetwork, bytes data) payable returns()
func (_PingReceiver *PingReceiverSession) OnMessageReceived(originAddress common.Address, originNetwork uint32, data []byte) (*types.Transaction, error) {
	return _PingReceiver.Contract.OnMessageReceived(&_PingReceiver.TransactOpts, originAddress, originNetwork, data)
}

// OnMessageReceived is a paid mutator transaction binding the contract method 0x1806b5f2.
//
// Solidity: function onMessageReceived(address originAddress, uint32 originNetwork, bytes data) payable returns()
func (_PingReceiver *PingReceiverTransactorSession) OnMessageReceived(originAddress common.Address, originNetwork uint32, data []byte) (*types.Transaction, error) {
	return _PingReceiver.Contract.OnMessageReceived(&_PingReceiver.TransactOpts, originAddress, originNetwork, data)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PingReceiver *PingReceiverTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PingReceiver.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PingReceiver *PingReceiverSession) RenounceOwnership() (*types.Transaction, error) {
	return _PingReceiver.Contract.RenounceOwnership(&_PingReceiver.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_PingReceiver *PingReceiverTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _PingReceiver.Contract.RenounceOwnership(&_PingReceiver.TransactOpts)
}

// SetSender is a paid mutator transaction binding the contract method 0xced32b0c.
//
// Solidity: function setSender(address newPingSender) returns()
func (_PingReceiver *PingReceiverTransactor) SetSender(opts *bind.TransactOpts, newPingSender common.Address) (*types.Transaction, error) {
	return _PingReceiver.contract.Transact(opts, "setSender", newPingSender)
}

// SetSender is a paid mutator transaction binding the contract method 0xced32b0c.
//
// Solidity: function setSender(address newPingSender) returns()
func (_PingReceiver *PingReceiverSession) SetSender(newPingSender common.Address) (*types.Transaction, error) {
	return _PingReceiver.Contract.SetSender(&_PingReceiver.TransactOpts, newPingSender)
}

// SetSender is a paid mutator transaction binding the contract method 0xced32b0c.
//
// Solidity: function setSender(address newPingSender) returns()
func (_PingReceiver *PingReceiverTransactorSession) SetSender(newPingSender common.Address) (*types.Transaction, error) {
	return _PingReceiver.Contract.SetSender(&_PingReceiver.TransactOpts, newPingSender)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PingReceiver *PingReceiverTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _PingReceiver.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PingReceiver *PingReceiverSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PingReceiver.Contract.TransferOwnership(&_PingReceiver.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_PingReceiver *PingReceiverTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _PingReceiver.Contract.TransferOwnership(&_PingReceiver.TransactOpts, newOwner)
}

// PingReceiverOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the PingReceiver contract.
type PingReceiverOwnershipTransferredIterator struct {
	Event *PingReceiverOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PingReceiverOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PingReceiverOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PingReceiverOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PingReceiverOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PingReceiverOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PingReceiverOwnershipTransferred represents a OwnershipTransferred event raised by the PingReceiver contract.
type PingReceiverOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PingReceiver *PingReceiverFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*PingReceiverOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PingReceiver.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &PingReceiverOwnershipTransferredIterator{contract: _PingReceiver.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PingReceiver *PingReceiverFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *PingReceiverOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _PingReceiver.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PingReceiverOwnershipTransferred)
				if err := _PingReceiver.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_PingReceiver *PingReceiverFilterer) ParseOwnershipTransferred(log types.Log) (*PingReceiverOwnershipTransferred, error) {
	event := new(PingReceiverOwnershipTransferred)
	if err := _PingReceiver.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PingReceiverPingReceivedIterator is returned from FilterPingReceived and is used to iterate over the raw logs and unpacked data for PingReceived events raised by the PingReceiver contract.
type PingReceiverPingReceivedIterator struct {
	Event *PingReceiverPingReceived // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PingReceiverPingReceivedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PingReceiverPingReceived)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PingReceiverPingReceived)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PingReceiverPingReceivedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PingReceiverPingReceivedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PingReceiverPingReceived represents a PingReceived event raised by the PingReceiver contract.
type PingReceiverPingReceived struct {
	PingValue *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPingReceived is a free log retrieval operation binding the contract event 0x51c4f05cea43f3d4604f77fd5a656743088090aa726deb5e3a9f670d8da75d65.
//
// Solidity: event PingReceived(uint256 pingValue)
func (_PingReceiver *PingReceiverFilterer) FilterPingReceived(opts *bind.FilterOpts) (*PingReceiverPingReceivedIterator, error) {

	logs, sub, err := _PingReceiver.contract.FilterLogs(opts, "PingReceived")
	if err != nil {
		return nil, err
	}
	return &PingReceiverPingReceivedIterator{contract: _PingReceiver.contract, event: "PingReceived", logs: logs, sub: sub}, nil
}

// WatchPingReceived is a free log subscription operation binding the contract event 0x51c4f05cea43f3d4604f77fd5a656743088090aa726deb5e3a9f670d8da75d65.
//
// Solidity: event PingReceived(uint256 pingValue)
func (_PingReceiver *PingReceiverFilterer) WatchPingReceived(opts *bind.WatchOpts, sink chan<- *PingReceiverPingReceived) (event.Subscription, error) {

	logs, sub, err := _PingReceiver.contract.WatchLogs(opts, "PingReceived")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PingReceiverPingReceived)
				if err := _PingReceiver.contract.UnpackLog(event, "PingReceived", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePingReceived is a log parse operation binding the contract event 0x51c4f05cea43f3d4604f77fd5a656743088090aa726deb5e3a9f670d8da75d65.
//
// Solidity: event PingReceived(uint256 pingValue)
func (_PingReceiver *PingReceiverFilterer) ParsePingReceived(log types.Log) (*PingReceiverPingReceived, error) {
	event := new(PingReceiverPingReceived)
	if err := _PingReceiver.contract.UnpackLog(event, "PingReceived", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PingReceiverSetSenderIterator is returned from FilterSetSender and is used to iterate over the raw logs and unpacked data for SetSender events raised by the PingReceiver contract.
type PingReceiverSetSenderIterator struct {
	Event *PingReceiverSetSender // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *PingReceiverSetSenderIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PingReceiverSetSender)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(PingReceiverSetSender)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *PingReceiverSetSenderIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PingReceiverSetSenderIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PingReceiverSetSender represents a SetSender event raised by the PingReceiver contract.
type PingReceiverSetSender struct {
	NewPingSender common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterSetSender is a free log retrieval operation binding the contract event 0x3f61e183128e8b0775a132afd07eb6b4d74e1e66d7de3c5fbb7a6349b1207b66.
//
// Solidity: event SetSender(address newPingSender)
func (_PingReceiver *PingReceiverFilterer) FilterSetSender(opts *bind.FilterOpts) (*PingReceiverSetSenderIterator, error) {

	logs, sub, err := _PingReceiver.contract.FilterLogs(opts, "SetSender")
	if err != nil {
		return nil, err
	}
	return &PingReceiverSetSenderIterator{contract: _PingReceiver.contract, event: "SetSender", logs: logs, sub: sub}, nil
}

// WatchSetSender is a free log subscription operation binding the contract event 0x3f61e183128e8b0775a132afd07eb6b4d74e1e66d7de3c5fbb7a6349b1207b66.
//
// Solidity: event SetSender(address newPingSender)
func (_PingReceiver *PingReceiverFilterer) WatchSetSender(opts *bind.WatchOpts, sink chan<- *PingReceiverSetSender) (event.Subscription, error) {

	logs, sub, err := _PingReceiver.contract.WatchLogs(opts, "SetSender")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PingReceiverSetSender)
				if err := _PingReceiver.contract.UnpackLog(event, "SetSender", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSetSender is a log parse operation binding the contract event 0x3f61e183128e8b0775a132afd07eb6b4d74e1e66d7de3c5fbb7a6349b1207b66.
//
// Solidity: event SetSender(address newPingSender)
func (_PingReceiver *PingReceiverFilterer) ParseSetSender(log types.Log) (*PingReceiverSetSender, error) {
	event := new(PingReceiverSetSender)
	if err := _PingReceiver.contract.UnpackLog(event, "SetSender", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
