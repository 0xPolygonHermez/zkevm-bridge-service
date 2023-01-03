package utils

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/BridgeMessageReceiver"
	"github.com/0xPolygonHermez/zkevm-node/encoding"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/bridge"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"
	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// LeafTypeAsset represents a bridge asset
	LeafTypeAsset uint32 = 0
	// LeafTypeMessage represents a bridge message
	LeafTypeMessage uint32 = 1
)

// Client is the utillity client
type Client struct {
	// Client ethclient
	*ethclient.Client
}

// NewClient creates client.
func NewClient(ctx context.Context, nodeURL string) (*Client, error) {
	client, err := ethclient.Dial(nodeURL)

	return &Client{
		client,
	}, err
}

// GetSigner return a transaction signer.
func (c Client) GetSigner(ctx context.Context, accHexPrivateKey string) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(accHexPrivateKey, "0x"))
	if err != nil {
		return nil, err
	}
	chainID, err := c.NetworkID(ctx)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactorWithChainID(privateKey, chainID)
}

// DeployERC20 deploys erc20 smc.
func (c Client) DeployERC20(ctx context.Context, name, symbol string, auth *bind.TransactOpts) (common.Address, *ERC20.ERC20, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, instance, err := ERC20.DeployERC20(auth, c.Client, name, symbol)
	if err != nil {
		return common.Address{}, nil, err
	}
	err = WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)

	return addr, instance, err
}

// DeployBridgeMessageReceiver deploys the brdige message receiver smc.
func (c Client) DeployBridgeMessageReceiver(ctx context.Context, auth *bind.TransactOpts) (common.Address, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, _, err := BridgeMessageReceiver.DeployBridgeMessageReceiver(auth, c.Client)
	if err != nil {
		return common.Address{}, err
	}
	err = WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)

	return addr, err
}

// ApproveERC20 approves erc20 tokens.
func (c Client) ApproveERC20(ctx context.Context, erc20Addr, spender common.Address, amount *big.Int, auth *bind.TransactOpts) error {
	erc20sc, err := ERC20.NewERC20(erc20Addr, c.Client)
	if err != nil {
		return err
	}
	tx, err := erc20sc.Approve(auth, spender, amount)
	if err != nil {
		return err
	}
	const txMinedTimeoutLimit = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)
}

// MintERC20 mint erc20 tokens.
func (c Client) MintERC20(ctx context.Context, erc20Addr common.Address, amount *big.Int, auth *bind.TransactOpts) error {
	erc20sc, err := ERC20.NewERC20(erc20Addr, c.Client)
	if err != nil {
		return err
	}
	tx, err := erc20sc.Mint(auth, amount)
	if err != nil {
		return err
	}
	const txMinedTimeoutLimit = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)
}

// SendBridgeAsset sends a bridge asset transaction.
func (c Client) SendBridgeAsset(ctx context.Context, tokenAddr common.Address, amount *big.Int, destNetwork uint32,
	destAddr *common.Address, metadata []byte, bridgeSCAddr common.Address, auth *bind.TransactOpts,
) error {
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	if destAddr == nil {
		destAddr = &auth.From
	}
	br, err := bridge.NewBridge(bridgeSCAddr, c.Client)
	if err != nil {
		return nil
	}
	tx, err := br.BridgeAsset(auth, tokenAddr, destNetwork, *destAddr, amount, metadata)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// SendBridgeMessage sends a bridge message transaction.
func (c Client) SendBridgeMessage(ctx context.Context, destNetwork uint32, destAddr common.Address, metadata []byte,
	bridgeSCAddr common.Address, auth *bind.TransactOpts,
) error {
	br, err := bridge.NewBridge(bridgeSCAddr, c.Client)
	if err != nil {
		return nil
	}
	tx, err := br.BridgeMessage(auth, destNetwork, destAddr, metadata)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// SendClaim sends a claim transaction.
func (c Client) SendClaim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte, globalExitRoot *etherman.GlobalExitRoot, bridgeSCAddr common.Address, auth *bind.TransactOpts) error {
	br, err := bridge.NewBridge(bridgeSCAddr, c.Client)
	if err != nil {
		return err
	}
	amount, _ := new(big.Int).SetString(deposit.Amount, encoding.Base10)
	var tx *types.Transaction
	if deposit.LeafType == LeafTypeAsset {
		tx, err = br.ClaimAsset(auth, smtProof, uint32(deposit.DepositCnt), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], deposit.OrigNet, common.HexToAddress(deposit.OrigAddr), deposit.DestNet, common.HexToAddress(deposit.DestAddr), amount, common.FromHex(deposit.Metadata))
	} else if deposit.LeafType == LeafTypeMessage {
		tx, err = br.ClaimMessage(auth, smtProof, uint32(deposit.DepositCnt), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], deposit.OrigNet, common.HexToAddress(deposit.OrigAddr), deposit.DestNet, common.HexToAddress(deposit.DestAddr), amount, common.FromHex(deposit.Metadata))
	}
	if err != nil {
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().String()
		}
		log.Error("Error: ", err, ". Tx Hash: ", txHash)
		return err
	}

	// wait transfer to be mined
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// WaitTxToBeMined waits until a tx is mined or forged.
func WaitTxToBeMined(ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) error {
	return ops.WaitTxToBeMined(ctx, client, tx, timeout)
}
