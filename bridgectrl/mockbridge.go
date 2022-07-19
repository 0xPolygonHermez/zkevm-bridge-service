package bridgectrl

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/db/pgstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/test/vectors"
	"github.com/ethereum/go-ethereum/common"
)

// MockBridgeCtrl prepares mock data in the bridge service
func MockBridgeCtrl(store *pgstorage.PostgresStorage) (*BridgeController, error) {
	data, err := os.ReadFile("test/vectors/src/block-raw.json")
	if err != nil {
		return nil, err
	}

	var testBlockVectors []vectors.BlockVectorRaw
	err = json.Unmarshal(data, &testBlockVectors)
	if err != nil {
		return nil, err
	}

	data, err = os.ReadFile("test/vectors/src/deposit-raw.json")
	if err != nil {
		return nil, err
	}

	var testDepositVectors []vectors.DepositVectorRaw
	err = json.Unmarshal(data, &testDepositVectors)
	if err != nil {
		return nil, err
	}

	data, err = os.ReadFile("test/vectors/src/claim-raw.json")
	if err != nil {
		return nil, err
	}

	var testClaimVectors []vectors.ClaimVectorRaw
	err = json.Unmarshal(data, &testClaimVectors)
	if err != nil {
		return nil, err
	}

	btCfg := Config{
		Height: uint8(32), //nolint:gomnd
		Store:  "postgres",
	}

	bt, err := NewBridgeController(btCfg, []uint{0, 1000, 1001, 1002}, store, store)
	if err != nil {
		return nil, err
	}

	for i, testBlockVector := range testBlockVectors {
		id, err := store.AddBlock(context.TODO(), &etherman.Block{
			BlockNumber: testBlockVector.BlockNumber,
			BlockHash:   common.HexToHash(testBlockVector.BlockHash),
			ParentHash:  common.HexToHash(testBlockVector.ParentHash),
		}, nil)
		if err != nil {
			return nil, err
		}

		batch := etherman.Batch{
			Coinbase:       common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fe"),
			GlobalExitRoot: common.HexToHash("0x30e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fe"),
			BatchNumber:    uint64(i + 1),
			BatchL2Data:    []byte{},
			Timestamp:      time.Time{},
		}
		err = store.AddBatch(context.TODO(), &batch, nil)
		if err != nil {
			return nil, err
		}

		amount, _ := new(big.Int).SetString(testDepositVectors[i].Amount, 10) //nolint:gomnd
		deposit := &etherman.Deposit{
			OriginalNetwork:    testDepositVectors[i].OriginalNetwork,
			TokenAddress:       common.HexToAddress(testDepositVectors[i].TokenAddress),
			Amount:             amount,
			DestinationNetwork: testDepositVectors[i].DestinationNetwork,
			DestinationAddress: common.HexToAddress(testDepositVectors[i].DestinationAddress),
			DepositCount:       uint(i + 1),
			BlockID:            id,
			NetworkID:          testDepositVectors[i].OriginalNetwork,
			BlockNumber:        0,
			Metadata:           common.FromHex(testDepositVectors[i].Metadata),
		}
		err = store.AddDeposit(context.TODO(), deposit, nil)
		if err != nil {
			return nil, err
		}

		amount, _ = new(big.Int).SetString(testClaimVectors[i].Amount, 10) //nolint:gomnd
		err = store.AddClaim(context.TODO(), &etherman.Claim{
			Index:              testClaimVectors[i].Index,
			OriginalNetwork:    testClaimVectors[i].OriginalNetwork,
			Token:              common.HexToAddress(testClaimVectors[i].Token),
			Amount:             amount,
			NetworkID:          testClaimVectors[i].DestinationNetwork,
			DestinationAddress: common.HexToAddress(testClaimVectors[i].DestinationAddress),
			BlockID:            id,
			BlockNumber:        testClaimVectors[i].BlockNumber,
		}, nil)
		if err != nil {
			return nil, err
		}

		err = bt.MockAddDeposit(deposit)
		if err != nil {
			return nil, err
		}
		err = store.AddGlobalExitRoot(context.TODO(), &etherman.GlobalExitRoot{
			BlockNumber:       0,
			GlobalExitRootNum: big.NewInt(int64(i)),
			ExitRoots:         []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
			BlockID:           id,
		}, nil)
		if err != nil {
			return nil, err
		}
	}
	err = store.AddTokenWrapped(context.TODO(), &etherman.TokenWrapped{
		OriginalNetwork:      0,
		OriginalTokenAddress: common.HexToAddress("0x617b3a3528F9cDd6630fd3301B9c8911F7Bf063D"),
		WrappedTokenAddress:  common.HexToAddress("0xC2716D3537ECA4B318E60F3D7D6A48714F1F3335"),
		BlockID:              1,
		BlockNumber:          1,
		NetworkID:            1000, //nolint:gomnd
	}, nil)
	if err != nil {
		return nil, err
	}

	return bt, nil
}
