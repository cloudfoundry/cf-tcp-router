package haproxy_client_test

import (
	"net/http"
    "io/ioutil"
	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("HaproxyClient", func() {

	var (
		haproxyClient haproxy_client.HaproxyClient
		server        *ghttp.Server
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		haproxyClient = haproxy_client.NewClient(server.URL())
	})
	AfterEach(func() {
		server.Close()
	})

	Describe("GetStats", func() {

		BeforeEach(func() {

			csvPayload, err := ioutil.ReadFile("testdata.csv")
			Expect(err).NotTo(HaveOccurred())

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", haproxy_client.STATS_PATH),
					ghttp.RespondWith(http.StatusOK, csvPayload),
				),
			)
		})
		Context("when haproxy provides statistics", func() {

			It("returns haproxy statistics", func() {
				stats := haproxyClient.GetStats()
				Expect(stats).Should(HaveLen(9))

				r0 := haproxy_client.HaproxyStat{
					ProxyName:            "stats",
					CurrentQueued:        100,
					CurrentSessions:      101,
					ErrorConnecting:      102,
					AverageQueueTimeMs:   103,
					AverageConnectTimeMs: 104,
					AverageSessionTimeMs: 105,
				}

				r8 := haproxy_client.HaproxyStat{
					ProxyName:            "listen_cfg_60001",
					CurrentQueued:        1000,
					CurrentSessions:      1001,
					ErrorConnecting:      1002,
					AverageQueueTimeMs:   1003,
					AverageConnectTimeMs: 1004,
					AverageSessionTimeMs: 1005,
				}

				Expect(stats[0]).Should(Equal(r0))
				Expect(stats[8]).Should(Equal(r8))
			})

		})
	})

})
