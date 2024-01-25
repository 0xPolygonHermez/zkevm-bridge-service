package utils

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
)

type contextKey string

const (
	CtxTraceID contextKey = "traceID"
)

const (
	TraceID    = "traceID"
	traceIDLen = 16
)

var (
	// L1TargetBlockConfirmations is the number of block confirmations need to wait for the transaction to be synced from L1 to L2
	L1TargetBlockConfirmations = apolloconfig.NewIntEntry[uint64]("l1TargetBlockConfirmations", 64) //nolint:gomnd
)
