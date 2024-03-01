package metrics

const (
	endpointMetrics = "/metrics"
)

// Metric types
const (
	typeGauge     = "gauge"
	typeCounter   = "counter"
	typeHistogram = "histogram"
)

// Metric names and labels
const (
	prefix = "bridge_"

	prefixRequest        = prefix + "request_"
	metricRequestCount   = prefixRequest + "count"
	metricRequestLatency = prefixRequest + "latency"
	labelIsSuccess       = "is_success"
	labelMethod          = "method"

	prefixOrder            = prefix + "order_"
	metricOrderCount       = prefixOrder + "count"
	metricOrderTotalAmount = prefixOrder + "total_amount"
	labelNetworkID         = "network_id"
	labelToken             = "token"
)
