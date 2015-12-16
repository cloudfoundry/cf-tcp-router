package haproxy_client

import "fmt"

//go:generate counterfeiter -o fakes/fake_haproxy_client.go . HaproxyClient
type HaproxyClient interface {
	GetStats() HaproxyStats
}

type HaproxyStatsClient struct{}

type HaproxyStats []HaproxyStat

type HaproxyStat struct {
	ProxyName            string `csv:"pxname"`
	CurrentQueued        uint64 `csv:"qcur"`
	CurrentSessions      uint64 `csv:"scur"`
	ErrorConnecting      uint64 `csv:"econ"`
	AverageQueueTimeMs   uint64 `csv:"qtime"`
	AverageConnectTimeMs uint64 `csv:"ctime"`
	AverageSessionTimeMs uint64 `csv:"ttime"`
}

func NewClient() *HaproxyStatsClient {
	return &HaproxyStatsClient{}
}

func (r *HaproxyStatsClient) GetStats() HaproxyStats {
	fmt.Printf("In stats function")
	// make api call
	// parse te csv
	// create haproxy stats[]
	return HaproxyStats{}
}
