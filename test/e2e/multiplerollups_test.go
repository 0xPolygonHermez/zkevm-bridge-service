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
		rollup1ID uint32 = 1
		rollup2ID uint32 = 2
	)
	ctx, opsman1, err := getOpsman("http://localhost:8123", "bridge_db_1", "8080", "9090")
	require.NoError(t, err)
	_, opsman2, err := getOpsman("http://localhost:8124", "bridge_db_2", "8081", "9091")
	require.NoError(t, err)

	// Fund L2 sequencer for rollup 2. This is super dirty, but have no better way to do this at the moment
	polAddr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	rollup2Sequencer := common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65")
	mintAmount, ok := big.NewInt(0).SetString("1000000000000000000000000", 10)
	require.True(t, ok)
	err = opsman2.MintPOL(ctx, polAddr, mintAmount, operations.L1)
	require.NoError(t, err)
	err = opsman2.ERC20Transfer(ctx, polAddr, rollup2Sequencer, mintAmount, operations.L1)
	require.NoError(t, err)

	// L1 and R1 interactions
	log.Info("L1 -- eth --> R1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R1 -- eth --> L1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	l1TokenAddr, _, err := opsman1.DeployERC20(ctx, "CREATED ON L1", "CL1", operations.L1)
	require.NoError(t, err)
	err = opsman1.MintERC20(ctx, l1TokenAddr, mintAmount, operations.L1)
	require.NoError(t, err)
	log.Info("L1 -- token from L1 --> R1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R1 -- token from L1 --> L1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	rollup1TokenAddr, _, err := opsman1.DeployERC20(ctx, "CREATED ON Rollup 1", "CR1", operations.L2)
	require.NoError(t, err)
	err = opsman1.MintERC20(ctx, rollup1TokenAddr, big.NewInt(999999999999999999), operations.L2)
	require.NoError(t, err)
	log.Info("R1 -- token from R1 --> L1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from R1 --> R1")
	bridge(t, ctx, opsman1, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	// L1 and R2 interactions
	nativeTokenR2 := opsman2.GetRollupNativeToken()
	err = opsman2.MintPOL(ctx, nativeTokenR2, mintAmount, operations.L1)
	require.NoError(t, err)
	log.Info("L1 -- native token @ R2 --> R2")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: nativeTokenR2,
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R2 -- native token @ R2 --> L1")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: nativeTokenR2,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- eth --> R2")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R2 -- wETH --> L1")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from L1 --> R2")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- token from L1 --> L1")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	rollup2TokenAddr, _, err := opsman2.DeployERC20(ctx, "CREATED ON Rollup 2", "CR2", operations.L2)
	require.NoError(t, err)
	err = opsman2.MintERC20(ctx, rollup2TokenAddr, mintAmount, operations.L2)
	require.NoError(t, err)
	log.Info("R2 -- token from R2 --> L1")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from R2 --> R2")
	bridge(t, ctx, opsman2, nil, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
		amount:          big.NewInt(42069),
	})

	// R1 and R2 interactions
	log.Info("R1 -- eth --> R2")
	bridge(t, ctx, opsman1, opsman2, bridgeData{
		originNet:       rollup1ID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- wETH --> R1")
	bridge(t, ctx, opsman2, opsman1, bridgeData{
		originNet:       rollup2ID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- token from L1 --> R2")
	bridge(t, ctx, opsman1, opsman2, bridgeData{
		originNet:       rollup1ID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- native token @ R2 --> R1")
	bridge(t, ctx, opsman2, opsman1, bridgeData{
		originNet:       rollup2ID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: nativeTokenR2,
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- native token @ R2 --> R2")
	bridge(t, ctx, opsman1, opsman2, bridgeData{
		originNet:       rollup1ID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: nativeTokenR2,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- token from L1 --> R1")
	bridge(t, ctx, opsman2, opsman1, bridgeData{
		originNet:       rollup2ID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- token from R1 --> R2")
	bridge(t, ctx, opsman1, opsman2, bridgeData{
		originNet:       rollup1ID,
		destNet:         rollup2ID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- token from R1 --> R1")
	bridge(t, ctx, opsman2, opsman1, bridgeData{
		originNet:       rollup2ID,
		destNet:         rollup1ID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- token from R2 --> R1")
	bridge(t, ctx, opsman2, opsman1, bridgeData{
		originNet:       rollup2ID,
		destNet:         rollup1ID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- token from R2 --> R2")
	bridge(t, ctx, opsman1, opsman2, bridgeData{
		originNet:       rollup1ID,
		destNet:         rollup2ID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
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
	// Only use in case of l2 to l2
	opsmanDest *operations.Manager,
	bd bridgeData,
) {
	isL2ToL2 := bd.originNet != 0 && bd.destNet != 0
	// Sanity check that opsman support involved networks
	rollupIDOpsman, err := opsman.GetRollupID()
	require.NoError(t, err)
	var rollupIDOpsmanDest uint32
	if isL2ToL2 {
		rollupIDOpsmanDest, err = opsmanDest.GetRollupID()
		require.NoError(t, err)
	} else {
		rollupIDOpsmanDest, err = opsman.GetRollupID()
		require.NoError(t, err)
	}
	require.False(
		t, bd.originNet != 0 && bd.originNet != rollupIDOpsman,
		"opsman deosnt support all the networks involved",
	)
	require.False(
		t, bd.destNet != 0 && bd.destNet != rollupIDOpsmanDest,
		"opsman deosnt support all the networks involved",
	)

	var (
		// This addresses are hardcoded on opsman. Would be nice to make it more flexible
		// to be able to operate multiple accounts
		destAddr common.Address
		l1Addr   = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		l2Addr   = common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	)
	if bd.destNet == 0 {
		destAddr = l1Addr
	} else {
		destAddr = l2Addr
	}

	// Fetching initial balances on origin and destination, so at the end of the test is possible to assert results
	getBalances := func() (origin, dest *big.Int) {
		if bd.originNet == 0 {
			origin, err = opsman.GetL1Balance(ctx, bd.originTokenNet, bd.originTokenAddr, l1Addr)
			require.NoError(t, err)
		} else {
			origin, err = opsman.GetL2Balance(ctx, bd.originTokenNet, bd.originTokenAddr, l2Addr)
			require.NoError(t, err)
		}
		if bd.destNet == 0 {
			dest, err = opsman.GetL1Balance(ctx, bd.originTokenNet, bd.originTokenAddr, l1Addr)
			require.NoError(t, err)
		} else if isL2ToL2 {
			dest, err = opsmanDest.GetL2Balance(ctx, bd.originTokenNet, bd.originTokenAddr, l2Addr)
			require.NoError(t, err)
		} else {
			dest, err = opsman.GetL2Balance(ctx, bd.originTokenNet, bd.originTokenAddr, l2Addr)
			require.NoError(t, err)
		}
		return origin, dest
	}
	originNetInitialBalance, destNetInitialBalance := getBalances()
	log.Debugf(
		"initial balance on network %d: %d, initial balance on network %d: %d",
		bd.originNet, originNetInitialBalance.Int64(), bd.destNet, destNetInitialBalance.Int64(),
	)

	// Deposit interaction
	if bd.originNet == 0 {
		tokenAddr, err := opsman.GetTokenAddr(operations.L1, bd.originTokenNet, bd.originTokenAddr)
		require.NoError(t, err)
		log.Debugf(
			"depositing %d tokens of addr %s on L1 to network %d",
			bd.amount.Uint64(), tokenAddr, bd.destNet,
		)
		err = opsman.SendL1Deposit(ctx, tokenAddr, bd.amount, bd.destNet, &destAddr)
		require.NoError(t, err)
	} else {
		// Don't need to care about opsman / opsmanDest as this interaction always happens on the origin.
		// So even if is l2 -> l2 opsman is used
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

	// Check deposit info from bridge service
	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	deposit := deposits[0]

	// Claim interaction
	if bd.originNet == 0 { // L1 => L2
		log.Debug("waiting for claim tx to be sent on behalf of the user by bridge service...")
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
		if bd.destNet == 0 { // L2 => L1
			log.Debug("sending claim tx to L1")
			err = opsman.SendL1Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
			require.NoError(t, err)
		} else { // L2 => L2
			// TODO: wait for destination L2 network to have the GER updated
			log.Debugf("sending claim tx to network %d", rollupIDOpsmanDest)
			err = opsmanDest.SendL2Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
			require.NoError(t, err)
		}
	}

	// Assert results
	originNetFinalBalance, destNetFInalBalance := getBalances()
	log.Debugf(
		"deosit claimed on network %d. final balance on network %d: %d, final balance on network %d: %d",
		bd.destNet, bd.originNet, originNetFinalBalance.Int64(), bd.destNet, destNetFInalBalance.Int64(),
	)
	// It's hard to get the expected balance after the process due to fees on L1/L2.
	// TODO: refactor SendL1Deposit / SendL2Deposit / SendL1Claim and to return the tx receipt or whatever so we can check the gas used and the gas price
	require.NotEqual(t, 0, originNetInitialBalance.Cmp(originNetFinalBalance))
	require.NotEqual(t, 0, destNetInitialBalance.Cmp(destNetFInalBalance))
}

func getOpsman(
	l2NetworkURL string,
	dbName string,
	bridgeServiceHTTPPort string,
	bridgeServiceGRPCPort string,
) (context.Context, *operations.Manager, error) {
	ctx := context.Background()
	opsCfg := &operations.Config{
		L1NetworkURL: "http://localhost:8545",
		L2NetworkURL: l2NetworkURL,
		Storage: db.Config{
			Database: "postgres",
			Name:     dbName,
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
			GRPCPort:         bridgeServiceGRPCPort,
			HTTPPort:         bridgeServiceHTTPPort,
			CacheSize:        100000,
			DefaultPageLimit: 25,
			MaxPageLimit:     100,
			BridgeVersion:    "v1",
		},
	}
	opsman, err := operations.NewManager(ctx, opsCfg)
	return ctx, opsman, err
}
