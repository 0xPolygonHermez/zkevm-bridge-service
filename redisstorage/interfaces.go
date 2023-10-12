package redisstorage

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
)

type RedisStorage interface {
	SetCoinPrice(ctx context.Context, prices []*pb.SymbolPrice) error
	GetCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error)
}
