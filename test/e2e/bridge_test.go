package e2e

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/vectors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var (
	l1BridgeAddr = common.HexToAddress("0x0165878A594ca255338adfa4d48449f69242Eb8F")
	l2BridgeAddr = common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")
)

// TestE2E tests the flow of deposit and withdraw funds using the vector
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testCases, err := vectors.LoadE2ETestVectors("./../vectors/src/e2e-test.json")
	require.NoError(t, err)

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

	for _, testCase := range testCases {
		t.Run("Test id "+strconv.FormatUint(uint64(testCase.ID), 10), func(t *testing.T) {
			// Check initial globalExitRoot. Must fail because at the beginning, no globalExitRoot event is thrown.
			globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
			require.NoError(t, err)
			t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
			// Send L1 deposit
			var destNetwork uint32 = 1
			amount := new(big.Int).SetUint64(10000000000000000000)
			tokenAddr := common.Address{} // This means is eth
			destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")

			l1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
			require.NoError(t, err)
			t.Logf("L1 Bridge Balance: %v", l1Balance)
			err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			require.NoError(t, err)
			l1Balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
			require.NoError(t, err)
			t.Logf("L1 Bridge Balance: %v", l1Balance)

			// Check globalExitRoot
			globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
			require.NoError(t, err)
			t.Logf("Before deposit global exit root: %v", globalExitRootSMC)
			t.Logf("After deposit global exit root: %v", globalExitRoot2)
			require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			// Get Bridge Info By DestAddr
			deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
			require.NoError(t, err)
			// Check L2 funds
			balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			require.NoError(t, err)
			initL2Balance := big.NewInt(0)
			require.Equal(t, 0, balance.Cmp(initL2Balance))
			t.Log("Deposit: ", deposits[0])
			// Get the claim data
			smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].OrigNet), uint(deposits[0].DepositCnt))
			require.NoError(t, err)
			proof := testCase.Txs[0].Params[5].([]interface{})
			require.Equal(t, len(proof), len(smtProof))
			for i, s := range smtProof {
				require.Equal(t, proof[i].(string), "0x"+hex.EncodeToString(s[:]))
			}
			// Claim funds in L1
			err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
			require.NoError(t, err)
			// Check L2 funds to see if the amount has been increased
			balance2, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			require.NoError(t, err)
			require.NotEqual(t, balance, balance2)
			require.Equal(t, amount, balance2)

			// Check globalExitRoot
			globalExitRoot3, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
			require.NoError(t, err)
			// Send L2 Deposit to withdraw the some funds
			destNetwork = 0
			amount = new(big.Int).SetUint64(1000000000000000000)
			l2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
			require.NoError(t, err)
			t.Logf("L2 Bridge Balance: %v", l2Balance)
			err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			require.NoError(t, err)
			l2Balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
			require.NoError(t, err)
			t.Logf("L2 Bridge Balance: %v", l2Balance)

			// Get Bridge Info By DestAddr
			deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
			require.NoError(t, err)
			t.Log("Deposit 2: ", deposits[0])
			// Check globalExitRoot
			globalExitRoot4, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
			require.NoError(t, err)
			t.Logf("Global3 %+v: ", globalExitRoot3)
			t.Logf("Global4 %+v: ", globalExitRoot4)
			require.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
			require.NotEqual(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
			require.Equal(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
			// Check L1 funds
			balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
			require.NoError(t, err)
			require.Equal(t, 0, big.NewInt(0).Cmp(balance))
			// Get the claim data
			smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
			require.NoError(t, err)
			// Claim funds in L1
			err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
			require.NoError(t, err)

			// Check L1 funds to see if the amount has been increased
			balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
			require.NoError(t, err)
			require.Equal(t, big.NewInt(1000000000000000000), balance)
			// Check L2 funds to see that the amount has been reduced
			balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			require.NoError(t, err)
			require.Equal(t, big.NewInt(9000000000000000000), balance)
		})
	}

	t.Run("Reversal ERC20 Test", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)

		// Send L1 deposit
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
		destAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		amount = new(big.Int).SetUint64(1000000000000000000)
		err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		t.Logf("Before deposit global exit root: %v", globalExitRootSMC)
		t.Logf("After deposit global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
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
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		t.Logf("globalExitRoot: %+v", globaExitRoot)
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
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
		t.Logf("Global3 %+v: ", globalExitRoot3)
		t.Logf("Global4 %+v: ", globalExitRoot4)
		require.NotEqual(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		require.Equal(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
		// Check L2 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &destAddr)
		require.NoError(t, err)
		t.Log("balance: ", balance)
		require.Equal(t, 0, big.NewInt(0).Cmp(balance))
		t.Log("deposits[0]: ", deposits[0])
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		t.Log("globaExitRoot: ", globaExitRoot, globaExitRoot.GlobalExitRootNum)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
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

	t.Run("ERC20 Test", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", operations.L1)
		require.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, amount, operations.L1)
		require.NoError(t, err)
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		balance, err := opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &origAddr)
		require.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetTrustedGlobalExitRootSynced(ctx)
		require.NoError(t, err)
		t.Logf("Before deposit global exit root: %v", globalExitRootSMC)
		t.Logf("After deposit global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit: ", deposits[0])
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
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
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount, destNetwork, &destAddr)
		require.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("Deposit 2: ", deposits[0])
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		require.NoError(t, err)
		t.Logf("Global3 %+v: ", globalExitRoot3)
		t.Logf("Global4 %+v: ", globalExitRoot4)
		require.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
		require.NotEqual(t, globalExitRoot3.ExitRoots[1], globalExitRoot4.ExitRoots[1])
		require.Equal(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		// Check L1 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &destAddr)
		require.NoError(t, err)
		require.Equal(t, 0, big.NewInt(0).Cmp(balance))
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
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
	t.Run("Multi deposits tests", func(t *testing.T) {
		/*
			1. Do 3 deposits/bridges
			2. Do 2 more deposits
			3. Claim the first 3 deposits in L2
		*/

		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		require.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
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
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
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
		t.Logf("Before deposits global exit root: %v", globalExitRootSMC)
		t.Logf("After deposits global exit root: %v", globalExitRoot2)
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

		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[2].NetworkId), uint(deposits[2].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[2], smtProof, globaExitRoot)
		require.NoError(t, err)
		time.Sleep(3 * time.Second) // wait for sync token_wrapped event
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		require.NoError(t, err)
		t.Log("TokenWrapped: ", tokenWrapped)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, amount1, balance)

		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[1].NetworkId), uint(deposits[1].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[1], smtProof, globaExitRoot)
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, new(big.Int).Add(amount1, amount2), balance)

		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, new(big.Int).SetUint64(6000000000000000000), balance)
	})
	t.Run("L1-L2 bridge tests", func(t *testing.T) {
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
		amount1 := new(big.Int).SetUint64(1000000000000000000)
		totAmount := new(big.Int).SetUint64(3500000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", "l1")
		require.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, totAmount, "l1")
		require.NoError(t, err)
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
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
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
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
		t.Logf("Before deposits global exit root: %v", globalExitRootSMC)
		t.Logf("After deposits global exit root: %v", globalExitRoot2)
		require.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		require.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])

		destNetwork = 0
		amount4 := new(big.Int).SetUint64(500000000000000000)
		// L2 deposit
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount4, destNetwork, &origAddr)
		require.NoError(t, err)
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &origAddr)
		require.NoError(t, err)
		t.Log("deposit: ", deposits[0])
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(500000000000000000), balance)
		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(2000000000000000000), balance)

		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		require.NoError(t, err)
		t.Log("deposit: ", deposits[0])
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		require.NoError(t, err)
		t.Log("globalExitRoot:", globaExitRoot)
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		require.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		require.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		require.Equal(t, new(big.Int).SetUint64(1500000000000000000), balance)
	})
}
