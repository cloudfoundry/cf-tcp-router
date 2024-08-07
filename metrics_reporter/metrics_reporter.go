package metrics_reporter

import (
	"os"
	"syscall"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/metrics_reporter/haproxy_client"
	"code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"
)

type MetricsReport struct {
	TotalCurrentQueuedRequests   uint64
	TotalBackendConnectionErrors uint64
	AverageQueueTimeMs           uint64
	AverageConnectTimeMs         uint64
	ProxyMetrics                 map[models.RoutingKey]ProxyStats
	RouteErrorMap                map[string]uint64
}

type RouteErrorReport struct {
	RouteErrors map[models.RoutingKey]uint64
}

type ProxyStats struct {
	ConnectionTime  uint64
	CurrentSessions uint64
}

type MetricsReporter struct {
	clock          clock.Clock
	emitInterval   time.Duration
	haproxyClient  haproxy_client.HaproxyClient
	metricsEmitter MetricsEmitter
	logger         lager.Logger
}

func NewMetricsReporter(clock clock.Clock, haproxyClient haproxy_client.HaproxyClient, metricsEmitter MetricsEmitter, interval time.Duration, logger lager.Logger) *MetricsReporter {
	return &MetricsReporter{
		clock:          clock,
		haproxyClient:  haproxyClient,
		metricsEmitter: metricsEmitter,
		emitInterval:   interval,
		logger:         logger.Session("metrics-reporter"),
	}
}

func (r *MetricsReporter) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	ticker := r.clock.NewTicker(r.emitInterval)
	close(ready)
	for {
		select {
		case <-ticker.C():
			r.emitStats()
		case sig := <-signals:
			if sig != syscall.SIGUSR2 {
				r.logger.Info("stopping")
				ticker.Stop()
				return nil
			}
		}
	}
}

func (r *MetricsReporter) emitStats() {
	// get stats
	stats := r.haproxyClient.GetStats()

	if len(stats) > 0 {
		// convert to report
		report := Convert(stats)

		// emit to firehose
		r.metricsEmitter.Emit(report)
	}
}
