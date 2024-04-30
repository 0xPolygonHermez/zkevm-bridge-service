// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package claimcompressor

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

// ClaimcompressorMetaData contains all meta data concerning the Claimcompressor contract.
var ClaimcompressorMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"__bridgeAddress\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"__networkID\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"mainnetExitRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"rollupExitRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bytes32[32]\",\"name\":\"smtProofLocalExitRoot\",\"type\":\"bytes32[32]\"},{\"internalType\":\"bytes32[32]\",\"name\":\"smtProofRollupExitRoot\",\"type\":\"bytes32[32]\"},{\"internalType\":\"uint256\",\"name\":\"globalIndex\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"originNetwork\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"originAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"destinationAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"metadata\",\"type\":\"bytes\"},{\"internalType\":\"bool\",\"name\":\"isMessage\",\"type\":\"bool\"}],\"internalType\":\"structClaimCompressor.CompressClaimCallData[]\",\"name\":\"compressClaimCalldata\",\"type\":\"tuple[]\"}],\"name\":\"compressClaimCall\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"compressedClaimCalls\",\"type\":\"bytes\"}],\"name\":\"sendCompressedClaims\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60c060405234801561001057600080fd5b50604051610e1c380380610e1c83398101604081905261002f9161004b565b6001600160a01b0390911660805263ffffffff1660a05261009a565b6000806040838503121561005e57600080fd5b82516001600160a01b038116811461007557600080fd5b602084015190925063ffffffff8116811461008f57600080fd5b809150509250929050565b60805160a051610d5d6100bf6000396000610442015260006104640152610d5d6000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c806304e5557b1461003b57806397b1410f14610064575b600080fd5b61004e6100493660046105fc565b610079565b60405161005b91906106a3565b60405180910390f35b6100776100723660046106f4565b61043b565b005b606060008585604051602001610099929190918252602082015260400190565b604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe081840301815291905290506000805b8481101561042f5760008686838181106100e9576100e9610766565b90506020028101906100fb9190610795565b61010490610992565b9050600082600003610118575060206101aa565b60005b60208110156101a8578888610131600187610a8d565b81811061014057610140610766565b90506020028101906101529190610795565b816020811061016357610163610766565b60200201358360000151826020811061017e5761017e610766565b60200201511461019657610193816001610aa6565b91505b806101a081610ab9565b91505061011b565b505b6040805160f883901b7fff0000000000000000000000000000000000000000000000000000000000000016602082015281516001818303018152602190910190915260005b828110156102465783518290826020811061020c5761020c610766565b6020020151604051602001610222929190610b0d565b6040516020818303038152906040529150808061023e90610ab9565b9150506101ef565b50602083015151600092501561030a57831580610261575084155b1561026f5760209150610305565b60005b6020811015610303578989610288600188610a8d565b81811061029757610297610766565b90506020028101906102a99190610795565b6104000181602081106102be576102be610766565b6020020135846020015182602081106102d9576102d9610766565b6020020151146102f1576102ee816001610aa6565b92505b806102fb81610ab9565b915050610272565b505b600194505b808260405160200161031d929190610b2f565b604051602081830303815290604052905060005b8281101561038a57818460200151826020811061035057610350610766565b6020020151604051602001610366929190610b0d565b6040516020818303038152906040529150808061038290610ab9565b915050610331565b50600083610100015161039e5760006103a1565b60015b6040858101516060870151608088015160a089015160c08a015160e08b0151805187516103e399988c989081901c979096909590949093909291602001610b79565b60405160208183030381529060405290508681604051602001610407929190610cf8565b604051602081830303815290604052965050505050808061042790610ab9565b9150506100cd565b50909695505050505050565b63ffffffff7f0000000000000000000000000000000000000000000000000000000000000000167f000000000000000000000000000000000000000000000000000000000000000063ccaa2d1163f5efcd798585602082610824376020828101610844376108a486905261092061090452604082015b8183018110156105f157803560f81c80156104d357600181146104f15761050b565b856003538560081c6002538560101c6001538560181c60005361050b565b846003538460081c6002538460101c6001538460181c6000535b5060028101906004906001013560f81c60200280838337909101600181019161040001906020903560f81c02808383379190910190610400810190823560f81c9061041701536001820191506008826018830137600882019150606081019050600482601c830137600482019150602081019050601482600c83013760149182019160408201918390604c01376014820191506020810190506020828237602082013560e01c6040820181905260249092019160609091019080838337600091810182905291820191601f80821660200316016109440190808281808b5af150506104b1565b505050505050505050565b6000806000806060858703121561061257600080fd5b8435935060208501359250604085013567ffffffffffffffff8082111561063857600080fd5b818701915087601f83011261064c57600080fd5b81358181111561065b57600080fd5b8860208260051b850101111561067057600080fd5b95989497505060200194505050565b60005b8381101561069a578181015183820152602001610682565b50506000910152565b60208152600082518060208401526106c281604085016020870161067f565b601f017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0169190910160400192915050565b6000806020838503121561070757600080fd5b823567ffffffffffffffff8082111561071f57600080fd5b818501915085601f83011261073357600080fd5b81358181111561074257600080fd5b86602082850101111561075457600080fd5b60209290920196919550909350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082357ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7218336030181126107c957600080fd5b9190910192915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b604051610120810167ffffffffffffffff81118282101715610826576108266107d3565b60405290565b600082601f83011261083d57600080fd5b60405161040080820182811067ffffffffffffffff82111715610862576108626107d3565b6040528301818582111561087557600080fd5b845b8281101561088f578035825260209182019101610877565b509195945050505050565b803563ffffffff811681146108ae57600080fd5b919050565b803573ffffffffffffffffffffffffffffffffffffffff811681146108ae57600080fd5b600082601f8301126108e857600080fd5b813567ffffffffffffffff80821115610903576109036107d3565b604051601f83017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0908116603f01168101908282118183101715610949576109496107d3565b8160405283815286602085880101111561096257600080fd5b836020870160208301376000602085830101528094505050505092915050565b803580151581146108ae57600080fd5b60006108e082360312156109a557600080fd5b6109ad610802565b6109b7368461082c565b81526109c736610400850161082c565b602082015261080083013560408201526109e4610820840161089a565b60608201526109f661084084016108b3565b6080820152610a0861086084016108b3565b60a082015261088083013560c08201526108a083013567ffffffffffffffff811115610a3357600080fd5b610a3f368286016108d7565b60e083015250610a526108c08401610982565b61010082015292915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b81810381811115610aa057610aa0610a5e565b92915050565b80820180821115610aa057610aa0610a5e565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610aea57610aea610a5e565b5060010190565b60008151610b0381856020860161067f565b9290920192915050565b60008351610b1f81846020880161067f565b9190910191825250602001919050565b60008351610b4181846020880161067f565b60f89390931b7fff00000000000000000000000000000000000000000000000000000000000000169190920190815260010192915050565b7fff000000000000000000000000000000000000000000000000000000000000008b60f81b16815260008a51610bb6816001850160208f0161067f565b7fff000000000000000000000000000000000000000000000000000000000000008b60f81b16600182850101527fffffffffffffffff0000000000000000000000000000000000000000000000008a60c01b16600282850101527fffffffff000000000000000000000000000000000000000000000000000000008960e01b16600a82850101527fffffffffffffffffffffffffffffffffffffffff0000000000000000000000008860601b16600e8285010152610c9d6022828501018860601b7fffffffffffffffffffffffffffffffffffffffff000000000000000000000000169052565b8560368285010152610cd86056828501018660e01b7fffffffff00000000000000000000000000000000000000000000000000000000169052565b610ce7605a8285010185610af1565b9d9c50505050505050505050505050565b60008351610d0a81846020880161067f565b835190830190610d1e81836020880161067f565b0194935050505056fea264697066735822122084bcbf2fb5ac30a76f853974feb3c954180779f186c81a3b6a6cd859e4f5209464736f6c63430008140033",
}

