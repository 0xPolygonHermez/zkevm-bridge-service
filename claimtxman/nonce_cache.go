package claimtxman

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	cacheSize = 1000
)

type NonceCache struct {
	ctx context.Context
	// client is the ethereum client
	l2Node     *utils.Client
	nonceCache *lru.Cache[string, uint64]
}

func NewNonceCache(ctx context.Context, l2Node *utils.Client) (*NonceCache, error) {
	cache, err := lru.New[string, uint64](int(cacheSize))
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &NonceCache{
		ctx:        ctx,
		l2Node:     l2Node,
		nonceCache: cache,
	}, nil
}

func (tm *NonceCache) GetNextNonce(from common.Address) (uint64, error) {
	nonce, err := tm.l2Node.NonceAt(tm.ctx, from, nil)
	if err != nil {
		return 0, err
	}
	if tempNonce, found := tm.nonceCache.Get(from.Hex()); found {
		if tempNonce >= nonce {
			nonce = tempNonce + 1
		}
	}
	tm.nonceCache.Add(from.Hex(), nonce)
	return nonce, nil
}

func (tm *NonceCache) Remove(from string) {
	tm.nonceCache.Remove(from)
}
