// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package erc20permitmock

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

// Erc20permitmockMetaData contains all meta data concerning the Erc20permitmock contract.
var Erc20permitmockMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"address\",\"name\":\"initialAccount\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"initialBalance\",\"type\":\"uint256\"}],\"stateMutability\":\"payable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"EIP712DOMAIN_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"NAME_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PERMIT_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VERSION_HASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"approveInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052604051620018e6380380620018e6833981016040819052620000269162000201565b8383600362000036838262000322565b50600462000045828262000322565b5050506200005a82826200007160201b60201c565b505081516020909201919091206006555062000416565b6001600160a01b038216620000cc5760405162461bcd60e51b815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f206164647265737300604482015260640160405180910390fd5b8060026000828254620000e09190620003ee565b90915550506001600160a01b038216600081815260208181526040808320805486019055518481527fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef910160405180910390a35050565b505050565b634e487b7160e01b600052604160045260246000fd5b600082601f8301126200016457600080fd5b81516001600160401b03808211156200018157620001816200013c565b604051601f8301601f19908116603f01168101908282118183101715620001ac57620001ac6200013c565b81604052838152602092508683858801011115620001c957600080fd5b600091505b83821015620001ed5785820183015181830184015290820190620001ce565b600093810190920192909252949350505050565b600080600080608085870312156200021857600080fd5b84516001600160401b03808211156200023057600080fd5b6200023e8883890162000152565b955060208701519150808211156200025557600080fd5b50620002648782880162000152565b604087015190945090506001600160a01b03811681146200028457600080fd5b6060959095015193969295505050565b600181811c90821680620002a957607f821691505b602082108103620002ca57634e487b7160e01b600052602260045260246000fd5b50919050565b601f8211156200013757600081815260208120601f850160051c81016020861015620002f95750805b601f850160051c820191505b818110156200031a5782815560010162000305565b505050505050565b81516001600160401b038111156200033e576200033e6200013c565b62000356816200034f845462000294565b84620002d0565b602080601f8311600181146200038e5760008415620003755750858301515b600019600386901b1c1916600185901b1785556200031a565b600085815260208120601f198616915b82811015620003bf578886015182559484019460019091019084016200039e565b5085821015620003de5787850151600019600388901b60f8161c191681555b5050505050600190811b01905550565b808201808211156200041057634e487b7160e01b600052601160045260246000fd5b92915050565b6114c080620004266000396000f3fe608060405234801561001057600080fd5b506004361061018d5760003560e01c806340c10f19116100e35780639e4e73181161008c578063c473af3311610066578063c473af33146103d6578063d505accf146103fd578063dd62ed3e1461041057600080fd5b80639e4e731814610389578063a457c2d7146103b0578063a9059cbb146103c357600080fd5b806370a08231116100bd57806370a082311461032b5780637ecebe001461036157806395d89b411461038157600080fd5b806340c10f19146102f257806342966c681461030557806356189cb41461031857600080fd5b806323b872dd116101455780633408e4701161011f5780633408e4701461024c5780633644e5151461025257806339509351146102df57600080fd5b806323b872dd1461020357806330adf81f14610216578063313ce5671461023d57600080fd5b8063095ea7b311610176578063095ea7b3146101c357806318160ddd146101e6578063222f5be0146101ee57600080fd5b806304622c2e1461019257806306fdde03146101ae575b600080fd5b61019b60065481565b6040519081526020015b60405180910390f35b6101b6610456565b6040516101a591906111e1565b6101d66101d1366004611276565b6104e8565b60405190151581526020016101a5565b60025461019b565b6102016101fc3660046112a0565b610502565b005b6101d66102113660046112a0565b610512565b61019b7f6e71edae12b1b97f4d1f60370fef10105fa2faae0126114a169c64845d6126c981565b604051601281526020016101a5565b4661019b565b61019b6006546000907f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f907fc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc646604080516020810195909552840192909252606083015260808201523060a082015260c00160405160208183030381529060405280519060200120905090565b6101d66102ed366004611276565b610536565b610201610300366004611276565b610582565b6102016103133660046112dc565b610590565b6102016103263660046112a0565b61059d565b61019b6103393660046112f5565b73ffffffffffffffffffffffffffffffffffffffff1660009081526020819052604090205490565b61019b61036f3660046112f5565b60056020526000908152604090205481565b6101b66105a8565b61019b7fc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc681565b6101d66103be366004611276565b6105b7565b6101d66103d1366004611276565b61068d565b61019b7f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f81565b61020161040b366004611317565b61069b565b61019b61041e36600461138a565b73ffffffffffffffffffffffffffffffffffffffff918216600090815260016020908152604080832093909416825291909152205490565b606060038054610465906113bd565b80601f0160208091040260200160405190810160405280929190818152602001828054610491906113bd565b80156104de5780601f106104b3576101008083540402835291602001916104de565b820191906000526020600020905b8154815290600101906020018083116104c157829003601f168201915b5050505050905090565b6000336104f68185856107e2565b60019150505b92915050565b61050d838383610995565b505050565b600033610520858285610c06565b61052b858585610995565b506001949350505050565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff871684529091528120549091906104f6908290869061057d90879061143f565b6107e2565b61058c8282610cd7565b5050565b61059a3382610dca565b50565b61050d8383836107e2565b606060048054610465906113bd565b33600081815260016020908152604080832073ffffffffffffffffffffffffffffffffffffffff8716845290915281205490919083811015610680576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f7760448201527f207a65726f00000000000000000000000000000000000000000000000000000060648201526084015b60405180910390fd5b61052b82868684036107e2565b6000336104f6818585610995565b42841015610705576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601960248201527f48455a3a3a7065726d69743a20415554485f45585049524544000000000000006044820152606401610677565b73ffffffffffffffffffffffffffffffffffffffff8716600090815260056020526040812080547f6e71edae12b1b97f4d1f60370fef10105fa2faae0126114a169c64845d6126c9918a918a918a91908661075f83611452565b9091555060408051602081019690965273ffffffffffffffffffffffffffffffffffffffff94851690860152929091166060840152608083015260a082015260c0810186905260e0016040516020818303038152906040528051906020012090506107cd8882868686610f8e565b6107d88888886107e2565b5050505050505050565b73ffffffffffffffffffffffffffffffffffffffff8316610884576040517f08c379a0000000000000000000000000000000000000000000000000000000008152602060048201526024808201527f45524332303a20617070726f76652066726f6d20746865207a65726f2061646460448201527f72657373000000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff8216610927576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a20617070726f766520746f20746865207a65726f20616464726560448201527f73730000000000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff83811660008181526001602090815260408083209487168084529482529182902085905590518481527f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925910160405180910390a3505050565b73ffffffffffffffffffffffffffffffffffffffff8316610a38576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602560248201527f45524332303a207472616e736665722066726f6d20746865207a65726f20616460448201527f64726573730000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff8216610adb576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602360248201527f45524332303a207472616e7366657220746f20746865207a65726f206164647260448201527f65737300000000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff831660009081526020819052604090205481811015610b91576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602660248201527f45524332303a207472616e7366657220616d6f756e742065786365656473206260448201527f616c616e636500000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff848116600081815260208181526040808320878703905593871680835291849020805487019055925185815290927fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef910160405180910390a35b50505050565b73ffffffffffffffffffffffffffffffffffffffff8381166000908152600160209081526040808320938616835292905220547fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8114610c005781811015610cca576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601d60248201527f45524332303a20696e73756666696369656e7420616c6c6f77616e63650000006044820152606401610677565b610c0084848484036107e2565b73ffffffffffffffffffffffffffffffffffffffff8216610d54576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152601f60248201527f45524332303a206d696e7420746f20746865207a65726f2061646472657373006044820152606401610677565b8060026000828254610d66919061143f565b909155505073ffffffffffffffffffffffffffffffffffffffff8216600081815260208181526040808320805486019055518481527fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef910160405180910390a35050565b73ffffffffffffffffffffffffffffffffffffffff8216610e6d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602160248201527f45524332303a206275726e2066726f6d20746865207a65726f2061646472657360448201527f73000000000000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff821660009081526020819052604090205481811015610f23576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602260248201527f45524332303a206275726e20616d6f756e7420657863656564732062616c616e60448201527f63650000000000000000000000000000000000000000000000000000000000006064820152608401610677565b73ffffffffffffffffffffffffffffffffffffffff83166000818152602081815260408083208686039055600280548790039055518581529192917fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef910160405180910390a3505050565b600654604080517f8b73c3c69bb8fe3d512ecc4cf759cc79239f7b179b0ffacaa9a75d522b39400f602080830191909152818301939093527fc89efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc660608201524660808201523060a0808301919091528251808303909101815260c08201909252815191909201207f190100000000000000000000000000000000000000000000000000000000000060e083015260e2820181905261010282018690529060009061012201604080517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe08184030181528282528051602091820120600080855291840180845281905260ff89169284019290925260608301879052608083018690529092509060019060a0016020604051602081039080840390855afa1580156110da573d6000803e3d6000fd5b50506040517fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0015191505073ffffffffffffffffffffffffffffffffffffffff81161580159061115557508773ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16145b6107d8576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820152602b60248201527f48455a3a3a5f76616c69646174655369676e6564446174613a20494e56414c4960448201527f445f5349474e41545552450000000000000000000000000000000000000000006064820152608401610677565b600060208083528351808285015260005b8181101561120e578581018301518582016040015282016111f2565b5060006040828601015260407fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0601f8301168501019250505092915050565b803573ffffffffffffffffffffffffffffffffffffffff8116811461127157600080fd5b919050565b6000806040838503121561128957600080fd5b6112928361124d565b946020939093013593505050565b6000806000606084860312156112b557600080fd5b6112be8461124d565b92506112cc6020850161124d565b9150604084013590509250925092565b6000602082840312156112ee57600080fd5b5035919050565b60006020828403121561130757600080fd5b6113108261124d565b9392505050565b600080600080600080600060e0888a03121561133257600080fd5b61133b8861124d565b96506113496020890161124d565b95506040880135945060608801359350608088013560ff8116811461136d57600080fd5b9699959850939692959460a0840135945060c09093013592915050565b6000806040838503121561139d57600080fd5b6113a68361124d565b91506113b46020840161124d565b90509250929050565b600181811c908216806113d157607f821691505b60208210810361140a577f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b50919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b808201808211156104fc576104fc611410565b60007fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361148357611483611410565b506001019056fea26469706673582212208b36599ef550f0c194373b0c511974c888d8ed7226087368a04dca3579711bcd64736f6c63430008140033",
}

