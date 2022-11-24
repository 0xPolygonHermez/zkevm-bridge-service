package e2e

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	l1BridgeAddr = common.HexToAddress("0x0165878A594ca255338adfa4d48449f69242Eb8F")
	l2BridgeAddr = common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")
)

// TestBridgeMessage tests the flow of bridge message tx.
func TestBridgeMessage(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	opsCfg := &operations.Config{
		Storage: db.Config{
			Database: "postgres",
			Name:     "test_db",
			User:     "test_user",
			Password: "test_password",
			Host:     "localhost",
			Port:     "5435",
			MaxConns: 10,
		},
		BT: bridgectrl.Config{
			Store:  "postgres",
			Height: uint8(32),
		},
	}
	opsman, err := operations.NewManager(ctx, opsCfg)
	require.NoError(t, err)

	t.Run("L1 Bridge Message Test", func(t *testing.T) {
		// Send L1 bridge message
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(10000000000000000000)

		destAddr, err := opsman.DeployBridgeMessageReceiver(ctx, operations.L1)
		require.NoError(t, err)

		err = opsman.SendL1BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata"))
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim a bridge message in L2
		t.Logf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		require.NoError(t, err)
	})

	t.Run("L2 Bridge Message Test", func(t *testing.T) {
		// Send L2 bridge message
		var destNetwork uint32 = 0
		amount := new(big.Int).SetUint64(10000000000000000000)

		destAddr, err := opsman.DeployBridgeMessageReceiver(ctx, operations.L2)
		require.NoError(t, err)

		err = opsman.SendL2BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata"))
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim a bridge message in L1
		t.Logf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
		require.NoError(t, err)
	})
}
