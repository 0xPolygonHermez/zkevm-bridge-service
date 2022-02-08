package db

import (
	"context"
	"errors"

	"github.com/hermeznetwork/hermez-bridge/etherman"
)

var (
	// ErrNotFound is used when the object is not found
	ErrNotFound = errors.New("Not found")
)

type Storage interface {
	GetLastBlock(ctx context.Context) (*etherman.Block, error)
}