// Erc20permitmockABI is the input ABI used to generate the binding from.
// Deprecated: Use Erc20permitmockMetaData.ABI instead.
var Erc20permitmockABI = Erc20permitmockMetaData.ABI

// Erc20permitmockBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Erc20permitmockMetaData.Bin instead.
var Erc20permitmockBin = Erc20permitmockMetaData.Bin

// DeployErc20permitmock deploys a new Ethereum contract, binding an instance of Erc20permitmock to it.
func DeployErc20permitmock(auth *bind.TransactOpts, backend bind.ContractBackend, name string, symbol string, initialAccount common.Address, initialBalance *big.Int) (common.Address, *types.Transaction, *Erc20permitmock, error) {
	parsed, err := Erc20permitmockMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Erc20permitmockBin), backend, name, symbol, initialAccount, initialBalance)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Erc20permitmock{Erc20permitmockCaller: Erc20permitmockCaller{contract: contract}, Erc20permitmockTransactor: Erc20permitmockTransactor{contract: contract}, Erc20permitmockFilterer: Erc20permitmockFilterer{contract: contract}}, nil
}

// Erc20permitmock is an auto generated Go binding around an Ethereum contract.
type Erc20permitmock struct {
	Erc20permitmockCaller     // Read-only binding to the contract
	Erc20permitmockTransactor // Write-only binding to the contract
	Erc20permitmockFilterer   // Log filterer for contract events
}

