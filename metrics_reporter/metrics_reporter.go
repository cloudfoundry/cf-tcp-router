package metrics_reporter

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
)

type MetricsReport struct {
	TotalCurrentQueuedRequests   uint64
	TotalBackendConnectionErrors uint64
	AverageQueueTimeMs           uint64
	AverageConnectTimeMs         uint64
	ProxyMetrics                 map[models.RoutingKey]ProxyStats
}

type ProxyStats struct {
	ConnectionTime  uint64
	CurrentSessions uint64
}

type MetricsReporter struct {
	emitInterval   time.Duration
	haproxyClient  haproxy_client.HaproxyClient
	metricsEmitter MetricsEmitter
}

func NewMetricsReporter(haproxyClient haproxy_client.HaproxyClient, metricsEmitter MetricsEmitter, interval time.Duration) *MetricsReporter {
	return &MetricsReporter{
		haproxyClient:  haproxyClient,
		metricsEmitter: metricsEmitter,
		emitInterval:   interval,
	}
}

func (r *MetricsReporter) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	ticker := time.NewTicker(r.emitInterval)
	close(ready)
	for {
		select {
		case <-ticker.C:
			r.emitStats()
		case <-signals:
			ticker.Stop()
			return nil
		}
	}
	return nil
}

func (r *MetricsReporter) emitStats() {
	// get stats
	stats := r.haproxyClient.GetStats()

	// convert to report
	report := Convert(stats)

	// emit to firehose
	r.metricsEmitter.Emit(report)
}
