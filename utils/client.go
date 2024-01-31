package utils

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/mocksmartcontracts/BridgeMessageReceiver"
	zkevmtypes "github.com/0xPolygonHermez/zkevm-node/config/types"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridge"
	"github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/polygonzkevmbridgel2"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"
	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
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

	mtHeight = 32
	keyLen   = 32
)

// Client is the utillity client
type Client struct {
	// Client ethclient
	*ethclient.Client
	bridge   *polygonzkevmbridge.Polygonzkevmbridge
	bridgeL2 *polygonzkevmbridgel2.Polygonzkevmbridgel2
}

// NewClient creates client.
func NewClient(ctx context.Context, nodeURL string, bridgeSCAddr common.Address) (*Client, error) {
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, err
	}
	var br *polygonzkevmbridge.Polygonzkevmbridge
	var brl2 *polygonzkevmbridgel2.Polygonzkevmbridgel2
	if len(bridgeSCAddr) != 0 {
		br, _ = polygonzkevmbridge.NewPolygonzkevmbridge(bridgeSCAddr, client)
		brl2, err = polygonzkevmbridgel2.NewPolygonzkevmbridgel2(bridgeSCAddr, client)
	}
	log.Infof("nodeURL:%v, bridgeSCAddr:%v, ", nodeURL, bridgeSCAddr.String())
	return &Client{
		Client:   client,
		bridge:   br,
		bridgeL2: brl2,
	}, err
}

