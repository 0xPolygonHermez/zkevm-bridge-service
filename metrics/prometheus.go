package metrics

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	mutex       sync.RWMutex
	registerer  prometheus.Registerer
	initialized bool

	gauges     map[string]*prometheus.GaugeVec
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
)

func getLogger(metricName, metricType string) *log.Logger {
	return log.WithFields("metricName", metricName, "metricType", metricType)
}

// StartMetricsHttpServer initializes the metrics registry and starts the prometheus metrics HTTP server
func StartMetricsHttpServer(c Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !c.Enabled {
		return
	}

	// Init metrics registry
	initMetrics()

	// Start metrics HTTP server
	mux := http.NewServeMux()
	addr := ":" + c.Port

	mux.Handle(endpointMetrics, promhttp.Handler())
	srv := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 5 * time.Second, //nolint:gomnd
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		// gracefully shutdown the server
		for range ch {
			_ = srv.Shutdown(ctx)
			<-ctx.Done()
		}

		_, cancel := context.WithTimeout(ctx, 5*time.Second) //nolint:gomnd
		defer cancel()

		_ = srv.Shutdown(ctx)
	}()

	err := srv.ListenAndServe()
	if err != nil {
		log.Errorf("serve metrics http server error: %v", err)
	}
}

/*
 * -------------------- Gauge functions --------------------
 */

func registerGauge(opt prometheus.GaugeOpts, labelNames ...string) {
	logger := getLogger(opt.Name, typeGauge)
	if !initialized {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := gauges[opt.Name]; ok {
		return
	}

	collector := prometheus.NewGaugeVec(opt, labelNames)
	if err := registerer.Register(collector); err != nil {
		logger.Errorf("metrics register error: %v", err)
		return
	}
	gauges[opt.Name] = collector

	logger.Debugf("metrics register successfully")
}

func gaugeSet(name string, value float64, labelValues map[string]string) {
	if !initialized {
		return
	}

	c, ok := gauges[name]
	if !ok {
		getLogger(name, typeGauge).Errorf("collector not found")
		return
	}
	c.With(labelValues).Set(value)
}

//func gaugeInc(name string, labelValues map[string]string) {
//	if !initialized {
//		return
//	}
//
//	c, ok := gauges[name]
//	if !ok {
//		getLogger(name, typeGauge).Errorf("collector not found")
//		return
//	}
//	c.With(labelValues).Inc()
//}
//
//func gaugeDec(name string, labelValues map[string]string) {
//	if !initialized {
//		return
//	}
//
//	c, ok := gauges[name]
//	if !ok {
//		getLogger(name, typeGauge).Errorf("collector not found")
//		return
//	}
//	c.With(labelValues).Dec()
//}
//
//func gaugeAdd(name string, value float64, labelValues map[string]string) {
//	if !initialized {
//		return
//	}
//
//	c, ok := gauges[name]
//	if !ok {
//		getLogger(name, typeGauge).Errorf("collector not found")
//		return
//	}
//	c.With(labelValues).Add(value)
//}
//
//func gaugeSub(name string, value float64, labelValues map[string]string) {
//	if !initialized {
//		return
//	}
//
//	c, ok := gauges[name]
//	if !ok {
//		getLogger(name, typeGauge).Errorf("collector not found")
//		return
//	}
//	c.With(labelValues).Sub(value)
//}

/*
 * -------------------- Counter functions --------------------
 */

func registerCounter(opt prometheus.CounterOpts, labelNames ...string) {
	logger := getLogger(opt.Name, typeCounter)
	if !initialized {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := gauges[opt.Name]; ok {
		return
	}

	collector := prometheus.NewCounterVec(opt, labelNames)
	if err := registerer.Register(collector); err != nil {
		logger.Errorf("metrics register error: %v", err)
		return
	}
	counters[opt.Name] = collector

	logger.Debugf("metrics register successfully")
}

func counterInc(name string, labelValues map[string]string) {
	if !initialized {
		return
	}

	c, ok := counters[name]
	if !ok {
		getLogger(name, typeCounter).Errorf("collector not found")
		return
	}
	c.With(labelValues).Inc()
}

func counterAdd(name string, value float64, labelValues map[string]string) {
	if !initialized {
		return
	}

	c, ok := counters[name]
	if !ok {
		getLogger(name, typeCounter).Errorf("collector not found")
		return
	}
	c.With(labelValues).Add(value)
}

/*
 * -------------------- Histogram functions --------------------
 */

func registerHistogram(opt prometheus.HistogramOpts, labelNames ...string) {
	logger := getLogger(opt.Name, typeHistogram)
	if !initialized {
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := gauges[opt.Name]; ok {
		return
	}

	collector := prometheus.NewHistogramVec(opt, labelNames)
	if err := registerer.Register(collector); err != nil {
		logger.Errorf("metrics register error: %v", err)
		return
	}
	histograms[opt.Name] = collector

	logger.Debugf("metrics register successfully")
}

func histogramObserve(name string, value float64, labelValues map[string]string) {
	if !initialized {
		return
	}

	c, ok := histograms[name]
	if !ok {
		getLogger(name, typeHistogram).Errorf("collector not found")
		return
	}
	c.With(labelValues).Observe(value)
}