// Erc20permitmockCaller is an auto generated read-only Go binding around an Ethereum contract.
type Erc20permitmockCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20permitmockTransactor is an auto generated write-only Go binding around an Ethereum contract.
type Erc20permitmockTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20permitmockFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Erc20permitmockFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Erc20permitmockSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Erc20permitmockSession struct {
	Contract     *Erc20permitmock  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Erc20permitmockCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Erc20permitmockCallerSession struct {
	Contract *Erc20permitmockCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// Erc20permitmockTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Erc20permitmockTransactorSession struct {
	Contract     *Erc20permitmockTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// Erc20permitmockRaw is an auto generated low-level Go binding around an Ethereum contract.
type Erc20permitmockRaw struct {
	Contract *Erc20permitmock // Generic contract binding to access the raw methods on
}

// Erc20permitmockCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Erc20permitmockCallerRaw struct {
	Contract *Erc20permitmockCaller // Generic read-only contract binding to access the raw methods on
}

// Erc20permitmockTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Erc20permitmockTransactorRaw struct {
	Contract *Erc20permitmockTransactor // Generic write-only contract binding to access the raw methods on
}

// NewErc20permitmock creates a new instance of Erc20permitmock, bound to a specific deployed contract.
func NewErc20permitmock(address common.Address, backend bind.ContractBackend) (*Erc20permitmock, error) {
	contract, err := bindErc20permitmock(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmock{Erc20permitmockCaller: Erc20permitmockCaller{contract: contract}, Erc20permitmockTransactor: Erc20permitmockTransactor{contract: contract}, Erc20permitmockFilterer: Erc20permitmockFilterer{contract: contract}}, nil
}

// NewErc20permitmockCaller creates a new read-only instance of Erc20permitmock, bound to a specific deployed contract.
func NewErc20permitmockCaller(address common.Address, caller bind.ContractCaller) (*Erc20permitmockCaller, error) {
	contract, err := bindErc20permitmock(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmockCaller{contract: contract}, nil
}

// NewErc20permitmockTransactor creates a new write-only instance of Erc20permitmock, bound to a specific deployed contract.
func NewErc20permitmockTransactor(address common.Address, transactor bind.ContractTransactor) (*Erc20permitmockTransactor, error) {
	contract, err := bindErc20permitmock(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmockTransactor{contract: contract}, nil
}

// NewErc20permitmockFilterer creates a new log filterer instance of Erc20permitmock, bound to a specific deployed contract.
func NewErc20permitmockFilterer(address common.Address, filterer bind.ContractFilterer) (*Erc20permitmockFilterer, error) {
	contract, err := bindErc20permitmock(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmockFilterer{contract: contract}, nil
}

// bindErc20permitmock binds a generic wrapper to an already deployed contract.
func bindErc20permitmock(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Erc20permitmockMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc20permitmock *Erc20permitmockRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc20permitmock.Contract.Erc20permitmockCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc20permitmock *Erc20permitmockRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Erc20permitmockTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc20permitmock *Erc20permitmockRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Erc20permitmockTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Erc20permitmock *Erc20permitmockCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Erc20permitmock.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Erc20permitmock *Erc20permitmockTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Erc20permitmock *Erc20permitmockTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.contract.Transact(opts, method, params...)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCaller) DOMAINSEPARATOR(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "DOMAIN_SEPARATOR")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Erc20permitmock.Contract.DOMAINSEPARATOR(&_Erc20permitmock.CallOpts)
}

// DOMAINSEPARATOR is a free data retrieval call binding the contract method 0x3644e515.
//
// Solidity: function DOMAIN_SEPARATOR() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCallerSession) DOMAINSEPARATOR() ([32]byte, error) {
	return _Erc20permitmock.Contract.DOMAINSEPARATOR(&_Erc20permitmock.CallOpts)
}

// EIP712DOMAINHASH is a free data retrieval call binding the contract method 0xc473af33.
//
// Solidity: function EIP712DOMAIN_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCaller) EIP712DOMAINHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "EIP712DOMAIN_HASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// EIP712DOMAINHASH is a free data retrieval call binding the contract method 0xc473af33.
//
// Solidity: function EIP712DOMAIN_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockSession) EIP712DOMAINHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.EIP712DOMAINHASH(&_Erc20permitmock.CallOpts)
}

// EIP712DOMAINHASH is a free data retrieval call binding the contract method 0xc473af33.
//
// Solidity: function EIP712DOMAIN_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCallerSession) EIP712DOMAINHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.EIP712DOMAINHASH(&_Erc20permitmock.CallOpts)
}

