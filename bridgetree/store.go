package bridgetree

import (
	"context"
)

// Store interface
type Store interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Set(ctx context.Context, key []byte, value []byte) error
}
