//go:build edge
// +build edge

package e2e

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func depositFromL1X1(ctx context.Context, opsman *operations.Manager, t *testing.T) {
	amount := new(big.Int).SetUint64(250000000000000000)
	okbAddr := common.HexToAddress("0x82109a709138A2953C720D3d775168717b668ba6") // This means is okb
	destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	var destNetwork uint32 = 1
	// L1 Deposit
	err := opsman.ApproveERC20OKB(ctx, okbAddr, amount)
	require.NoError(t, err)
	err = opsman.SendL1Deposit(ctx, okbAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)

	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	// Check a L2 claim tx
	err = opsman.CheckL2Claim(ctx, uint(deposits[0].DestNet), uint(deposits[0].DepositCnt))
	require.NoError(t, err)
}

func depositFromL2X1(ctx context.Context, opsman *operations.Manager, t *testing.T) {
	// Send L2 Deposit to withdraw the some funds
	var destNetwork uint32 = 0
	amount := new(big.Int).SetUint64(100000000000000000)
	tokenAddr := common.Address{} // This means is okb
	destAddr := common.HexToAddress("0x2ecf31ece36ccac2d3222a303b1409233ecbb225")
	err := opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)

	// Get Bridge Info By DestAddr
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	// Check globalExitRoot
	// Get the claim data
	smtProof, globalExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
	require.NoError(t, err)
	// Claim funds in L1
	err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globalExitRoot)
	require.NoError(t, err)
}
