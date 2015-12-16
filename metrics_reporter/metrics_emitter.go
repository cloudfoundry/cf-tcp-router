package metrics_reporter

type MetricsEmitter interface {
	Emit(*MetricsReport)
}
