package metrics

const (
	defaultMetricsEndpoint = "/metrics"
)

// Metric types
const (
	typeGauge     = "gauge"
	typeCounter   = "counter"
	typeHistogram = "histogram"
)

// Metric names and labels
const (
	prefix   = "xlayer_bridge_"
	labelEnv = "env"

	prefixRequest        = prefix + "request_"
	metricRequestCount   = prefixRequest + "count"
	metricRequestLatency = prefixRequest + "latency_ms"
	labelMethod          = "method"
	labelCode            = "code"
	labelErrMessage      = "msg"

	prefixOrder             = prefix + "order_"
	metricOrderCount        = prefixOrder + "count"
	metricOrderTotalAmount  = prefixOrder + "total_amount"
	metricOrderWaitTime     = prefixOrder + "wait_time_sec"
	labelNetworkID          = "network_id"
	labelLeafType           = "leaf_type"
	labelToken              = "token"
	labelTokenAddress       = "token_address"
	labelTokenOriginNetwork = "token_origin_network"
	labelDestNet            = "dest_net"

	prefixMonitoredTxs             = prefix + "monitored_txs_"
	metricMonitoredTxsPendingCount = prefixMonitoredTxs + "pending_count"
	metricMonitoredTxsResultCount  = prefixMonitoredTxs + "result_count"
	metricMonitoredTxsDuration     = prefixMonitoredTxs + "duration_sec"
	labelStatus                    = "status"

	prefixSynchronizer           = prefix + "synchronizer_"
	metricSynchronizerEventCount = prefixSynchronizer + "event_count"
	metricLastSyncedBlockNum     = prefixSynchronizer + "last_synced_block_num"
	metricLatestBlockNum         = prefixSynchronizer + "latest_block_num"
	labelEventType               = "type"
)
