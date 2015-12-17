package metrics_reporter

import (
  "github.com/cloudfoundry/dropsonde/metrics"
  "time"
)

// Prototype ideas about how to use dropsonde
type Counter string

func (name Counter) Send(proxyName string, value uint64) {
	metrics.SendValue(proxyName+"."+string(name), float64(value), "Metric")
}

type Duration string

func (name Duration) Send(proxyName string, duration time.Duration) {
	metrics.SendValue(proxyName+"."+string(name), float64(duration), "nanos")
}

var (
	currentQueued = Counter("CurrentQueued")
)
type MetricsEmitter interface {
	Emit(*MetricsReport)
}

type metricsEmitter struct{}

func(e metricsEmitter) Emit(r *MetricsReport) {
	// currentQueued.Send(r.P, value)
}