package metrics

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/localcache"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
)

func initMetrics() {
	if !initialized {
		registerer = prometheus.DefaultRegisterer
		gauges = make(map[string]*prometheus.GaugeVec)
		counters = make(map[string]*prometheus.CounterVec)
		histograms = make(map[string]*prometheus.HistogramVec)
		initialized = true
	}

	registerCounter(prometheus.CounterOpts{Name: metricRequestCount}, labelMethod, labelIsSuccess)
	registerHistogram(prometheus.HistogramOpts{Name: metricRequestLatency}, labelMethod, labelIsSuccess)
	registerCounter(prometheus.CounterOpts{Name: metricOrderCount}, labelNetworkID, labelToken)
	registerCounter(prometheus.CounterOpts{Name: metricOrderTotalAmount}, labelNetworkID, labelToken)
	registerHistogram(prometheus.HistogramOpts{Name: metricOrderWaitTime}, labelNetworkID)
}

// RecordRequest increments the request count for the method
func RecordRequest(method string, isSuccess bool) {
	counterInc(metricRequestCount, map[string]string{labelMethod: method, labelIsSuccess: strconv.FormatBool(isSuccess)})
}

// RecordRequestLatency records the latency histogram in nanoseconds
func RecordRequestLatency(method string, latency time.Duration, isSuccess bool) {
	latencyNs := latency / time.Nanosecond
	histogramObserve(metricRequestLatency, float64(latencyNs), map[string]string{labelMethod: method, labelIsSuccess: strconv.FormatBool(isSuccess)})
}

// RecordOrder records one bridge order, increase the order count and add the amount to the total order amount
// networkID is the "from" network of the transaction
func RecordOrder(networkID, tokenOriginNetwork uint32, tokenAddress common.Address, amount *big.Int) {
	tokenSymbol := ""
	if coinsCache := localcache.GetDefaultCache(); coinsCache != nil {
		coinInfo, err := coinsCache.GetCoinInfoByAddress(context.Background(), tokenOriginNetwork, tokenAddress)
		if err == nil {
			tokenSymbol = coinInfo.Symbol
		}
	}

	// This is inflated amount, e.g.: 1 ETH is stored as 1000000000000000000
	floatAmount, err := strconv.ParseFloat(amount.String(), 10) //nolint:gomnd
	if err != nil {
		log.Warnf("cannot convert [%v] to float", amount.String())
	}

	labels := map[string]string{labelNetworkID: strconv.Itoa(int(networkID)), labelToken: tokenSymbol}

	counterInc(metricOrderCount, labels)
	counterAdd(metricOrderTotalAmount, floatAmount, labels)
}

func RecordOrderWaitTime(networkID uint32, dur time.Duration) {
	histogramObserve(metricOrderWaitTime, float64(dur/time.Second), map[string]string{labelNetworkID: strconv.Itoa(int(networkID))})
}
