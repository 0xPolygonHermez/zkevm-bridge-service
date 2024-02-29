package metrics

const (
	endpointMetrics = "/metrics"
)

// Metric names and labels
const (
	prefix = "x1_bridge_"

	prefixRequest        = prefix + "request_"
	metricRequestCount   = prefixRequest + "count"
	metricRequestLatency = prefixRequest + "latency"
	labelIsSuccess       = "is_success"
	labelMethod          = "method"
)
