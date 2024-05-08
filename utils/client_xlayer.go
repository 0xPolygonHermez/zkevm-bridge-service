package utils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// BuildSendClaimXLayer builds a tx data to be sent to the bridge method SendClaim.
func (c *Client) BuildSendClaimXLayer(ctx context.Context, deposit *etherman.Deposit, smtProof [mtHeight][keyLen]byte, smtRollupProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, nonce, gasPrice int64, gasLimit uint64, rollupID uint, auth *bind.TransactOpts) (*types.Transaction, error) {
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
	mainnetFlag := deposit.NetworkID == 0
	rollupIndex := rollupID - 1
	localExitRootIndex := deposit.DepositCount
	globalIndex := etherman.GenerateGlobalIndex(mainnetFlag, rollupIndex, localExitRootIndex)
	if deposit.LeafType == uint8(LeafTypeAsset) {
		tx, err = c.bridge.ClaimAsset(&opts, smtProof, smtRollupProof, globalIndex, globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), deposit.DestinationAddress, deposit.Amount, deposit.Metadata)
	} else if deposit.LeafType == uint8(LeafTypeMessage) {
		destAddr := deposit.DestinationAddress
		emptyAddress := common.Address{}
		if deposit.DestContractAddress != emptyAddress {
			destAddr = deposit.DestContractAddress
		}
		tx, err = c.bridge.ClaimMessage(&opts, smtProof, smtRollupProof, globalIndex, globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), destAddr, deposit.Amount, deposit.Metadata)
	}
	if err != nil {
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().String()
		}
		log.Error("Error: ", err, ". Tx Hash: ", txHash)
		return nil, fmt.Errorf("failed to build SendClaim tx, err: %v", err)
	}

	return tx, nil
}

// SendClaimXLayer sends a claim transaction
func (c *Client) SendClaimXLayer(ctx context.Context, deposit *etherman.Deposit, smtProof, rollupSmtProof [mtHeight][keyLen]byte, globalExitRoot *etherman.GlobalExitRoot, rollupID uint, auth *bind.TransactOpts) (*types.Transaction, error) {
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
		destAddr := deposit.DestinationAddress
		emptyAddress := common.Address{}
		if deposit.DestContractAddress != emptyAddress {
			destAddr = deposit.DestContractAddress
		}
		tx, err = c.bridge.ClaimMessage(auth, smtProof, rollupSmtProof, globalIndex, globalExitRoot.ExitRoots[0], globalExitRoot.ExitRoots[1], uint32(deposit.OriginalNetwork), deposit.OriginalAddress, uint32(deposit.DestinationNetwork), destAddr, deposit.Amount, deposit.Metadata)
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
