//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	l1BridgeAddr = common.HexToAddress("0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E")
	l2BridgeAddr = common.HexToAddress("0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E")
)

// TestE2E tests the flow of deposit and withdraw funds using the vector
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	opsCfg := &operations.Config{
		L1NetworkURL: "http://localhost:8545",
		L2NetworkURL: "http://localhost:8123",
		L2NetworkID:  1,
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

	t.Run("L1-L2 eth bridge", func(t *testing.T) {
		// Check initial globalExitRoot. Must fail because at the beginning, no globalExitRoot event is thrown.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		log.Debugf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr := common.Address{} // This means is eth
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		initL1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
		require.NoError(t, err)
		log.Debugf("Initial L1 Balance: %v", initL1Balance)
		l1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
		require.NoError(t, err)
		log.Debugf("L1 Bridge Balance: %v", l1Balance)
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		l1Balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
		require.NoError(t, err)
		log.Debugf("L1 Bridge Balance: %v", l1Balance)

		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		log.Debugf("Before deposit global exit root: %v", globalExitRootSMC)
		log.Debugf("After deposit global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Check L2 funds
		balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
		require.NoError(t, err)
		log.Debugf("Initial L2 Balance: %v", balance)
		initL2Balance := balance
		//require.Equal(t, 0, balance.Cmp(initL2Balance))
		log.Debug("Deposit: ", deposits[0], " waiting to be auto-claimed")
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance2, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
		require.NoError(t, err)
		require.NotEqual(t, balance, balance2)

		//require.Equal(t, amount, balance2)
		require.Equal(t, 0, balance2.Sub(balance2, initL2Balance).Cmp(amount))

		// Check globalExitRoot
		globalExitRoot3, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		// Send L2 Deposit to withdraw the some funds
		destNetwork = 0
		amount = new(big.Int).SetUint64(1000000000000000000)
		l2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
		require.NoError(t, err)
		t.Logf("L2 Bridge Balance: %v", l2Balance)
		err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr, operations.L2)
		require.NoError(t, err)
		l2Balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
		require.NoError(t, err)
		log.Debugf("L2 Bridge Balance: %v", l2Balance)

		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		log.Debugf("Deposit 2: ", deposits[0])
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		log.Debugf("Global3 %+v: ", globalExitRoot3)
		log.Debugf("Global4 %+v: ", globalExitRoot4)
		require.NotEqual(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
		require.Equal(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		// Check L1 funds
		balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
		require.NoError(t, err)
		//require.Equal(t, 0, big.NewInt(0).Cmp(balance))
		require.Equal(t, 0, balance.Sub(balance, initL1Balance).Cmp(big.NewInt(0)))
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		testClaimParams := testClaimByGERParams{
			resultOriginal: claimResult{smtProof, smtRollupProof, globaExitRoot},
			opsman:         opsman,
			deposit:        deposits[0],
		}
		testClaimByGER(t, ctx, testClaimParams)

		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)

		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(1000000000000000000), balance.Sub(balance, initL1Balance))
		// Check L2 funds to see that the amount has been reduced
		balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
		require.NoError(t, err)
		require.True(t, big.NewInt(9000000000000000000).Cmp(balance.Sub(balance, initL2Balance)) > 0)

		// TESTING ClaimByGER
		//////////////////////////////
		log.Debugf("Sending another L2Deposit to generate a new globalExitRoot and get ClaimByGER")
		err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Get Bridge Info By DestAddr
		require.NoError(t, err)
		log.Debugf("There are a new GER, I'm checking to generate claim for previous deposit with previous GER")
		testClaimParams = testClaimByGERParams{
			resultOriginal: claimResult{smtProof, smtRollupProof, globaExitRoot},
			opsman:         opsman,
			deposit:        deposits[0],
			useGER:         nil, // nil means use the one in resultOriginal
		}
		testClaimByGER(t, ctx, testClaimParams)
		// Claim funds in L1
		deposits2, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)

		smtProof, smtRollupProof, globaExitRoot, err = opsman.GetClaimData(ctx, uint(deposits2[0].NetworkId), uint(deposits2[0].DepositCnt))
		require.NoError(t, err)

		err = opsman.SendL1Claim(ctx, deposits2[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)

	})

	t.Run("L1-L2 token bridge", func(t *testing.T) {
		/*
			1. Bridge from L1 to L2.
			2. Bridge from L2 to L1
			3. Claim the deposits in both layers
		*/
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		log.Debugf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount1 := new(big.Int).SetUint64(1000000000000000000)
		totAmount := new(big.Int).SetUint64(3500000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", "l1")
		require.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, totAmount, "l1")
		require.NoError(t, err)
		origAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		balance, err := opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		require.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		// First deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		require.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit: ", deposits[0])
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		time.Sleep(3 * time.Second) // wait for sync token_wrapped event
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		require.NoError(t, err)
		balance2, err := opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Init account balance l2: ", balance2)

		// Second deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		log.Debugf("Before deposits global exit root: %v", globalExitRootSMC)
		log.Debugf("After deposits global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])

		destNetwork = 0
		amount4 := new(big.Int).SetUint64(500000000000000000)
		// L2 deposit
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount4, destNetwork, &origAddr, operations.L2)
		require.NoError(t, err)
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &origAddr)
		require.NoError(t, err)
		t.Log("deposit: ", deposits[0])
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(2000000000000000000), balance)

		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("deposit: ", deposits[0])
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, new(big.Int).SetUint64(1500000000000000000), balance)
	})

	t.Run("Reversal ERC20", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		log.Debugf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L2 deposit
		var destNetwork uint32 = 0
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", operations.L2)
		require.NoError(t, err)
		//Mint tokens
		err = opsman.MintERC20(ctx, tokenAddr, amount, operations.L2)
		require.NoError(t, err)
		//Check balance
		origAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		balance, err := opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &origAddr)
		require.NoError(t, err)
		t.Log("Token balance: ", balance, ". tokenaddress: ", tokenAddr, ". account: ", origAddr)
		destAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		amount = new(big.Int).SetUint64(1000000000000000000)
		err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr, operations.L2)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		log.Debugf("Before deposit global exit root: %v", globalExitRootSMC)
		log.Debugf("After deposit global exit root: %v", globalExitRoot2)
		require.Equal(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.NotEqual(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Check L2 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &origAddr)
		require.NoError(t, err)
		require.Equal(t, 0, balance.Cmp(big.NewInt(9000000000000000000)))
		t.Log("Deposit: ", deposits[0])
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		log.Debugf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
		// Get tokenWrappedAddr
		t.Log("token Address:", tokenAddr)
		time.Sleep(3 * time.Second) // wait for sync token_wrapped event
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 1, tokenAddr, true)
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance2, err := opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("balance l1 account after claim funds: ", balance2)
		require.NotEqual(t, balance, balance2)
		require.Equal(t, amount, balance2)

		// Check globalExitRoot
		globalExitRoot3, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		// Send L2 Deposit to withdraw the some funds
		destNetwork = 1
		amount = new(big.Int).SetUint64(600000000000000000)
		err = opsman.SendL1Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit 2: ", deposits[0])
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		log.Debugf("Global3 %+v: ", globalExitRoot3)
		log.Debugf("Global4 %+v: ", globalExitRoot4)
		require.NotEqual(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		// Check L2 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &destAddr)
		require.NoError(t, err)
		t.Log("balance: ", balance)
		require.Equal(t, 0, big.NewInt(0).Cmp(balance))
		t.Log("deposits[0]: ", deposits[0])
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &destAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(600000000000000000), balance)
		// Check L1 funds to see that the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		require.Equal(t, 0, big.NewInt(400000000000000000).Cmp(balance))
	})

	t.Run("ERC20", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		log.Debugf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", operations.L1)
		require.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, amount, operations.L1)
		require.NoError(t, err)
		origAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		balance, err := opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &origAddr)
		require.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		log.Debugf("Before deposit global exit root: %v", globalExitRootSMC)
		log.Debugf("After deposit global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit: ", deposits[0])
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		time.Sleep(3 * time.Second) // wait for sync token_wrapped event
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		require.NoError(t, err)
		t.Log("TokenWrapped: ", tokenWrapped)
		// Check L2 funds to see if the amount has been increased
		balance2, err := opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance2)
		require.Equal(t, amount, balance2)

		// Check globalExitRoot
		globalExitRoot3, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		// Send L2 Deposit to withdraw the some funds
		destNetwork = 0
		amount = new(big.Int).SetUint64(8000000000000000000)
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount, destNetwork, &destAddr, operations.L2)
		require.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit 2: ", deposits[0])
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		log.Debugf("Global3 %+v: ", globalExitRoot3)
		log.Debugf("Global4 %+v: ", globalExitRoot4)
		require.NotEqual(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
		require.Equal(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		// Check L1 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &destAddr)
		require.NoError(t, err)
		require.Equal(t, 0, big.NewInt(0).Cmp(balance))
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &destAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(8000000000000000000), balance)
		// Check L2 funds to see that the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(2000000000000000000), balance)
	})

	t.Run("Multi deposits", func(t *testing.T) {
		/*
			1. Do 3 deposits/bridges
			2. Do 2 more deposits
			3. Claim the first 3 deposits in L2
		*/

		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		log.Debugf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount1 := new(big.Int).SetUint64(1000000000000000000)
		amount2 := new(big.Int).SetUint64(2000000000000000000)
		amount3 := new(big.Int).SetUint64(3000000000000000000)
		totAmount := new(big.Int).SetUint64(9000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", "l1")
		require.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, totAmount, "l1")
		require.NoError(t, err)
		origAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		balance, err := opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		require.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		// First deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		require.NoError(t, err)
		// Second deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount2, destNetwork, &destAddr)
		require.NoError(t, err)
		// Third deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount3, destNetwork, &destAddr)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		log.Debugf("Before deposits global exit root: %v", globalExitRootSMC)
		log.Debugf("After deposits global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposits: ", deposits[:2])
		t.Log("Before getClaimData: ", deposits[2].NetworkId, deposits[2].DepositCnt)
		// Fourth deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &origAddr)
		require.NoError(t, err)
		// Fifth deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount2, destNetwork, &origAddr)
		require.NoError(t, err)
		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		time.Sleep(3 * time.Second) // wait for sync token_wrapped event
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		require.NoError(t, err)
		t.Log("TokenWrapped: ", tokenWrapped)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)

		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, new(big.Int).SetUint64(6000000000000000000), balance)
	})

	t.Run("Bridge Message Test", func(t *testing.T) {
		// Test L1 Bridge Message
		// Send L1 bridge message
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(1000000000000000000)

		destAddr, err := opsman.DeployBridgeMessageReceiver(ctx, operations.L1)
		require.NoError(t, err)

		err = opsman.SendL1BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata 1"), nil)
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Check the claim tx
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot, operations.L2)
		require.NoError(t, err)

		// Test L2 Bridge Message
		// Send L2 bridge message
		destNetwork = 0

		destAddr, err = opsman.DeployBridgeMessageReceiver(ctx, operations.L2)
		require.NoError(t, err)

		err = opsman.SendL2BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata 2"))
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err = opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim a bridge message in L1
		log.Debugf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
	})
	t.Run("Bridge Message Authorized Account Test", func(t *testing.T) {
		// Test L1 Bridge Message
		// Send L1 bridge message
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(1000000000000000000)

		destAddr, err := opsman.DeployBridgeMessageReceiver(ctx, operations.L1)
		require.NoError(t, err)

		pk := "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" //0x90F79bf6EB2c4f870365E785982E1f101E93b906

		err = opsman.SendL1BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata 3"), &pk)
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)

		// Check the claim tx
		err = opsman.CheckClaim(ctx, deposits[0])
		require.NoError(t, err)

		// Test L2 Bridge Message
		// Send L2 bridge message
		destNetwork = 0

		destAddr, err = opsman.DeployBridgeMessageReceiver(ctx, operations.L2)
		require.NoError(t, err)

		err = opsman.SendL2BridgeMessage(ctx, destAddr, destNetwork, amount, []byte("metadata 4"))
		require.NoError(t, err)

		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		// Get the claim data
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim a bridge message in L1
		log.Debugf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
	})
}

