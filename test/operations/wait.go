package operations

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	ops "github.com/0xPolygonHermez/zkevm-node/test/operations"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	defaultInterval = 1 * time.Second
	defaultDeadline = 60 * time.Second
)

func poll(interval, deadline time.Duration, condition ops.ConditionFunc) error {
	return ops.Poll(interval, deadline, condition)
}

// WaitRestHealthy waits for a rest endpoint to be ready
func WaitRestHealthy(address string) error {
	return poll(defaultInterval, defaultDeadline, func() (bool, error) {
		return restHealthyCondition(address)
	})
}

func restHealthyCondition(address string) (bool, error) {
	resp, err := http.Get(address + "/healthz")

	return resp.StatusCode == http.StatusOK, err
}

// WaitGRPCHealthy waits for a gRPC endpoint to be responding according to the
// health standard in package grpc.health.v1
func WaitGRPCHealthy(address string) error {
	return ops.WaitGRPCHealthy(address)
}

func networkUpCondition() (bool, error) {
	return ops.NodeUpCondition(l1NetworkURL)
}

func proverUpCondition() (bool, error) {
	return true, nil
	// return ops.ProverUpCondition()
}

func zkevmNodeUpCondition() (done bool, err error) {
	return ops.NodeUpCondition(l2NetworkURL)
}

func bridgeUpCondition() (done bool, err error) {
	res, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		// we allow connection errors to wait for the container up
		return false, nil
	}
	if res.Body != nil {
		defer func() {
			err = res.Body.Close()
		}()
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	r := struct {
		Status string
	}{}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return false, err
	}
	done = r.Status == "SERVING"
	return done, nil
}

// WaitTxToBeMined waits until a tx is mined or forged.
func WaitTxToBeMined(ctx context.Context, client *ethclient.Client, tx *types.Transaction, timeout time.Duration) error {
	return ops.WaitTxToBeMined(ctx, client, tx, timeout)
}
