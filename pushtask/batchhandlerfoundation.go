package pushtask

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-node/hex"
	"github.com/0xPolygonHermez/zkevm-node/jsonrpc/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

const (
	queryLatestCommitBatchNumMethod = "zkevm_virtualBatchNumber" //nolint:gosec
	queryLatestVerifyBatchNumMethod = "zkevm_verifiedBatchNumber"
	secondsPreMinute                = 60
)

// BatchInfo just simple info, can get more from https://okg-block.larksuite.com/wiki/DqM0wLcm1i5fztk5iKAu2Tw6spg
type BatchInfo struct {
	Number string   `json:"number"`
	Blocks []string `json:"blocks"`
}

func QueryLatestCommitBatch(rpcUrl string) (uint64, error) {
	return queryLatestBatchNum(rpcUrl, queryLatestCommitBatchNumMethod)
}

func QueryLatestVerifyBatch(rpcUrl string) (uint64, error) {
	return queryLatestBatchNum(rpcUrl, queryLatestVerifyBatchNumMethod)
}

func queryLatestBatchNum(rpcUrl string, methodName string) (uint64, error) {
	response, err := client.JSONRPCCall(rpcUrl, methodName)
	if err != nil {
		log.Errorf("query for %v error: %v", methodName, err)
		return 0, errors.Wrap(err, fmt.Sprintf("query %v error", methodName))
	}

	if response.Error != nil {
		log.Errorf("query for %v, back failed data, %v, %v", methodName, response.Error.Code, response.Error.Message)
		return 0, fmt.Errorf("query %v failed", methodName)
	}

	var result string
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		log.Errorf("query for %v, parse json error: %v", methodName, err)
		return 0, errors.Wrap(err, fmt.Sprintf("query %v, parse json error", methodName))
	}

	bigBatchNumber := hex.DecodeBig(result)
	latestBatchNum := bigBatchNumber.Uint64()
	return latestBatchNum, nil
}

func QueryMaxBlockHashByBatchNum(rpcUrl string, batchNum uint64) (string, error) {
	blocks, err := queryBlockHashListByBatchNum(rpcUrl, batchNum)
	if err != nil {
		log.Errorf("query for %v, batch: %v, error: %v", "zkevm_getBatchByNumber", batchNum, err)
		return "", err
	}
	if len(blocks) == 0 {
		log.Errorf("query for %v, blocks is empty, batch num: %v", "zkevm_getBatchByNumber", batchNum)
		return "", nil
	}
	return blocks[len(blocks)-1], nil
}

func queryBlockHashListByBatchNum(rpcUrl string, batchNum uint64) ([]string, error) {
	response, err := client.JSONRPCCall(rpcUrl, "zkevm_getBatchByNumber", batchNum)
	if err != nil {
		log.Errorf("query for %v error: %v", "zkevm_getBatchByNumber", err)
		return nil, errors.Wrap(err, "query zkevm_getBatchByNumber error")
	}

	if response.Error != nil {
		log.Errorf("query for zkevm_getBatchByNumber failed, %v, %v", response.Error.Code, response.Error.Message)
		return nil, fmt.Errorf("query zkevm_getBatchByNumber failed")
	}

	var result BatchInfo
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		log.Errorf("query for %v, parse json error: %v", "zkevm_getBatchByNumber", err)
		return nil, errors.Wrap(err, "query zkevm_getBatchByNumber, parse json error")
	}
	return result.Blocks, nil
}

func QueryBlockNumByBlockHash(ctx context.Context, client *ethclient.Client, blockHash string) (uint64, error) {
	block, err := client.BlockByHash(ctx, common.HexToHash(blockHash))
	if err != nil {
		log.Errorf("query for blockByHash, block hash: %v, error: %v", blockHash, err)
		return 0, errors.Wrap(err, "query blockByHash error")
	}
	return block.Number().Uint64(), nil
}
