package utils

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-core/encoding"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/test/contracts/bin/ERC20"
	ops "github.com/hermeznetwork/hermez-core/test/operations"
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
	const txMinedTimeoutLimit = 20 * time.Second
	addr, tx, instance, err := ERC20.DeployERC20(auth, c.Client, name, symbol)
	if err != nil {
		return common.Address{}, nil, err
	}
	err = WaitTxToBeMined(ctx, c.Client, tx.Hash(), txMinedTimeoutLimit)

	return addr, instance, err
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
	const txMinedTimeoutLimit = 20 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx.Hash(), txMinedTimeoutLimit)
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
	const txMinedTimeoutLimit = 20 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx.Hash(), txMinedTimeoutLimit)
}

// SendBridge sends a bridge transaction.
func (c Client) SendBridge(ctx context.Context, tokenAddr common.Address, amount *big.Int,
	destNetwork uint32, destAddr *common.Address, bridgeSCAddr common.Address, auth *bind.TransactOpts,
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
	tx, err := br.Bridge(auth, tokenAddr, amount, destNetwork, *destAddr)
	if err != nil {
		return err
	}

	// wait transfer to be included in a batch
	const txTimeout = 15 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx.Hash(), txTimeout)
	if err != nil {
		return err
	}
	// Wait until the batch that includes the tx is consolidated
	const t time.Duration = 45
	time.Sleep(t * time.Second)
	return nil
}

// SendClaim send a claim transaction.
func (c Client) SendClaim(ctx context.Context, deposit *pb.Deposit, smtProof [][32]byte, globalExitRooNum *big.Int, globalExitRoot *etherman.GlobalExitRoot, bridgeSCAddr common.Address, auth *bind.TransactOpts) error {
	br, err := bridge.NewBridge(bridgeSCAddr, c.Client)
	if err != nil {
		return err
	}
	amount, _ := new(big.Int).SetString(deposit.Amount, encoding.Base10)
	tx, err := br.Claim(auth, common.HexToAddress(deposit.TokenAddr), amount, deposit.OrigNet, deposit.DestNet,
		common.HexToAddress(deposit.DestAddr), smtProof, uint32(deposit.DepositCnt), globalExitRooNum,
		globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1])
	if err != nil {
		return err
	}

	// wait transfer to be mined
	const txTimeout = 15 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx.Hash(), txTimeout)
	if err != nil {
		return err
	}

	// Wait for the consolidation
	const t time.Duration = 30
	time.Sleep(t * time.Second)
	return nil
}

// WaitTxToBeMined waits until a tx is mined or forged.
func WaitTxToBeMined(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) error {
	w := ops.NewWait()
	return w.TxToBeMined(client, hash, timeout)
}
