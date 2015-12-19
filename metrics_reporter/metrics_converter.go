package metrics_reporter

import (
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
)

func Convert(proxyStats haproxy_client.HaproxyStats) *MetricsReport {

	if len(proxyStats) == 0 {
		return nil
	}

	var (
		totalCurrentQueuedRequests   uint64
		totalBackendConnectionErrors uint64
		averageQueueTimeMs           uint64
		averageConnectTimeMs         uint64
		totalQueueTimeMs             uint64
		totalConnectTimeMs           uint64
		proxyStatsMap                map[models.RoutingKey]ProxyStats
	)

	proxyStatsMap = map[models.RoutingKey]ProxyStats{}

	length := uint64(len(proxyStats))

	for _, proxyStat := range proxyStats {
		totalCurrentQueuedRequests += proxyStat.CurrentQueued
		totalBackendConnectionErrors += proxyStat.ErrorConnecting
		totalConnectTimeMs += proxyStat.AverageConnectTimeMs
		totalQueueTimeMs += proxyStat.AverageQueueTimeMs

		populateProxyStats(proxyStat, proxyStatsMap)
	}
	averageQueueTimeMs = totalQueueTimeMs / length
	averageConnectTimeMs = totalConnectTimeMs / length

	return &MetricsReport{
		TotalCurrentQueuedRequests:   totalCurrentQueuedRequests,
		TotalBackendConnectionErrors: totalBackendConnectionErrors,
		AverageQueueTimeMs:           averageQueueTimeMs,
		AverageConnectTimeMs:         averageConnectTimeMs,
		ProxyMetrics:                 proxyStatsMap,
	}
}

func populateProxyStats(proxyStat haproxy_client.HaproxyStat, proxyStatsMap map[models.RoutingKey]ProxyStats) {
	key, err := proxyKey(proxyStat.ProxyName)
	if err == nil {
		if _, ok := proxyStatsMap[key]; !ok {
			proxyStatsMap[key] = ProxyStats{}
		}
		v, _ := proxyStatsMap[key]
		v.ConnectionTime += proxyStat.AverageConnectTimeMs
		v.CurrentSessions += proxyStat.CurrentSessions
		proxyStatsMap[key] = v
	}
}

// proxyname i.e.  listen_cfg_9001, listen_cfg_9002
func proxyKey(proxy string) (models.RoutingKey, error) {
	proxyNameParts := strings.Split(proxy, "_")
	port, err := strconv.ParseUint(proxyNameParts[2], 10, 16)
	if err != nil {
		return models.RoutingKey{}, err
	}
	return models.RoutingKey{Port: uint16(port)}, nil
}
