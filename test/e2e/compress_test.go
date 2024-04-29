//go:build compress
// +build compress
package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)
const (
	defaultInterval = 10 * time.Second
	defaultDeadline = 600 * time.Second
)
func depositFromL1(ctx context.Context, opsman *operations.Manager, destAddr common.Address, t *testing.T) {
	amount := new(big.Int).SetUint64(250000000000000000)
	tokenAddr := common.Address{} // This means is eth
	var destNetwork uint32 = 1
	// L1 Deposit
	err := opsman.SendMultipleL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr, 30)
	require.NoError(t, err)

	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	time.Sleep(5 * time.Second) // Delay to give time to the synchronizer to read all events
	// Check a L2 claim tx
	err = opsman.CustomCheckL2Claim(ctx, uint(deposits[0].DestNet), uint(deposits[0].DepositCnt), defaultInterval, defaultDeadline)
	require.NoError(t, err)
}

func TestClaimCompressor(t *testing.T) {
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
		BS: server.Config{
			GRPCPort:         "9090",
			HTTPPort:         "8080",
			CacheSize:        100000,
			DefaultPageLimit: 25,
			MaxPageLimit:     100,
			BridgeVersion:    "v1",
			DB: db.Config{
				Database: "postgres",
				Name:     "test_db",
				User:     "test_user",
				Password: "test_password",
				Host:     "localhost",
				Port:     "5435",
				MaxConns: 10,
			},
		},
	}

	require.NoError(t, operations.StartBridge())
	opsman, err := operations.NewManager(ctx, opsCfg)
	require.NoError(t, err)
	const st time.Duration = 20 // wait until the syncing is finished
	time.Sleep(st * time.Second)

	t.Run("Test claim compressor", func(t *testing.T) {
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		depositFromL1(ctx, opsman, destAddr, t)
		// Check number claim events
		numberClaims, err := opsman.GetNumberClaims(ctx, destAddr.String())
		require.NoError(t, err)
		require.Equal(t, 30, numberClaims)
		// Check L2 balance
		balance, err := opsman.CheckAccountBalance(ctx, "l2", &destAddr)
		require.NoError(t, err)
		require.Equal(t, "7500000000000000435", balance.String())
	})
}
