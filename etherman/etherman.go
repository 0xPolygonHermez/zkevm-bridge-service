package etherman

import (
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-bridge/log"
)

var (
	depositEventSignatureHash              = crypto.Keccak256Hash([]byte("DepositEvent(address,uint256,uint32,address,uint32)"))
	updateGlobalExitRootEventSignatureHash = crypto.Keccak256Hash([]byte("UpdateGlobalExitRoot(bytes32,bytes32)"))
	claimEventSignatureHash                = crypto.Keccak256Hash([]byte("WithdrawEvent(uint64,uint32,address,uint256,address)"))

	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

type ethClienter interface {
	ethereum.ChainReader
	ethereum.LogFilterer
	ethereum.TransactionReader
}

type ClientEtherMan struct {
	EtherClient ethClienter
	Bridge      *bridge.Bridge
	SCAddresses []common.Address
}

type EtherMan interface {
}

// NewEtherman creates a new etherman
func NewEtherman(cfg Config, bridgeAddr common.Address) (*ClientEtherMan, error) {
	// TODO: PoEAddr can be got from bridge smc. Son only bridge smc is required
	// Connect to ethereum node
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.URL, err)
		return nil, err
	}
	// Create smc clients
	bridge, err := bridge.NewBridge(bridgeAddr, ethClient)
	if err != nil {
		return nil, err
	}
	var scAddresses []common.Address
	scAddresses = append(scAddresses, bridgeAddr)

	return &ClientEtherMan{EtherClient: ethClient, Bridge: bridge, SCAddresses: scAddresses}, nil
}
