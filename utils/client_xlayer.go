package utils

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
)

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