// NAMEHASH is a free data retrieval call binding the contract method 0x04622c2e.
//
// Solidity: function NAME_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCaller) NAMEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "NAME_HASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// NAMEHASH is a free data retrieval call binding the contract method 0x04622c2e.
//
// Solidity: function NAME_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockSession) NAMEHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.NAMEHASH(&_Erc20permitmock.CallOpts)
}

// NAMEHASH is a free data retrieval call binding the contract method 0x04622c2e.
//
// Solidity: function NAME_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCallerSession) NAMEHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.NAMEHASH(&_Erc20permitmock.CallOpts)
}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCaller) PERMITTYPEHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "PERMIT_TYPEHASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockSession) PERMITTYPEHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.PERMITTYPEHASH(&_Erc20permitmock.CallOpts)
}

// PERMITTYPEHASH is a free data retrieval call binding the contract method 0x30adf81f.
//
// Solidity: function PERMIT_TYPEHASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCallerSession) PERMITTYPEHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.PERMITTYPEHASH(&_Erc20permitmock.CallOpts)
}

// VERSIONHASH is a free data retrieval call binding the contract method 0x9e4e7318.
//
// Solidity: function VERSION_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCaller) VERSIONHASH(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "VERSION_HASH")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// VERSIONHASH is a free data retrieval call binding the contract method 0x9e4e7318.
//
// Solidity: function VERSION_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockSession) VERSIONHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.VERSIONHASH(&_Erc20permitmock.CallOpts)
}