const mtProofHeight = 32

type claimResult struct {
	smtProof       [mtProofHeight][bridgectrl.KeyLen]byte
	smtRollupProof [mtProofHeight][bridgectrl.KeyLen]byte
	globaExitRoot  *etherman.GlobalExitRoot
}

type testClaimByGERParams struct {
	resultOriginal claimResult
	opsman         *operations.Manager
	deposit        *pb.Deposit
	useGER         *common.Hash
}

func calculateGlobalExitRoot(ger *etherman.GlobalExitRoot) common.Hash {
	return bridgectrl.Hash(ger.ExitRoots[0], ger.ExitRoots[1])
}

func testClaimByGER(t *testing.T, ctx context.Context, params testClaimByGERParams) {
	var ger common.Hash
	if params.useGER != nil {
		ger = *params.useGER
	} else {
		ger = calculateGlobalExitRoot(params.resultOriginal.globaExitRoot)
	}

	log.Debugf("GetClaimDataByGER: network: %d deposit_cnt:%d GER:%s",
		uint(params.deposit.NetworkId), uint(params.deposit.DepositCnt), ger.String())
	log.Debugf("Checking same claim as GetClaim")
	smtProofByGer, smtRollupProofByGer, globaExitRootByGer, err := params.opsman.GetClaimDataByGER(ctx, uint(params.deposit.NetworkId), uint(params.deposit.DepositCnt), ger)
	require.NoError(t, err)
	require.Equal(t, params.resultOriginal.globaExitRoot, globaExitRootByGer)
	require.Equal(t, params.resultOriginal.smtProof, smtProofByGer)
	require.Equal(t, params.resultOriginal.smtRollupProof, smtRollupProofByGer)
}
