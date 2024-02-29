package metrics

import (
	"strconv"
	"time"

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
