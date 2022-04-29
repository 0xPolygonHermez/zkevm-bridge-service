package bridgectrl

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/test/vectors"
)

func init() {
	// Change dir to project root
	// This is important because we have relative paths to files containing test vectors
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

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

		amount, _ := new(big.Int).SetString(testDepositVectors[i].Amount, 10) //nolint:gomnd
		deposit := &etherman.Deposit{
			OriginalNetwork:    testDepositVectors[i].OriginalNetwork,
			TokenAddress:       common.HexToAddress(testDepositVectors[i].TokenAddress),
			Amount:             amount,
			DestinationNetwork: testDepositVectors[i].DestinationNetwork,
			DestinationAddress: common.HexToAddress(testDepositVectors[i].DestinationAddress),
			DepositCount:       uint(i + 1),
			BlockID:            id,
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
	}

	return bt, nil
}
