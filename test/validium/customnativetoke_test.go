package validium

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/test/contracts/bin/ERC20"
	validiumOperations "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestCustomNativeToken(t *testing.T) {
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
	opsman, err := operations.NewManager(ctx, opsCfg)
	require.NoError(t, err)
	/*
		1. Bridge from L1 to L2.
		2. Bridge from L2 to L1
		3. Claim the deposits in both layers
	*/
	// Check initial globalExitRoot.
	globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
	require.NoError(t, err)
	t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
	// Send L1 deposit
	var destNetwork uint32 = 1
	amount1 := new(big.Int).SetUint64(1000000)
	nativeTokenL1Addr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	clientL1, err := ethclient.Dial("http://localhost:8545")
	require.NoError(t, err)
	l1MaticSC, err := ERC20.NewERC20(nativeTokenL1Addr, clientL1)
	require.NoError(t, err)
	// Send MATIC to the random account
	authL1WithFunds, err := validiumOperations.GetAuth(validiumOperations.DefaultSequencerPrivateKey, validiumOperations.DefaultL1ChainID)
	require.NoError(t, err)
	authL1, err := validiumOperations.GetAuth("0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", validiumOperations.DefaultL1ChainID)
	require.NoError(t, err)
	balance, err := l1MaticSC.BalanceOf(nil, authL1WithFunds.From)
	require.NoError(t, err)
	origAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
	require.GreaterOrEqual(t, balance.Int64(), amount1.Int64())
	tx, err := l1MaticSC.Transfer(authL1WithFunds, origAddr, amount1)
	require.NoError(t, err)
	err = operations.WaitTxToBeMined(ctx, clientL1, tx, validiumOperations.DefaultTimeoutTxToBeMined)
	require.NoError(t, err)
	balance, err = l1MaticSC.BalanceOf(nil, origAddr)
	require.NoError(t, err)
	require.GreaterOrEqual(t, balance.Int64(), amount1.Int64())

	bridgeAddr := common.HexToAddress("0x4C739372258826995C302CD655beE12689B97d3F")
	client, err := utils.NewClient(ctx, "http://localhost:8545", bridgeAddr)
	require.NoError(t, err)
	err = client.ApproveERC20(ctx, nativeTokenL1Addr, bridgeAddr, amount1, authL1)
	require.NoError(t, err)

	destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	// Check that recevier doesnt have funds beforedeposit
	balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
	require.NoError(t, err)
	require.Equal(t, balance.Int64(), int64(0))
	// First deposit
	err = opsman.SendL1Deposit(ctx, nativeTokenL1Addr, amount1, destNetwork, &destAddr)
	require.NoError(t, err)
	// Get Bridge Info By DestAddr
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	t.Log("Deposit: ", deposits[0])
	t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
	// Check the claim tx
	err = opsman.CheckL2Claim(ctx, uint(deposits[0].DestNet), uint(deposits[0].DepositCnt))
	require.NoError(t, err)

	// time.Sleep(3 * time.Second) // wait for sync token_wrapped event
	// tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, nativeTokenL1Addr, false)
	// require.NoError(t, err)
	// balance2, err := opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
	// require.NoError(t, err)
	// t.Log("Init account balance l2: ", balance2)
	balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
	require.NoError(t, err)
	require.Equal(t, balance.Int64(), amount1.Int64())
}