// VERSIONHASH is a free data retrieval call binding the contract method 0x9e4e7318.
//
// Solidity: function VERSION_HASH() view returns(bytes32)
func (_Erc20permitmock *Erc20permitmockCallerSession) VERSIONHASH() ([32]byte, error) {
	return _Erc20permitmock.Contract.VERSIONHASH(&_Erc20permitmock.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.Allowance(&_Erc20permitmock.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.Allowance(&_Erc20permitmock.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.BalanceOf(&_Erc20permitmock.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.BalanceOf(&_Erc20permitmock.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Erc20permitmock *Erc20permitmockCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Erc20permitmock *Erc20permitmockSession) Decimals() (uint8, error) {
	return _Erc20permitmock.Contract.Decimals(&_Erc20permitmock.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Erc20permitmock *Erc20permitmockCallerSession) Decimals() (uint8, error) {
	return _Erc20permitmock.Contract.Decimals(&_Erc20permitmock.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_Erc20permitmock *Erc20permitmockCaller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_Erc20permitmock *Erc20permitmockSession) GetChainId() (*big.Int, error) {
	return _Erc20permitmock.Contract.GetChainId(&_Erc20permitmock.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_Erc20permitmock *Erc20permitmockCallerSession) GetChainId() (*big.Int, error) {
	return _Erc20permitmock.Contract.GetChainId(&_Erc20permitmock.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Erc20permitmock *Erc20permitmockCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Erc20permitmock *Erc20permitmockSession) Name() (string, error) {
	return _Erc20permitmock.Contract.Name(&_Erc20permitmock.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Erc20permitmock *Erc20permitmockCallerSession) Name() (string, error) {
	return _Erc20permitmock.Contract.Name(&_Erc20permitmock.CallOpts)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCaller) Nonces(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "nonces", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockSession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.Nonces(&_Erc20permitmock.CallOpts, arg0)
}

// Nonces is a free data retrieval call binding the contract method 0x7ecebe00.
//
// Solidity: function nonces(address ) view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCallerSession) Nonces(arg0 common.Address) (*big.Int, error) {
	return _Erc20permitmock.Contract.Nonces(&_Erc20permitmock.CallOpts, arg0)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Erc20permitmock *Erc20permitmockCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Erc20permitmock *Erc20permitmockSession) Symbol() (string, error) {
	return _Erc20permitmock.Contract.Symbol(&_Erc20permitmock.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Erc20permitmock *Erc20permitmockCallerSession) Symbol() (string, error) {
	return _Erc20permitmock.Contract.Symbol(&_Erc20permitmock.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Erc20permitmock.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Erc20permitmock *Erc20permitmockSession) TotalSupply() (*big.Int, error) {
	return _Erc20permitmock.Contract.TotalSupply(&_Erc20permitmock.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Erc20permitmock *Erc20permitmockCallerSession) TotalSupply() (*big.Int, error) {
	return _Erc20permitmock.Contract.TotalSupply(&_Erc20permitmock.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Approve(&_Erc20permitmock.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Approve(&_Erc20permitmock.TransactOpts, spender, amount)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockTransactor) ApproveInternal(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "approveInternal", owner, spender, value)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockSession) ApproveInternal(owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.ApproveInternal(&_Erc20permitmock.TransactOpts, owner, spender, value)
}

// ApproveInternal is a paid mutator transaction binding the contract method 0x56189cb4.
//
// Solidity: function approveInternal(address owner, address spender, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockTransactorSession) ApproveInternal(owner common.Address, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.ApproveInternal(&_Erc20permitmock.TransactOpts, owner, spender, value)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockTransactor) Burn(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "burn", amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Burn(&_Erc20permitmock.TransactOpts, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockTransactorSession) Burn(amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Burn(&_Erc20permitmock.TransactOpts, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.DecreaseAllowance(&_Erc20permitmock.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.DecreaseAllowance(&_Erc20permitmock.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.IncreaseAllowance(&_Erc20permitmock.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.IncreaseAllowance(&_Erc20permitmock.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockTransactor) Mint(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "mint", account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockSession) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Mint(&_Erc20permitmock.TransactOpts, account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_Erc20permitmock *Erc20permitmockTransactorSession) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Mint(&_Erc20permitmock.TransactOpts, account, amount)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc20permitmock *Erc20permitmockTransactor) Permit(opts *bind.TransactOpts, owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "permit", owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc20permitmock *Erc20permitmockSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Permit(&_Erc20permitmock.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Permit is a paid mutator transaction binding the contract method 0xd505accf.
//
// Solidity: function permit(address owner, address spender, uint256 value, uint256 deadline, uint8 v, bytes32 r, bytes32 s) returns()
func (_Erc20permitmock *Erc20permitmockTransactorSession) Permit(owner common.Address, spender common.Address, value *big.Int, deadline *big.Int, v uint8, r [32]byte, s [32]byte) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Permit(&_Erc20permitmock.TransactOpts, owner, spender, value, deadline, v, r, s)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactor) Transfer(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "transfer", to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Transfer(&_Erc20permitmock.TransactOpts, to, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactorSession) Transfer(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.Transfer(&_Erc20permitmock.TransactOpts, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "transferFrom", from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.TransferFrom(&_Erc20permitmock.TransactOpts, from, to, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 amount) returns(bool)
func (_Erc20permitmock *Erc20permitmockTransactorSession) TransferFrom(from common.Address, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.TransferFrom(&_Erc20permitmock.TransactOpts, from, to, amount)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockTransactor) TransferInternal(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.contract.Transact(opts, "transferInternal", from, to, value)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockSession) TransferInternal(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.TransferInternal(&_Erc20permitmock.TransactOpts, from, to, value)
}

// TransferInternal is a paid mutator transaction binding the contract method 0x222f5be0.
//
// Solidity: function transferInternal(address from, address to, uint256 value) returns()
func (_Erc20permitmock *Erc20permitmockTransactorSession) TransferInternal(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _Erc20permitmock.Contract.TransferInternal(&_Erc20permitmock.TransactOpts, from, to, value)
}

// Erc20permitmockApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Erc20permitmock contract.
type Erc20permitmockApprovalIterator struct {
	Event *Erc20permitmockApproval // Event containing the contract specifics and raw log

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
func (it *Erc20permitmockApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20permitmockApproval)
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
		it.Event = new(Erc20permitmockApproval)
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
func (it *Erc20permitmockApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20permitmockApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20permitmockApproval represents a Approval event raised by the Erc20permitmock contract.
type Erc20permitmockApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*Erc20permitmockApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Erc20permitmock.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmockApprovalIterator{contract: _Erc20permitmock.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *Erc20permitmockApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Erc20permitmock.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20permitmockApproval)
				if err := _Erc20permitmock.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) ParseApproval(log types.Log) (*Erc20permitmockApproval, error) {
	event := new(Erc20permitmockApproval)
	if err := _Erc20permitmock.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// Erc20permitmockTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Erc20permitmock contract.
type Erc20permitmockTransferIterator struct {
	Event *Erc20permitmockTransfer // Event containing the contract specifics and raw log

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
func (it *Erc20permitmockTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(Erc20permitmockTransfer)
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
		it.Event = new(Erc20permitmockTransfer)
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
func (it *Erc20permitmockTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *Erc20permitmockTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// Erc20permitmockTransfer represents a Transfer event raised by the Erc20permitmock contract.
type Erc20permitmockTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*Erc20permitmockTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc20permitmock.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &Erc20permitmockTransferIterator{contract: _Erc20permitmock.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *Erc20permitmockTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Erc20permitmock.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(Erc20permitmockTransfer)
				if err := _Erc20permitmock.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Erc20permitmock *Erc20permitmockFilterer) ParseTransfer(log types.Log) (*Erc20permitmockTransfer, error) {
	event := new(Erc20permitmockTransfer)
	if err := _Erc20permitmock.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
