package operations

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultInterval = 500 * time.Millisecond
	defaultDeadline = 30 * time.Second
)

// Wait handles polliing until conditions are met.
type Wait struct{}

// NewWait is the Wait constructor.
func NewWait() *Wait {
	return &Wait{}
}

type conditionFunc func() (done bool, err error)

// Poll retries the given condition with the given interval until it succeeds
// or the given deadline expires.
func (w *Wait) Poll(interval, deadline time.Duration, condition conditionFunc) error {
	timeout := time.After(deadline)
	tick := time.NewTicker(interval)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("condition not met after %s", deadline)
		case <-tick.C:
			ok, err := condition()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
}

// WaitGRPCHealthy waits for a gRPC endpoint to be responding according to the
// health standard in package grpc.health.v1
func WaitGRPCHealthy(address string) error {
	w := NewWait()
	return w.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
		return grpcHealthyCondition(address)
	})
}

// WaitRestHealthy waits for a rest enpoint to be ready
func WaitRestHealthy(address string) error {
	w := NewWait()
	return w.Poll(defaultInterval, defaultDeadline, func() (bool, error) {
		return restHealthyCondition(address)
	})
}

func restHealthyCondition(address string) (bool, error) {
	resp, err := http.Get(address + "/healthz")

	return resp.StatusCode == http.StatusOK, err
}

func grpcHealthyCondition(address string) (bool, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		// we allow connection errors to wait for the container up
		return false, nil
	}
	defer func() {
		err = conn.Close()
	}()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	state, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		// we allow connection errors to wait for the container up
		return false, nil
	}

	done := state.Status == grpc_health_v1.HealthCheckResponse_SERVING

	return done, nil
}
