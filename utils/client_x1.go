package utils

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

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

// SendClaimX1 sends a claim transaction
func (c *Client) SendClaimX1(ctx context.Context, deposit *etherman.Deposit, smtProof, rollupSmtProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, rollupID uint, auth *bind.TransactOpts) (*types.Transaction, error) {
	var (
		tx  *types.Transaction
		err error
	)
	mainnetFlag := deposit.NetworkID == 0
	rollupIndex := rollupID - 1
	localExitRootIndex := deposit.DepositCount
	globalIndex := etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	if deposit.LeafType == uint8(LeafTypeAsset) {
		tx, err = c.bridge.ClaimAsset(auth, smtProof, rollupSmtProof, globalIndex, globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	} else if deposit.LeafType == uint8(LeafTypeMessage) {
		tx, err = c.bridge.ClaimMessage(auth, smtProof, rollupSmtProof, globalIndex, globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
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
