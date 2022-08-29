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
	"github.com/stretchr/testify/assert"
)

var (
	l1BridgeAddr = common.HexToAddress("0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9")
	l2BridgeAddr = common.HexToAddress("0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988")
)

// TestE2E tests the flow of deposit and withdraw funds using the vector
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	defer func() {
		assert.NoError(t, operations.Teardown())
	}()

	testCases, err := vectors.LoadE2ETestVectors("./../vectors/src/e2e-test.json")
	assert.NoError(t, err)

	ctx := context.Background()
	opsCfg := &operations.Config{
		Storage: db.Config{
			Database: "postgres",
			Name:     "test_db",
			User:     "test_user",
			Password: "test_password",
			Host:     "localhost",
			Port:     "5433",
			MaxConns: 10,
		},
		BT: bridgectrl.Config{
			Store:  "postgres",
			Height: uint8(32),
		},
	}
	opsman, err := operations.NewManager(ctx, opsCfg)
	assert.NoError(t, err)

	//Run environment
	// assert.NoError(t, opsman.Setup())

	for _, testCase := range testCases {
		t.Run("Test id "+strconv.FormatUint(uint64(testCase.ID), 10), func(t *testing.T) {
			// Check initial globalExitRoot. Must fail because at the beginning, no globalExitRoot event is thrown.
			globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
			assert.NoError(t, err)
			t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
			// Send L1 deposit
			var destNetwork uint32 = 1
			amount := new(big.Int).SetUint64(10000000000000000000)
			tokenAddr := common.Address{} // This means is eth
			destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")

			l1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
			assert.NoError(t, err)
			t.Logf("L1 Bridge Balance: %v", l1Balance)
			err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			assert.NoError(t, err)
			l1Balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &l1BridgeAddr)
			assert.NoError(t, err)
			t.Logf("L1 Bridge Balance: %v", l1Balance)

			// Check globalExitRoot
			globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
			assert.NoError(t, err)
			t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
			t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			assert.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
			t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			assert.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
			assert.Equal(t, common.HexToHash("0x573768af52d1354a7b83fb784ecbacecf8fead6ad49f25af8909a35b0a7bba05"), globalExitRoot2.ExitRoots[0])
			assert.Equal(t, common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"), globalExitRoot2.ExitRoots[1])
			// Get Bridge Info By DestAddr
			deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
			assert.NoError(t, err)
			// Check L2 funds
			balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			assert.NoError(t, err)
			initL2Balance := big.NewInt(0)
			assert.Equal(t, 0, balance.Cmp(initL2Balance))
			t.Log("Deposits: ", deposits)
			t.Log("deposits[0].OrigNet: ", deposits[0].OrigNet, ". deposits[0].OrigNet: ", deposits[0].DepositCnt)
			// Check the trusted exitroot is synced
			err = opsman.CheckTrustedExitRootSynced(ctx, deposits[0].DepositCnt)
			assert.NoError(t, err)
			// Get the claim data
			smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].OrigNet), uint(deposits[0].DepositCnt))
			assert.NoError(t, err)
			proof := testCase.Txs[0].Params[5].([]interface{})
			assert.Equal(t, len(proof), len(smtProof))
			for i, s := range smtProof {
				assert.Equal(t, proof[i].(string), "0x"+hex.EncodeToString(s[:]))
			}
			// Claim funds in L1
			err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
			assert.NoError(t, err)
			// Check L2 funds to see if the amount has been increased
			balance2, err := opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			assert.NoError(t, err)
			assert.NotEqual(t, balance, balance2)
			assert.Equal(t, amount, balance2)

			// Check globalExitRoot
			globalExitRoot3, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			assert.NoError(t, err)
			// Send L2 Deposit to withdraw the some funds
			destNetwork = 0
			amount = new(big.Int).SetUint64(1000000000000000000)
			l2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
			assert.NoError(t, err)
			t.Logf("L2 Bridge Balance: %v", l2Balance)
			err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
			assert.NoError(t, err)
			l2Balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &l2BridgeAddr)
			assert.NoError(t, err)
			t.Logf("L2 Bridge Balance: %v", l2Balance)

			// Get Bridge Info By DestAddr
			deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
			assert.NoError(t, err)
			t.Log("Deposits 2: ", deposits)
			// Check globalExitRoot
			globalExitRoot4, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
			assert.NoError(t, err)
			t.Logf("Global3 %+v: ", globalExitRoot3)
			t.Logf("Global4 %+v: ", globalExitRoot4)
			assert.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
			assert.Equal(t, common.HexToHash("0x573768af52d1354a7b83fb784ecbacecf8fead6ad49f25af8909a35b0a7bba05"), globalExitRoot3.ExitRoots[0])
			// Check L1 funds
			balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
			assert.NoError(t, err)
			assert.Equal(t, 0, big.NewInt(0).Cmp(balance))
			// Get the claim data
			smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
			assert.NoError(t, err)
			t.Log("smt2: ", smtProof)
			// Claim funds in L1
			if len(deposits) > 0 {
				err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
				assert.NoError(t, err)
			}
			// Check L1 funds to see if the amount has been increased
			balance, err = opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
			assert.NoError(t, err)
			assert.Equal(t, big.NewInt(1000000000000000000), balance)
			// Check L2 funds to see that the amount has been reduced
			balance, err = opsman.CheckAccountBalance(ctx, operations.L2, &destAddr)
			assert.NoError(t, err)
			assert.Equal(t, big.NewInt(9000000000000000000), balance)
		})
	}

	t.Run("Reversal ERC20 Test", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		assert.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)

		// Send L1 deposit
		var destNetwork uint32 = 0
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", operations.L2)
		assert.NoError(t, err)
		//Mint tokens
		err = opsman.MintERC20(ctx, tokenAddr, amount, operations.L2)
		assert.NoError(t, err)
		//Check balance
		origAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		balance, err := opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &origAddr)
		assert.NoError(t, err)
		t.Log("Token balance: ", balance, ". tokenaddress: ", tokenAddr, ". account: ", origAddr)
		destAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		amount = new(big.Int).SetUint64(1000000000000000000)
		err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
		assert.NoError(t, err)
		t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		assert.Equal(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.NotEqual(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		// Check L2 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &origAddr)
		assert.NoError(t, err)
		assert.Equal(t, 0, balance.Cmp(big.NewInt(9000000000000000000)))
		t.Log("Deposits: ", deposits)
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		// Claim funds in L1
		t.Logf("Deposits: %+v", deposits)
		for _, s := range smtProof {
			t.Log("smt: ", hex.EncodeToString(s[:]))
		}
		t.Logf("globalExitRoot: %+v", globaExitRoot)
		if len(deposits) > 0 {
			err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
			assert.NoError(t, err)
		}
		// Wait for sync token_wrapped event.
		time.Sleep(5 * time.Second)

		// Get tokenWrappedAddr
		t.Log("token Address:", tokenAddr)
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 1, tokenAddr, true)
		assert.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance2, err := opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("balance l1 account after claim funds: ", balance2)
		assert.NotEqual(t, balance, balance2)
		assert.Equal(t, amount, balance2)

		// Check globalExitRoot
		globalExitRoot3, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
		assert.NoError(t, err)
		// Send L2 Deposit to withdraw the some funds
		destNetwork = 1
		amount = new(big.Int).SetUint64(600000000000000000)
		err = opsman.SendL1Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		t.Log("Deposits 2: ", deposits)
		// Force to propose a new batch
		// err = opsman.ForceBatchProposal(ctx)
		assert.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
		assert.NoError(t, err)
		t.Logf("Global3 %+v: ", globalExitRoot3)
		t.Logf("Global4 %+v: ", globalExitRoot4)
		assert.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
		assert.NotEqual(t, globalExitRoot3.ExitRoots[0], globalExitRoot4.ExitRoots[0])
		assert.Equal(t, common.HexToHash("0x573768af52d1354a7b83fb784ecbacecf8fead6ad49f25af8909a35b0a7bba05"), globalExitRoot3.ExitRoots[0])
		assert.Equal(t, common.HexToHash("0xe9bab979a6f644ae7bd04f32f2d3d8b1043b29ff3db99c2f1344690ca3d193c4"), globalExitRoot3.ExitRoots[1])
		assert.Equal(t, common.HexToHash("0xb6a9f62f9c7998457c2f1fec780066903f201a000936641040651c9fa550f4b1"), globalExitRoot4.ExitRoots[0])
		// Check L2 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &destAddr)
		assert.NoError(t, err)
		t.Log("balance: ", balance)
		assert.Equal(t, 0, big.NewInt(0).Cmp(balance))
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		t.Log("smt2: ", smtProof)
		t.Log("deposits[0]: ", deposits[0])
		t.Log("globaExitRoot: ", globaExitRoot, globaExitRoot.GlobalExitRootNum)
		// Claim funds in L2
		if len(deposits) > 0 {
			err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
			assert.NoError(t, err)
		}
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenAddr, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(600000000000000000), balance)
		// Check L1 funds to see that the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, 0, big.NewInt(400000000000000000).Cmp(balance))
	})

	t.Run("ERC20 Test", func(t *testing.T) {
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		assert.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount := new(big.Int).SetUint64(10000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", operations.L1)
		assert.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, amount, operations.L1)
		assert.NoError(t, err)
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		balance, err := opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &origAddr)
		assert.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		assert.NoError(t, err)
		t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		assert.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		t.Log("Deposits: ", deposits)
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Force to propose a new batch
		// err = opsman.ForceBatchProposal(ctx)
		assert.NoError(t, err)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		// Claim funds in L2
		if len(deposits) > 0 {
			err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
			assert.NoError(t, err)
		}
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		assert.NoError(t, err)
		t.Log("TokenWrapped: ", tokenWrapped)
		// Check L2 funds to see if the amount has been increased
		balance2, err := opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance2)
		assert.Equal(t, amount, balance2)

		// Check globalExitRoot
		globalExitRoot3, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
		assert.NoError(t, err)
		// Send L2 Deposit to withdraw the some funds
		destNetwork = 0
		amount = new(big.Int).SetUint64(8000000000000000000)
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		t.Log("Deposits 2: ", deposits)
		// Check globalExitRoot
		globalExitRoot4, err := opsman.GetCurrentGlobalExitRootSynced(ctx)
		assert.NoError(t, err)
		t.Logf("Global3 %+v: ", globalExitRoot3)
		t.Logf("Global4 %+v: ", globalExitRoot4)
		assert.NotEqual(t, globalExitRoot3.GlobalExitRootNum, globalExitRoot4.GlobalExitRootNum)
		assert.Equal(t, common.HexToHash("0x7b6d271ef15fc512b8c3993a75206a18001ac49c7c4060464767826a3c8e888f"), globalExitRoot3.ExitRoots[0])
		assert.Equal(t, common.HexToHash("0xe9bab979a6f644ae7bd04f32f2d3d8b1043b29ff3db99c2f1344690ca3d193c4"), globalExitRoot3.ExitRoots[1])
		assert.Equal(t, common.HexToHash("0x7b6d271ef15fc512b8c3993a75206a18001ac49c7c4060464767826a3c8e888f"), globalExitRoot4.ExitRoots[0])
		// Check L1 funds
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, 0, big.NewInt(0).Cmp(balance))
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		t.Log("smt3: ", smtProof)
		// Claim funds in L1
		if len(deposits) > 0 {
			err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
			assert.NoError(t, err)
		}
		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L1, tokenAddr, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(8000000000000000000), balance)
		// Check L2 funds to see that the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, operations.L2, tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(2000000000000000000), balance)
	})
	t.Run("Multi deposits tests", func(t *testing.T) {
		/*
			1. Do 3 deposits/bridges
			2. Force a new batch proposal to sync the globalexitroot in L2
			3. Do 2 more deposits (without proposing a new batch)
			4. Claim the firs 3 deposits in L2
		*/

		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		assert.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount1 := new(big.Int).SetUint64(1000000000000000000)
		amount2 := new(big.Int).SetUint64(2000000000000000000)
		amount3 := new(big.Int).SetUint64(3000000000000000000)
		totAmount := new(big.Int).SetUint64(9000000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", "l1")
		assert.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, totAmount, "l1")
		assert.NoError(t, err)
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		balance, err := opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		assert.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		// First deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Second deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount2, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Third deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount3, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		assert.NoError(t, err)
		t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		assert.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.Equal(t, common.HexToHash("0x84202ff0ffb93e0b58c5338bd8e08319556a6b0f217c34a1c8afb1277e608c67"), globalExitRoot2.ExitRoots[0])
		assert.Equal(t, common.HexToHash("0x90d2f2daddea67d339cf1a35677080b390a8fcda3f492c4d2b487449af1babe7"), globalExitRoot2.ExitRoots[1])
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		t.Log("Deposits: ", deposits)
		t.Log("Before getClaimData: ", deposits[2].NetworkId, deposits[2].DepositCnt)
		// Force to propose a new batch
		// err = opsman.ForceBatchProposal(ctx)
		assert.NoError(t, err)
		// Fourth deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &origAddr)
		assert.NoError(t, err)
		// Fifth deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount2, destNetwork, &origAddr)
		assert.NoError(t, err)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[2].NetworkId), uint(deposits[2].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[2], smtProof, globaExitRoot)
		assert.NoError(t, err)
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		assert.NoError(t, err)
		t.Log("TokenWrapped: ", tokenWrapped)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		assert.Equal(t, amount1, balance)

		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[1].NetworkId), uint(deposits[1].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[1], smtProof, globaExitRoot)
		assert.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		assert.Equal(t, new(big.Int).Add(amount1, amount2), balance)

		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		assert.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		assert.Equal(t, new(big.Int).SetUint64(6000000000000000000), balance)
	})
	t.Run("L1-L2 bridge tests", func(t *testing.T) {
		/*
			1. Bridge from L1 to L2.
			2. Force a new batch proposal doing a bridge from L2 to L1
			3. Claim the deposits in both layers
		*/
		// Check initial globalExitRoot.
		globalExitRootSMC, err := opsman.GetCurrentGlobalExitRootFromSmc(ctx)
		assert.NoError(t, err)
		t.Logf("initial globalExitRootSMC: %+v,", globalExitRootSMC)
		// Send L1 deposit
		var destNetwork uint32 = 1
		amount1 := new(big.Int).SetUint64(1000000000000000000)
		totAmount := new(big.Int).SetUint64(3500000000000000000)
		tokenAddr, _, err := opsman.DeployERC20(ctx, "A COIN", "ACO", "l1")
		assert.NoError(t, err)
		err = opsman.MintERC20(ctx, tokenAddr, totAmount, "l1")
		assert.NoError(t, err)
		origAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		balance, err := opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		assert.NoError(t, err)
		t.Log("Init account balance l1: ", balance)
		destAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
		// First deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Force to propose a new batch
		// err = opsman.ForceBatchProposal(ctx)
		assert.NoError(t, err)
		// Get Bridge Info By DestAddr
		deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		t.Log("Deposits: ", deposits)
		t.Log("Before getClaimData: ", deposits[0].NetworkId, deposits[0].DepositCnt)
		// Get the claim data
		smtProof, globaExitRoot, err := opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		assert.NoError(t, err)
		tokenWrapped, err := opsman.GetTokenWrapped(ctx, 0, tokenAddr, false)
		assert.NoError(t, err)
		balance2, err := opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Init account balance l2: ", balance2)

		// Second deposit
		err = opsman.SendL1Deposit(ctx, tokenAddr, amount1, destNetwork, &destAddr)
		assert.NoError(t, err)
		// Check globalExitRoot
		globalExitRoot2, err := opsman.GetLatestGlobalExitRootFromL1(ctx)
		assert.NoError(t, err)
		t.Logf("globalExitRoot.GlobalExitRootNum: %d, globalExitRoot2.GlobalExitRootNum: %d", globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		assert.NotEqual(t, globalExitRootSMC.GlobalExitRootNum, globalExitRoot2.GlobalExitRootNum)
		t.Logf("globalExitRootSMC.mainnet: %v, globalExitRoot2.mainnet: %v", globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		assert.NotEqual(t, globalExitRootSMC.ExitRoots[0], globalExitRoot2.ExitRoots[0])
		t.Logf("globalExitRootSMC.rollup: %v, globalExitRoot2.rollup: %v", globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.Equal(t, globalExitRootSMC.ExitRoots[1], globalExitRoot2.ExitRoots[1])
		assert.Equal(t, common.HexToHash("0x2442d14fe86e87dc21f839680c0cbe31ea2e0872886f24617213345deae20048"), globalExitRoot2.ExitRoots[0])
		assert.Equal(t, common.HexToHash("0xd8076d8a9fbc049fd31ad074aaad7c90bd52407bb70f0d25c6bf4e4646bf0a84"), globalExitRoot2.ExitRoots[1])

		destNetwork = 0
		amount4 := new(big.Int).SetUint64(500000000000000000)
		// L2 deposit
		err = opsman.SendL2Deposit(ctx, tokenWrapped.WrappedTokenAddress, amount4, destNetwork, &origAddr)
		assert.NoError(t, err)
		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &origAddr)
		assert.NoError(t, err)
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		// Claim funds in L1
		err = opsman.SendL1Claim(ctx, deposits[0], smtProof, globaExitRoot)
		assert.NoError(t, err)
		// Check L2 funds to see if the amount has been reduced
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(500000000000000000), balance)
		// Check L1 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l1", tokenAddr, &origAddr)
		assert.NoError(t, err)
		assert.Equal(t, big.NewInt(2000000000000000000), balance)

		deposits, err = opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
		assert.NoError(t, err)
		// Get the claim data
		smtProof, globaExitRoot, err = opsman.GetClaimData(uint(deposits[0].NetworkId), uint(deposits[0].DepositCnt))
		assert.NoError(t, err)
		for _, s := range smtProof {
			t.Log("smt: 0x" + hex.EncodeToString(s[:]))
		}
		t.Log("globalExitRoot:", globaExitRoot)
		t.Log("deposit: ", deposits[0])
		// Claim funds in L2
		err = opsman.SendL2Claim(ctx, deposits[0], smtProof, globaExitRoot)
		assert.NoError(t, err)
		// Check L2 funds to see if the amount has been increased
		balance, err = opsman.CheckAccountTokenBalance(ctx, "l2", tokenWrapped.WrappedTokenAddress, &destAddr)
		assert.NoError(t, err)
		t.Log("Balance tokenWrapped: ", balance)
		assert.Equal(t, new(big.Int).SetUint64(1500000000000000000), balance)
		assert.NoError(t, operations.Teardown())
	})
}
