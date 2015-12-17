package haproxy_client

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/cf_http"
)

//go:generate counterfeiter -o fakes/fake_haproxy_client.go . HaproxyClient
type HaproxyClient interface {
	GetStats() HaproxyStats
}

type HaproxyStatsClient struct {
	haproxyStatsUrl string
	httpClient      *http.Client
}

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

const STATS_PATH = "/haproxy/stats;csv"

func NewClient(haproxyStatsUrl string) *HaproxyStatsClient {
	return &HaproxyStatsClient{
		haproxyStatsUrl: haproxyStatsUrl,
		httpClient:      cf_http.NewClient(),
	}
}

func (r *HaproxyStatsClient) GetStats() HaproxyStats {
	fmt.Printf("In stats function")

	resp, err := r.httpClient.Get(r.haproxyStatsUrl + STATS_PATH)
	if err != nil {
		return HaproxyStats{}
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New("http-error-fetching-key")
		return HaproxyStats{}
	}

	csvReader := csv.NewReader(resp.Body)
	lines, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("error reading all lines: %v", err)
	}

	stats := HaproxyStats{}

	for i, line := range lines {
		if i == 0 {
			// skip header line
			continue
		}
		stats = append(stats, fromCsv(line))
	}
	return stats
}

func fromCsv(row []string) HaproxyStat {
	return HaproxyStat{
		ProxyName:            row[0],
		CurrentQueued:        convertToInt(row[2]),
		CurrentSessions:      convertToInt(row[4]),
		ErrorConnecting:      convertToInt(row[13]),
		AverageQueueTimeMs:   convertToInt(row[58]),
		AverageConnectTimeMs: convertToInt(row[59]),
		AverageSessionTimeMs: convertToInt(row[61]),
	}
}

func convertToInt(s string) uint64 {
	i, _ := strconv.ParseUint(s, 10, 64)
	return i
}
