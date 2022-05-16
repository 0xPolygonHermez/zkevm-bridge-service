package bridgectrl

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
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
			BlockNumber:     testBlockVector.BlockNumber,
			BlockHash:       common.HexToHash(testBlockVector.BlockHash),
			ParentHash:      common.HexToHash(testBlockVector.ParentHash),
			Batches:         []etherman.Batch{},
			Deposits:        []etherman.Deposit{},
			GlobalExitRoots: []etherman.GlobalExitRoot{},
			Claims:          []etherman.Claim{},
			Tokens:          []etherman.TokenWrapped{},
		})
		if err != nil {
			return nil, err
		}

		head := types.Header{
			TxHash:     common.Hash{},
			Difficulty: big.NewInt(0),
			Number:     new(big.Int).SetUint64(uint64(i + 1)),
		}
		batch := etherman.Batch{
			BlockNumber:    1,
			Sequencer:      common.HexToAddress("0x29e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fe"),
			ChainID:        big.NewInt(100),
			GlobalExitRoot: common.HexToHash("0x30e885edaf8e4b51e1d2e05f9da28161d2fb4f6b1d53827d9b80a23cf2d7d9fe"),
			Header:         &head,
			ReceivedAt:     time.Now(),
			BlockID:        id,
			NetworkID:      testDepositVectors[i].OriginalNetwork,
		}
		err = store.AddBatch(context.TODO(), &batch)
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
		}
		err = store.AddDeposit(context.TODO(), deposit)
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
		})
		if err != nil {
			return nil, err
		}

		err = bt.MockAddDeposit(deposit)
		if err != nil {
			return nil, err
		}
		err = store.AddExitRoot(context.TODO(), &etherman.GlobalExitRoot{
			BlockNumber:         0,
			GlobalExitRootNum:   big.NewInt(int64(i)),
			GlobalExitRootL2Num: big.NewInt(int64(i)),
			ExitRoots:           []common.Hash{common.BytesToHash(bt.exitTrees[0].root[:]), common.BytesToHash(bt.exitTrees[1].root[:])},
			BlockID:             id,
		})
		if err != nil {
			return nil, err
		}
	}
	err = store.AddTokenWrapped(context.TODO(), &etherman.TokenWrapped{
		OriginalNetwork:      1,
		OriginalTokenAddress: common.HexToAddress("0x0EF3B0BC8D6313AB7DC03CF7225C872071BE1E6D"),
		WrappedTokenAddress:  common.HexToAddress("0xC2716D3537ECA4B318E60F3D7D6A48714F1F3335"),
		BlockID:              1,
		BlockNumber:          1,
		NetworkID:            0,
	})
	if err != nil {
		return nil, err
	}

	return bt, nil
}