// ClaimcompressorABI is the input ABI used to generate the binding from.
// Deprecated: Use ClaimcompressorMetaData.ABI instead.
var ClaimcompressorABI = ClaimcompressorMetaData.ABI

// ClaimcompressorBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ClaimcompressorMetaData.Bin instead.
var ClaimcompressorBin = ClaimcompressorMetaData.Bin

// DeployClaimcompressor deploys a new Ethereum contract, binding an instance of Claimcompressor to it.
func DeployClaimcompressor(auth *bind.TransactOpts, backend bind.ContractBackend, __bridgeAddress common.Address, __networkID uint32) (common.Address, *types.Transaction, *Claimcompressor, error) {
	parsed, err := ClaimcompressorMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ClaimcompressorBin), backend, __bridgeAddress, __networkID)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Claimcompressor{ClaimcompressorCaller: ClaimcompressorCaller{contract: contract}, ClaimcompressorTransactor: ClaimcompressorTransactor{contract: contract}, ClaimcompressorFilterer: ClaimcompressorFilterer{contract: contract}}, nil
}

// Claimcompressor is an auto generated Go binding around an Ethereum contract.
type Claimcompressor struct {
	ClaimcompressorCaller     // Read-only binding to the contract
	ClaimcompressorTransactor // Write-only binding to the contract
	ClaimcompressorFilterer   // Log filterer for contract events
}

// ClaimcompressorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClaimcompressorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimcompressorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClaimcompressorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimcompressorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClaimcompressorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClaimcompressorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClaimcompressorSession struct {
	Contract     *Claimcompressor  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClaimcompressorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClaimcompressorCallerSession struct {
	Contract *ClaimcompressorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// ClaimcompressorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClaimcompressorTransactorSession struct {
	Contract     *ClaimcompressorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// ClaimcompressorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClaimcompressorRaw struct {
	Contract *Claimcompressor // Generic contract binding to access the raw methods on
}

// ClaimcompressorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClaimcompressorCallerRaw struct {
	Contract *ClaimcompressorCaller // Generic read-only contract binding to access the raw methods on
}

// ClaimcompressorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClaimcompressorTransactorRaw struct {
	Contract *ClaimcompressorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClaimcompressor creates a new instance of Claimcompressor, bound to a specific deployed contract.
func NewClaimcompressor(address common.Address, backend bind.ContractBackend) (*Claimcompressor, error) {
	contract, err := bindClaimcompressor(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Claimcompressor{ClaimcompressorCaller: ClaimcompressorCaller{contract: contract}, ClaimcompressorTransactor: ClaimcompressorTransactor{contract: contract}, ClaimcompressorFilterer: ClaimcompressorFilterer{contract: contract}}, nil
}

// NewClaimcompressorCaller creates a new read-only instance of Claimcompressor, bound to a specific deployed contract.
func NewClaimcompressorCaller(address common.Address, caller bind.ContractCaller) (*ClaimcompressorCaller, error) {
	contract, err := bindClaimcompressor(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClaimcompressorCaller{contract: contract}, nil
}

// NewClaimcompressorTransactor creates a new write-only instance of Claimcompressor, bound to a specific deployed contract.
func NewClaimcompressorTransactor(address common.Address, transactor bind.ContractTransactor) (*ClaimcompressorTransactor, error) {
	contract, err := bindClaimcompressor(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClaimcompressorTransactor{contract: contract}, nil
}

// NewClaimcompressorFilterer creates a new log filterer instance of Claimcompressor, bound to a specific deployed contract.
func NewClaimcompressorFilterer(address common.Address, filterer bind.ContractFilterer) (*ClaimcompressorFilterer, error) {
	contract, err := bindClaimcompressor(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClaimcompressorFilterer{contract: contract}, nil
}

// bindClaimcompressor binds a generic wrapper to an already deployed contract.
func bindClaimcompressor(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClaimcompressorMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Claimcompressor *ClaimcompressorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Claimcompressor.Contract.ClaimcompressorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Claimcompressor *ClaimcompressorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Claimcompressor.Contract.ClaimcompressorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Claimcompressor *ClaimcompressorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Claimcompressor.Contract.ClaimcompressorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Claimcompressor *ClaimcompressorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Claimcompressor.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Claimcompressor *ClaimcompressorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Claimcompressor.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Claimcompressor *ClaimcompressorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Claimcompressor.Contract.contract.Transact(opts, method, params...)
}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_Claimcompressor *ClaimcompressorCaller) CompressClaimCall(opts *bind.CallOpts, mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	var out []interface{}
	err := _Claimcompressor.contract.Call(opts, &out, "compressClaimCall", mainnetExitRoot, rollupExitRoot, compressClaimCalldata)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_Claimcompressor *ClaimcompressorSession) CompressClaimCall(mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	return _Claimcompressor.Contract.CompressClaimCall(&_Claimcompressor.CallOpts, mainnetExitRoot, rollupExitRoot, compressClaimCalldata)
}

// CompressClaimCall is a free data retrieval call binding the contract method 0x04e5557b.
//
// Solidity: function compressClaimCall(bytes32 mainnetExitRoot, bytes32 rollupExitRoot, (bytes32[32],bytes32[32],uint256,uint32,address,address,uint256,bytes,bool)[] compressClaimCalldata) pure returns(bytes)
func (_Claimcompressor *ClaimcompressorCallerSession) CompressClaimCall(mainnetExitRoot [32]byte, rollupExitRoot [32]byte, compressClaimCalldata []ClaimCompressorCompressClaimCallData) ([]byte, error) {
	return _Claimcompressor.Contract.CompressClaimCall(&_Claimcompressor.CallOpts, mainnetExitRoot, rollupExitRoot, compressClaimCalldata)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_Claimcompressor *ClaimcompressorTransactor) SendCompressedClaims(opts *bind.TransactOpts, compressedClaimCalls []byte) (*types.Transaction, error) {
	return _Claimcompressor.contract.Transact(opts, "sendCompressedClaims", compressedClaimCalls)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_Claimcompressor *ClaimcompressorSession) SendCompressedClaims(compressedClaimCalls []byte) (*types.Transaction, error) {
	return _Claimcompressor.Contract.SendCompressedClaims(&_Claimcompressor.TransactOpts, compressedClaimCalls)
}

// SendCompressedClaims is a paid mutator transaction binding the contract method 0x97b1410f.
//
// Solidity: function sendCompressedClaims(bytes compressedClaimCalls) returns()
func (_Claimcompressor *ClaimcompressorTransactorSession) SendCompressedClaims(compressedClaimCalls []byte) (*types.Transaction, error) {
	return _Claimcompressor.Contract.SendCompressedClaims(&_Claimcompressor.TransactOpts, compressedClaimCalls)
}
