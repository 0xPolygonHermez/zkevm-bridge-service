package utils

type contextKey string

const (
	CtxTraceID contextKey = "traceID"
)

const (
	TraceID    = "traceID"
	traceIDLen = 16
)

const (
	// Number of block confirmations need to wait for the transaction to be synced from L1 to L2
	L1TargetBlockConfirmations = 64
)
