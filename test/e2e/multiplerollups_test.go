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
		mainnetID uint32 = 0
		rollupID1 uint32 = 1
	)
	ctx, opsman, err := getOpsman("http://localhost:8123", common.Address{})
	require.NoError(t, err)
	l1TokenAddr, _, err := opsman.DeployERC20(ctx, "CREATED ON L1", "CL1", operations.L1)
	require.NoError(t, err)
	err = opsman.MintERC20(ctx, l1TokenAddr, big.NewInt(999999999999999999), operations.L1)
	require.NoError(t, err)

	log.Info("L1 -> RollupID1 eth bridge")
	bridge(t, ctx, opsman, bridgeData{
		originNet:       mainnetID,
		destNet:         rollupID1,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("RollupID1 -> L1 eth bridge")
	bridge(t, ctx, opsman, bridgeData{
		originNet:       1,
		destNet:         0,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -> RollupID1 token from L1 bridge")
	bridge(t, ctx, opsman, bridgeData{
		originNet:       mainnetID,
		destNet:         rollupID1,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("RollupID1 -> L1 token from L1 bridge")
	bridge(t, ctx, opsman, bridgeData{
		originNet:       1,
		destNet:         0,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})
}

type bridgeData struct {
	originNet       uint32
	destNet         uint32
	originTokenNet  uint32
	originTokenAddr common.Address
	amount          *big.Int
}

func bridge(
	t *testing.T,
	ctx context.Context,
	opsman *operations.Manager,
	bd bridgeData,
) {
	// Sanity check that opsman support involved networks
	rID, err := opsman.GetRollupID()
	require.NoError(t, err)
	require.False(t, bd.originNet != 0 && bd.originNet != rID, "opsman deosnt support all the networks involved")
	require.False(t, bd.destNet != 0 && bd.destNet != rID, "opsman deosnt support all the networks involved")

	var (
		// This addressess are hardcoded on opsman. Would be nice to make it more flexible
		// to be able to operate multiple accounts
		originAddr, destAddr common.Address
		l1Addr               = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		l2Addr               = common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	)
	if bd.originNet == 0 {
		originAddr = l1Addr
	} else {
		originAddr = l2Addr
	}
	if bd.destNet == 0 {
		destAddr = l1Addr
	} else {
		destAddr = l2Addr
	}
	initialL1Balance, initialL2Balance, err := opsman.GetBalances(
		ctx,
		bd.originTokenNet,
		bd.originTokenAddr,
		originAddr,
		destAddr,
	)
	require.NoError(t, err)
	log.Debugf(
		"initial balance on L1: %d, initial balance on L2: %d",
		bd.destNet, initialL1Balance.Int64(), initialL2Balance.Int64(),
	)
	if bd.originNet == 0 {
		tokenAddr, err := opsman.GetTokenAddr(operations.L1, bd.originTokenNet, bd.originTokenAddr)
		require.NoError(t, err)
		log.Debugf(
			"depositing %d tokens of addr %s on L1 to %d",
			bd.amount.Uint64(), tokenAddr, bd.destNet,
		)
		err = opsman.SendL1Deposit(ctx, tokenAddr, bd.amount, bd.destNet, &destAddr)
		require.NoError(t, err)
	} else {
		tokenAddr, err := opsman.GetTokenAddr(operations.L2, bd.originTokenNet, bd.originTokenAddr)
		require.NoError(t, err)
		log.Debugf(
			"depositing %d tokens of addr %s on Rollup %d to Network %d",
			bd.amount.Uint64(), tokenAddr, bd.destNet,
		)
		err = opsman.SendL2Deposit(ctx, tokenAddr, bd.amount, bd.destNet, &destAddr)
		require.NoError(t, err)
	}
	log.Debug("deposit sent")

	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	deposit := deposits[0]

	if bd.originNet == 0 {
		log.Debug("waiting for claim tx to be sent on behalf of the user by bridges service...")
		err = opsman.CheckL2Claim(ctx, deposit)
		require.NoError(t, err)
		log.Debug("deposit claimed on L2")
	} else {
		log.Debug("getting proof to perform claim from bridge service...")
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(
			ctx,
			uint(deposit.NetworkId),
			uint(deposit.DepositCnt),
		)
		require.NoError(t, err)
		log.Debug("sending claim tx to L1")
		err = opsman.SendL1Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
		log.Debug("claim sent")
	}

	afterClaimL1Balance, afterClaimL2Balance, err := opsman.GetBalances(
		ctx,
		bd.originTokenNet,
		bd.originTokenAddr,
		originAddr,
		destAddr,
	)
	require.NoError(t, err)
	log.Debugf(
		"deosit claimed on network %d. final balance on L1: %d, final balance on L2: %d",
		bd.originNet, afterClaimL1Balance.Int64(), afterClaimL2Balance.Int64(),
	)
	// It's hard to get the expected balance after the process due to fees on L1/L2.
	// TODO: refactor SendL1Deposit / SendL2Deposit / SendL1Claim and to return the tx receipt or whatever so we can check the gas used and the gas price
	require.NotEqual(t, 0, initialL1Balance.Cmp(afterClaimL1Balance))
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
