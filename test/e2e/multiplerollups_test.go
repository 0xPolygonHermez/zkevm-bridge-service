package e2e

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl"
	"github.com/0xPolygonHermez/zkevm-bridge-service/db"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMultipleRollups(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const (
		rollupID1 uint32 = 1
	)
	var (
		// This addressess are hardcoded on opsman. Would be nice to make it more flexible
		// to be able to operate multiple accounts
		l1Addr = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		l2Addr = common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	)
	ctx, opsman, err := getOpsman("http://localhost:8123", common.Address{})
	require.NoError(t, err)
	nativeToken := common.Address{}
	hugeAmount := big.NewInt(999999999999999999)
	sendAmount := big.NewInt(999999999999999999)
	l1TokenAddr, _, err := opsman.DeployERC20(ctx, "CREATED ON L1", "CL1", operations.L1)
	require.NoError(t, err)
	err = opsman.MintERC20(ctx, l1TokenAddr, hugeAmount, operations.L1)
	require.NoError(t, err)

	log.Info("L1 -> RollupID1 eth bridge")
	l1Tol2(t, ctx, opsman, nativeToken, l2Addr, sendAmount, rollupID1)

	log.Info("RollupID1 -> L1 eth bridge")
	sendAmount = big.NewInt(42069)
	l2Tol1(t, ctx, opsman, nativeToken, l1Addr, sendAmount)

	log.Info("L1 -> RollupID1 token bridge")
	l1Tol2(t, ctx, opsman, l1TokenAddr, l2Addr, sendAmount, rollupID1)

	// log.Debug("RollupID1 -> L1 token bridge")
	// 	amount := new(big.Int).SetUint64(42069)
	// 	tokenAddr := common.Address{} // This means is eth
	// 	destAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
	// 	l2Tol1(t, ctx, opsman, tokenAddr, destAddr, amount)

}

func l1Tol2(
	t *testing.T,
	ctx context.Context,
	opsman *operations.Manager,
	tokenAddr,
	destAddr common.Address,
	amount *big.Int,
	destNetwork uint32,
) {
	origAddr := common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
	initialL1Balance, initialL2Balance, err := opsman.GetBalances(ctx, tokenAddr, origAddr, destAddr)
	require.NoError(t, err)
	log.Debugf(
		"sending deosit L1 to RollupID %d. initial balance on L1: %d, initial balance on L2: %d",
		destNetwork, initialL1Balance.Int64(), initialL2Balance.Int64(),
	)
	// TODO: it takes too long for L2 to update GER, try to tweak l2 node config to speed-up
	// operations/manager.go:721	WaitExitRootToBeSynced
	err = opsman.SendL1Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)
	log.Debug("deposit sent")

	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	log.Debug("waiting for claim tx to be sent on behalf of the user by bridges service...")
	err = opsman.CheckL2Claim(ctx, deposits[0])
	require.NoError(t, err)
	log.Debug("deposit claimed on L2")

	afterClaimL1Balance, afterClaimL2Balance, err := opsman.GetBalances(ctx, tokenAddr, origAddr, destAddr)
	require.NoError(t, err)
	log.Debugf(
		"deosit claimed on RollupID %d. final balance on L1: %d, final balance on L2: %d",
		destNetwork, afterClaimL1Balance.Int64(), afterClaimL2Balance.Int64(),
	)
	// It's hard to get the expected balance after the process due to fees on L1/L2.
	// TODO: refactor SendL1Deposit to return the tx receipt or whatever so we can check the gas used and the gas price
	require.NotEqual(t, 0, initialL1Balance.Cmp(afterClaimL1Balance))
	require.Equal(t, amount, afterClaimL2Balance.Sub(afterClaimL2Balance, initialL2Balance))
}

func l2Tol1(
	t *testing.T,
	ctx context.Context,
	opsman *operations.Manager,
	tokenAddr,
	destAddr common.Address,
	amount *big.Int,
) {
	origAddr := common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	initialL1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
	require.NoError(t, err)
	initialL2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &origAddr)
	require.NoError(t, err)
	origRID, err := opsman.GetRollupID()
	require.NoError(t, err)

	log.Debugf(
		"sending deposit from RollupID %d to L1. initial balance on L1: %d, initial balance on L2: %d",
		origRID, initialL1Balance.Int64(), initialL2Balance.Int64(),
	)
	var destNetwork uint32 = 0
	err = opsman.SendL2Deposit(ctx, tokenAddr, amount, destNetwork, &destAddr)
	require.NoError(t, err)
	log.Debug("deposit sent")

	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	deposit := deposits[0]

	log.Debug("getting proof to perform claim from bridge service...")
	smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposit.NetworkId), uint(deposit.DepositCnt))
	require.NoError(t, err)
	log.Debug("sending claim tx to L1")
	err = opsman.SendL1Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
	require.NoError(t, err)

	// Check that the amount has been deduced from L2 and increased on L1
	afterClaimL1Balance, err := opsman.CheckAccountBalance(ctx, operations.L1, &destAddr)
	require.NoError(t, err)
	afterClaimL2Balance, err := opsman.CheckAccountBalance(ctx, operations.L2, &origAddr)
	require.NoError(t, err)
	log.Debugf(
		"deosit claimed on L1. final balance on L1: %d, final balance on RollupID %d: %d",
		afterClaimL1Balance.Int64(), origRID, afterClaimL2Balance.Int64(),
	)
	// It's hard to get the expected balance after the process due to fees on L1/L2.
	// TODO: refactor SendL1Claim / SendL2Deposit to return the tx receipt or whatever so we can check the gas used and the gas price
	require.NotEqual(t, 0, afterClaimL1Balance.Cmp(initialL1Balance))
	require.NotEqual(t, 0, initialL2Balance.Cmp(afterClaimL2Balance))
}

func getOpsman(l2NetworkURL string, l2NativeToken common.Address) (context.Context, *operations.Manager, error) {
	ctx := context.Background()
	opsCfg := &operations.Config{
		L1NetworkURL: "http://localhost:8545",
		L2NetworkURL: l2NetworkURL,
		Storage: db.Config{
			Database: "postgres",
			Name:     "bridge_db_1",
			User:     "user",
			Password: "pass",
			Host:     "localhost",
			Port:     "5432",
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
		},
	}
	opsman, err := operations.NewManager(ctx, opsCfg)
	return ctx, opsman, err
}