// GetSigner returns a transaction signer.
func (c *Client) GetSigner(ctx context.Context, accHexPrivateKey string) (*bind.TransactOpts, error) {
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

// GetSignerFromKeystore returns a transaction signer from the keystore file.
func (c *Client) GetSignerFromKeystore(ctx context.Context, ks zkevmtypes.KeystoreFileConfig) (*bind.TransactOpts, error) {
	keystoreEncrypted, err := os.ReadFile(filepath.Clean(ks.Path))
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(keystoreEncrypted, ks.Password)
	if err != nil {
		return nil, err
	}
	chainID, err := c.NetworkID(ctx)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactorWithChainID(key.PrivateKey, chainID)
}

// CheckTxWasMined check if a tx was already mined
func (c *Client) CheckTxWasMined(ctx context.Context, txHash common.Hash) (bool, *types.Receipt, error) {
	receipt, err := c.TransactionReceipt(ctx, txHash)
	if errors.Is(err, ethereum.NotFound) {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	return true, receipt, nil
}

// DeployERC20 deploys erc20 smc.
func (c *Client) DeployERC20(ctx context.Context, name, symbol string, auth *bind.TransactOpts) (common.Address, *ERC20.ERC20, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, instance, err := ERC20.DeployERC20(auth, c.Client, name, symbol)
	if err != nil {
		return common.Address{}, nil, err
	}
	err = WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)

	return addr, instance, err
}

// DeployBridgeMessageReceiver deploys the brdige message receiver smc.
func (c *Client) DeployBridgeMessageReceiver(ctx context.Context, auth *bind.TransactOpts) (common.Address, error) {
	const txMinedTimeoutLimit = 60 * time.Second
	addr, tx, _, err := BridgeMessageReceiver.DeployBridgeMessageReceiver(auth, c.Client)
	if err != nil {
		return common.Address{}, err
	}
	err = WaitTxToBeMined(ctx, c.Client, tx, txMinedTimeoutLimit)

	return addr, err
}

// ApproveERC20 approves erc20 tokens.
func (c *Client) ApproveERC20(ctx context.Context, erc20Addr, spender common.Address, amount *big.Int, auth *bind.TransactOpts) error {
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
func (c *Client) MintERC20(ctx context.Context, erc20Addr common.Address, amount *big.Int, auth *bind.TransactOpts) error {
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
func (c *Client) SendBridgeAsset(ctx context.Context, tokenAddr common.Address, amount *big.Int, destNetwork uint32,
	destAddr *common.Address, metadata []byte, auth *bind.TransactOpts,
) error {
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	if destAddr == nil {
		destAddr = &auth.From
	}
	log.Infof("token address:%v, amount:%v, destnetwork:%v, dest address:%v", tokenAddr.String(), amount.String(), destNetwork, destAddr.String())
	tx, err := c.bridge.BridgeAsset(auth, destNetwork, *destAddr, amount, tokenAddr, true, metadata)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// SendBridgeMessage sends a bridge message transaction.
func (c *Client) SendBridgeMessage(ctx context.Context, destNetwork uint32, destAddr common.Address, metadata []byte,
	auth *bind.TransactOpts,
) error {
	tx, err := c.bridge.BridgeMessage(auth, destNetwork, destAddr, true, metadata)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// SendL2BridgeMessage sends a bridge message transaction.
func (c *Client) SendL2BridgeMessage(ctx context.Context, destNetwork uint32, amountWETH *big.Int, destAddr common.Address, metadata []byte,
	auth *bind.TransactOpts,
) error {
	tx, err := c.bridgeL2.BridgeMessage(auth, destNetwork, destAddr, amountWETH, true, metadata)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// BuildSendClaim builds a tx data to be sent to the bridge method SendClaim.
func (c *Client) BuildSendClaim(ctx context.Context, deposit *etherman.Deposit, smtProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, nonce, gasPrice int64, gasLimit uint64, auth *bind.TransactOpts) (*types.Transaction, error) {
	opts := *auth
	opts.NoSend = true
	// force nonce, gas limit and gas price to avoid querying it from the chain
	opts.Nonce = big.NewInt(nonce)
	opts.GasPrice = big.NewInt(gasPrice)
	opts.GasLimit = gasLimit

	var (
		tx  *types.Transaction
		err error
	)
	if deposit.LeafType == uint8(LeafTypeAsset) {
		tx, err = c.bridge.ClaimAsset(&opts, smtProof, uint32(deposit.DepositCount), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	} else if deposit.LeafType == uint8(LeafTypeMessage) {
		tx, err = c.bridge.ClaimMessage(&opts, smtProof, uint32(deposit.DepositCount), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	}
	if err != nil {
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().String()
		}
		log.Error("Error: ", err, ". Tx Hash: ", txHash)
		return nil, fmt.Errorf("failed to build SendClaim tx, err: %w", err)
	}

	return tx, nil
}

// SendClaim sends a claim transaction
func (c *Client) SendClaim(ctx context.Context, deposit *etherman.Deposit, smtProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, auth *bind.TransactOpts) (*types.Transaction, error) {
	var (
		tx  *types.Transaction
		err error
	)
	if deposit.LeafType == uint8(LeafTypeAsset) {
		tx, err = c.bridge.ClaimAsset(auth, smtProof, uint32(deposit.DepositCount), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	} else if deposit.LeafType == uint8(LeafTypeMessage) {
		tx, err = c.bridge.ClaimMessage(auth, smtProof, uint32(deposit.DepositCount), globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	}
	if err != nil {
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().String()
		}
		log.Error("Error: ", err, ". Tx Hash: ", txHash)
		return nil, err
	}

	return tx, nil
}

// SendClaimAndWait sends a claim transaction and wait for the claim tx to be mined.
func (c *Client) SendClaimAndWait(ctx context.Context, deposit *pb.Deposit, smtProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, auth *bind.TransactOpts) error {
	tx, err := c.SendClaim(ctx, PbToEthermanDeposit(deposit), smtProof, globalExitRoot, auth)
	if err != nil {
		return err
	}

	// wait transfer to be mined
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// SetL2TokensAllowed set l2 token allowed.
func (c *Client) SetL2TokensAllowed(ctx context.Context, allowed bool, auth *bind.TransactOpts) error {
	result, _ := c.bridgeL2.IsAllL2TokensAllowed(&bind.CallOpts{})
	if result == allowed {
		log.Infof("Do nothing, allowed:%v", allowed)
		return nil
	}

	tx, err := c.bridgeL2.SetAllL2TokensAllowed(auth, allowed)
	if err != nil {
		log.Error("Error: ", err)
		return err
	}
	// wait transfer to be included in a batch
	const txTimeout = 60 * time.Second
	return WaitTxToBeMined(ctx, c.Client, tx, txTimeout)
}

// WaitTxToBeMined waits until a tx is mined or forged.
func WaitTxToBeMined(ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) error {
	return ops.WaitTxToBeMined(ctx, client, tx, timeout)
}
