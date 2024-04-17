package metrics

import (
	"context"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

func initMetrics(c Config) {
	if !initialized {
		registerer = prometheus.DefaultRegisterer
		gauges = make(map[string]*prometheus.GaugeVec)
		counters = make(map[string]*prometheus.CounterVec)
		histograms = make(map[string]*prometheus.HistogramVec)
		initialized = true
	}

	constLabels := map[string]string{labelEnv: c.Env}

	registerCounter(prometheus.CounterOpts{Name: metricRequestCount, ConstLabels: constLabels}, labelMethod, labelIsSuccess)
	registerHistogram(prometheus.HistogramOpts{
		Name:        metricRequestLatency,
		ConstLabels: constLabels,
		Buckets:     []float64{1, 10, 25, 50, 100, 250, 500, 750, 1000, 2500, 5000, 10000},
	}, labelMethod, labelIsSuccess)
	registerCounter(prometheus.CounterOpts{Name: metricOrderCount, ConstLabels: constLabels}, labelNetworkID, labelLeafType, labelToken, labelTokenAddress, labelTokenOriginNetwork, labelDestNet)
	registerCounter(prometheus.CounterOpts{Name: metricOrderTotalAmount, ConstLabels: constLabels}, labelNetworkID, labelLeafType, labelToken, labelTokenAddress, labelTokenOriginNetwork, labelDestNet)
	registerHistogram(prometheus.HistogramOpts{
		Name:        metricOrderWaitTime,
		ConstLabels: constLabels,
		Buckets:     []float64{50, 100, 400, 700, 900, 1200, 2000, 3000, 4000, 5000, 10000},
	}, labelNetworkID, labelDestNet)
	registerGauge(prometheus.GaugeOpts{Name: metricMonitoredTxsPendingCount, ConstLabels: constLabels})
	registerCounter(prometheus.CounterOpts{Name: metricMonitoredTxsResultCount, ConstLabels: constLabels}, labelStatus)
	registerHistogram(prometheus.HistogramOpts{
		Name:        metricMonitoredTxsDuration,
		ConstLabels: constLabels,
		Buckets:     []float64{0.5, 1, 2.5, 5, 10, 20, 50, 80, 100, 500, 1000},
	})
	registerCounter(prometheus.CounterOpts{Name: metricSynchronizerEventCount, ConstLabels: constLabels}, labelNetworkID, labelEventType)
}

// RecordRequest increments the request count for the method
func RecordRequest(method string, isSuccess bool) {
	counterInc(metricRequestCount, map[string]string{labelMethod: method, labelIsSuccess: strconv.FormatBool(isSuccess)})
}

// RecordRequestLatency records the latency histogram in nanoseconds
func RecordRequestLatency(method string, latency time.Duration, isSuccess bool) {
	latencyMs := float64(latency) / float64(time.Millisecond)
	histogramObserve(metricRequestLatency, latencyMs, map[string]string{labelMethod: method, labelIsSuccess: strconv.FormatBool(isSuccess)})
}

// RecordOrder records one bridge order, increase the order count and add the amount to the total order amount
// networkID is the "from" network of the transaction
func RecordOrder(networkID, destNet, leafType, tokenOriginNetwork uint32, tokenAddress common.Address, amount *big.Int) {
	tokenSymbol := "unknown"
	decimals := uint64(0)
	if coinsCache := localcache.GetDefaultCache(); coinsCache != nil {
		coinInfo, err := coinsCache.GetCoinInfoByAddress(context.Background(), tokenOriginNetwork, tokenAddress)
		if err == nil {
			tokenSymbol = coinInfo.Symbol
			decimals = coinInfo.Decimals
		}
	}

	// This is inflated amount, e.g.: 1 ETH is stored as 1000000000000000000
	floatAmount, err := strconv.ParseFloat(amount.String(), 64) //nolint:gomnd
	if err != nil {
		log.Warnf("cannot convert [%v] to float", amount.String())
	}
	// Deflate the amount
	if decimals != 0 {
		floatAmount = floatAmount / math.Pow(10, float64(decimals)) //nolint:gomnd
	}

	labels := map[string]string{
		labelNetworkID:          strconv.Itoa(int(networkID)),
		labelDestNet:            strconv.Itoa(int(destNet)),
		labelLeafType:           strconv.Itoa(int(leafType)),
		labelToken:              tokenSymbol,
		labelTokenAddress:       tokenAddress.String(),
		labelTokenOriginNetwork: strconv.Itoa(int(tokenOriginNetwork)),
	}

	counterInc(metricOrderCount, labels)
	counterAdd(metricOrderTotalAmount, floatAmount, labels)
}

// RecordOrderWaitTime records the waiting time (seconds) of a bridge order, from order creation (deposit time) to ready_for_claim time
func RecordOrderWaitTime(networkID, destNet uint32, dur time.Duration) {
	histogramObserve(metricOrderWaitTime, float64(dur)/float64(time.Second), map[string]string{labelNetworkID: strconv.Itoa(int(networkID)), labelDestNet: strconv.Itoa(int(destNet))})
}

// RecordPendingMonitoredTxsCount records the current number of pending monitored txs (status == "created")
func RecordPendingMonitoredTxsCount(count int) {
	gaugeSet(metricMonitoredTxsPendingCount, float64(count), map[string]string{})
}

// RecordMonitoredTxsResult records the final result of a monitored tx (confirmed or failed)
func RecordMonitoredTxsResult(status string) {
	counterInc(metricMonitoredTxsResultCount, map[string]string{labelStatus: status})
}

// RecordAutoClaimDuration records the duration (seconds) of a confirmed monitored tx (from creation time to confirm time)
func RecordAutoClaimDuration(dur time.Duration) {
	histogramObserve(metricMonitoredTxsDuration, float64(dur)/float64(time.Second), map[string]string{})
}

// RecordSynchronizerEvent records an event log consumed by the synchronizer
func RecordSynchronizerEvent(networkID uint32, eventType string) {
	counterInc(metricSynchronizerEventCount, map[string]string{labelNetworkID: strconv.Itoa(int(networkID)), labelEventType: eventType})
}
