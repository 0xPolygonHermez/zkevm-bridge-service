package operations

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	ops "github.com/hermeznetwork/hermez-core/test/operations"
)

const (
	defaultInterval = 2 * time.Second
	defaultDeadline = 45 * time.Second
)

func poll(interval, deadline time.Duration, condition ops.ConditionFunc) error {
	w := ops.NewWait()
	return w.Poll(interval, deadline, condition)
}

// WaitRestHealthy waits for a rest enpoint to be ready
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
	return ops.ProverUpCondition()
}

func coreUpCondition() (done bool, err error) {
	return ops.NodeUpCondition(l2NetworkURL)
}

func bridgeUpCondition() (done bool, err error) {
	res, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		return false, err
	}
	if res.Body != nil {
		defer func() {
			err = res.Body.Close()
		}()
	}
	body, err := ioutil.ReadAll(res.Body)
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

//WaitTxToBeMined waits until a tx is mined or forged
func WaitTxToBeMined(ctx context.Context, client *ethclient.Client, hash common.Hash, timeout time.Duration) error {
	w := ops.NewWait()
	return w.TxToBeMined(client, hash, timeout)
}
