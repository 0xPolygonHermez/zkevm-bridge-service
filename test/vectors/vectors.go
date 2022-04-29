package vectors

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
)

// E2ETestVectors holds the metadata needed to run a etherman test
type E2ETestVectors struct {
	// TODO Define the field of the e2e test vector
	ID               uint            `json:"id"`
	BridgeDeployed   bool            `json:"bridgeDeployed"`
	ChainIDSequencer uint            `json:"chainIdSequencer"`
	DefaultChainID   uint            `json:"defaultChainId"`
	SequencerAddress common.Address  `json:"sequencerAddress"`
	SequencerPvtKey  string          `json:"sequencerPvtKey"`
	Genesis          []interface{}   `json:"genesis"`
	ExpectedOldRoot  common.Hash     `json:"expectedOldRoot"`
	Txs              []Tx            `json:"txs"`
	ExpectedNewRoot  common.Hash     `json:"expectedNewRoot"`
	ExpectedNewLeafs map[string]Leaf `json:"expectedNewLeafs"`
	BatchL2Data      string          `json:"batchL2Data"`
	GlobalExitRoot   common.Hash     `json:"globalExitRoot"`
	NewLocalExitRoot common.Hash     `json:"newLocalExitRoot"`
	InputHash        common.Hash     `json:"inputHash"`
	BatchHashData    common.Hash     `json:"batchHashData"`
	OldLocalExitRoot common.Hash     `json:"oldLocalExitRoot"`
	Timestamp        uint64          `json:"timestamp"`
}

// Leaf represents a mt leaf
type Leaf struct {
	Balance argBigInt         `json:"balance"`
	Nonce   argBigInt         `json:"nonce"`
	Storage map[string]string `json:"storage"`
}

// Tx represents a transactions that will be applied during the test
type Tx struct {
	ContractName string         `json:"contractName"`
	Function     string         `json:"function"`
	From         common.Address `json:"from"`
	To           common.Address `json:"to"`
	Nonce        uint64         `json:"nonce"`
	Value        argBigInt      `json:"value"`
	Params       []interface{}  `json:"params"`
	GasLimit     uint64         `json:"gasLimit"`
	GasPrice     argBigInt      `json:"gasPrice"`
	Data         string         `json:"data"`
	ChainID      uint64         `json:"chainId"`
	Reason       string         `json:"reason"`
	CustomRawTx  string         `json:"customRawTx"`
	RawTx        string         `json:"rawTx"`
}

// LoadE2ETestVectors loads the calldata-test-vector.json
func LoadE2ETestVectors(path string) ([]E2ETestVectors, error) {
	var testCases []E2ETestVectors
	jsonFile, err := os.Open(filepath.Clean(path))
	if err != nil {
		return testCases, err
	}
	defer func() { _ = jsonFile.Close() }()

	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return testCases, err
	}

	err = json.Unmarshal(bytes, &testCases)
	if err != nil {
		return testCases, err
	}

	return testCases, nil
}
