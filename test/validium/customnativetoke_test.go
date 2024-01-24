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
	// Check initial globalExitRoot.
	globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
	require.NoError(t, err)
	t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
	var destNetwork uint32 = 1
	amount1 := new(big.Int).SetUint64(1000000)
	nativeTokenL1Addr := common.HexToAddress(operations.PolTokenAddress)
	clientL1, err := ethclient.Dial("http://localhost:8545")
	require.NoError(t, err)
	l1MaticSC, err := ERC20.NewERC20(nativeTokenL1Addr, clientL1)
	require.NoError(t, err)

	// Send MATIC to the L1 account
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

	// Check that recevier doesnt have funds before bridge
	destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
	require.NoError(t, err)
	require.Equal(t, balance.Int64(), int64(0))

	// Send L1 deposit: MATIC on L1 => Native token on L2
	bridgeAddr := common.HexToAddress("0x80a540502706aa690476D5534e26939894559c05")
	client, err := utils.NewClient(ctx, "http://localhost:8545", bridgeAddr)
	require.NoError(t, err)
	err = client.ApproveERC20(ctx, nativeTokenL1Addr, bridgeAddr, amount1, authL1)
	require.NoError(t, err)
	err = opsman.SendL1Deposit(ctx, nativeTokenL1Addr, amount1, destNetwork, &destAddr)
	require.NoError(t, err)
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	t.Log("Deposit: ", deposits[0])
	t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
	err = opsman.CheckL2Claim(ctx, uint(deposits[0].DestNet), uint(deposits[0].DepositCnt))
	require.NoError(t, err)

	// Check the balance of native token on L2
	balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
	require.NoError(t, err)
	require.Equal(t, balance.Int64(), amount1.Int64())

	// Check L1 funds of MATIC before L2 => L1 bridge
	balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, nativeTokenL1Addr, &destAddr)
	require.NoError(t, err)
	require.Equal(t, int64(0), balance.Int64())

	// Send L2 Deposit: Native token on L2 => MATIC L1
	destNetwork = 0
	l2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &bridgeAddr)
	require.NoError(t, err)
	t.Logf("L2 Bridge Balance: %v", l2Balance)
	err = opsman.SendL2Deposit(ctx, common.Address{}, amount1, destNetwork, &destAddr)
	require.NoError(t, err)
	l2Balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &bridgeAddr)
	require.NoError(t, err)
	t.Logf("L2 Bridge Balance: %v", l2Balance)

	// Get Bridge Info By DestAddr
	deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	t.Log("Deposit 2: ", deposits[0])
	// Get the claim data
	smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
	require.NoError(t, err)
	// Claim funds in L1
	err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
	require.NoError(t, err)

	// Check L1 funds to see if the amount has been increased (MATIC)
	balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, nativeTokenL1Addr, &destAddr)
	require.NoError(t, err)
	require.Equal(t, amount1, balance)
	// Check L2 funds to see that the amount has been reduced (Native coin)
	balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
	require.NoError(t, err)
	require.Equal(t, int64(0), balance.Int64())
}
