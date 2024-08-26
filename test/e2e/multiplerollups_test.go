//go:build multiplerollups
// +build multiplerollups

package e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

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
		mainnetID uint = 0
		rollup1ID uint = 1
		rollup2ID uint = 2
	)
	ctx := context.Background()
	opsman1, err := operations.GetOpsman(ctx, "http://localhost:8123", "test_db", "8080", "9090", "5435", 1)
	require.NoError(t, err)
	opsman2, err := operations.GetOpsman(ctx, "http://localhost:8124", "test_db", "8080", "9090", "5435", 2)
	require.NoError(t, err)

	// Fund L2 sequencer for rollup 2. This is super dirty, but have no better way to do this at the moment
	polAddr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	rollup2Sequencer := common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65")
	polAmount, ok := big.NewInt(0).SetString("10000000000000000000000", 10)
	require.True(t, ok)
	err = opsman2.MintPOL(ctx, polAddr, polAmount, operations.L1)
	require.NoError(t, err)
	err = opsman2.ERC20Transfer(ctx, polAddr, rollup2Sequencer, polAmount, operations.L1)
	require.NoError(t, err)

	// L1 and R1 interactions
	log.Info("L1 -- eth --> R1")
	bridge(t, ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R1 -- eth --> L1")
	bridge(t, ctx, opsman1, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	l1TokenAddr, _, err := opsman1.DeployERC20(ctx, "CREATED ON L1", "CL1", operations.L1)
	require.NoError(t, err)
	err = opsman1.MintERC20(ctx, l1TokenAddr, big.NewInt(999999999999999999), operations.L1)
	require.NoError(t, err)
	log.Info("L1 -- token from L1 --> R1")
	bridge(t, ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- token from L1 --> L1")
	bridge(t, ctx, opsman1, bridgeData{
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
	bridge(t, ctx, opsman1, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from R1 --> R1")
	bridge(t, ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- eth --> R2")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R2 -- eth --> L1")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from L1 --> R2")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R2 -- token from L1 --> L1")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	rollup2TokenAddr, _, err := opsman2.DeployERC20(ctx, "CREATED ON Rollup 2", "CR2", operations.L2)
	require.NoError(t, err)
	err = opsman2.MintERC20(ctx, rollup2TokenAddr, big.NewInt(999999999999999999), operations.L2)
	require.NoError(t, err)
	log.Info("R2 -- token from R2 --> L1")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       rollup2ID,
		destNet:         mainnetID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from R2 --> R2")
	bridge(t, ctx, opsman2, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup2ID,
		originTokenNet:  rollup2ID,
		originTokenAddr: rollup2TokenAddr,
		amount:          big.NewInt(42069),
	})
}

type bridgeData struct {
	originNet       uint
	destNet         uint
	originTokenNet  uint
	originTokenAddr common.Address
	amount          *big.Int
}

func bridge(
	t *testing.T,
	ctx context.Context,
	opsman *operations.Manager,
	bd bridgeData,
) {
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
	initialL1Balance, initialL2Balance, err := opsman.GetBalances(ctx, uint32(bd.originTokenNet), bd.originTokenAddr, l1Addr, l2Addr)
	require.NoError(t, err)
	log.Debugf("initial balance on L1: %d, initial balance on L2: %d", initialL1Balance.Int64(), initialL2Balance.Int64())
	if bd.originNet == 0 {
		tokenAddr, err := opsman.GetTokenAddress(ctx, operations.L1, bd.originTokenNet, bd.originTokenAddr)
		require.NoError(t, err)
		log.Debugf("depositing %d tokens of addr %s on L1 to network %d", bd.amount.Uint64(), tokenAddr, bd.destNet)
		err = opsman.SendL1Deposit(ctx, tokenAddr, bd.amount, uint32(bd.destNet), &destAddr)
		require.NoError(t, err)
	} else {
		tokenAddr, err := opsman.GetTokenAddress(ctx, operations.L2, bd.originTokenNet, bd.originTokenAddr)
		require.NoError(t, err)
		log.Debugf("depositing %d tokens of addr %s on Rollup %d to Network %d", bd.amount.Uint64(), tokenAddr, bd.destNet)
		err = opsman.SendL2Deposit(ctx, tokenAddr, bd.amount, uint32(bd.destNet), &destAddr, operations.L2)
		require.NoError(t, err)
	}
	log.Debug("deposit sent")

	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	require.NoError(t, err)
	deposit := deposits[0]

	if bd.originNet == 0 {
		log.Debug("waiting for claim tx to be sent on behalf of the user by bridge service...")
		err = opsman.CheckClaim(ctx, deposit)
		require.NoError(t, err)
		log.Debug("deposit claimed on L2")
	} else {
		log.Debug("getting proof to perform claim from bridge service...")
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, uint(deposit.NetworkId), uint(deposit.DepositCnt))
		require.NoError(t, err)
		log.Debug("sending claim tx to L1")
		err = opsman.SendL1Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
		require.NoError(t, err)
		log.Debug("claim sent")
	}
	time.Sleep(2 * time.Second)

	afterClaimL1Balance, afterClaimL2Balance, err := opsman.GetBalances(ctx, uint32(bd.originTokenNet), bd.originTokenAddr, l1Addr, l2Addr)
	require.NoError(t, err)
	log.Debugf("deposit claimed on network %d. final balance on L1: %d, final balance on L2: %d", bd.originNet, afterClaimL1Balance.Int64(), afterClaimL2Balance.Int64())
	require.NotEqual(t, 0, initialL1Balance.Cmp(afterClaimL1Balance))
	require.NotEqual(t, 0, initialL2Balance.Cmp(afterClaimL2Balance))
}
