package vectors

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

// E2ETestCase holds the metadata needed to run a etherman test
type E2ETestVectors struct {
	// TODO Define the field of the e2e test vector
	ID  uint     `json:"id"`
	Txs []string `json:"txs"`
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
