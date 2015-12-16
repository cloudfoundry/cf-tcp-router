package metrics_reporter_test

import (
	"os"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter"
	emitter_fakes "github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/fakes"
	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client"
	haproxy_fakes "github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics Reporter", func() {
	var (
		fakeClient      *haproxy_fakes.FakeHaproxyClient
		fakeEmitter     *emitter_fakes.FakeMetricsEmitter
		metricsReporter *metrics_reporter.MetricsReporter
	)

	BeforeEach(func() {
		fakeClient = &haproxy_fakes.FakeHaproxyClient{}
		fakeEmitter = &emitter_fakes.FakeMetricsEmitter{}
		metricsReporter = metrics_reporter.NewMetricsReporter(fakeClient, fakeEmitter, 10*time.Millisecond)
	})

	Context("Run", func() {

		var (
			readyChan chan struct{}
			signals   chan os.Signal
		)

		BeforeEach(func() {
			readyChan = make(chan struct{})
			signals = make(chan os.Signal)
			go func() {
				metricsReporter.Run(signals, readyChan)
			}()
			select {
			case <-readyChan:
			}

			fakeClient.GetStatsStub = func() haproxy_client.HaproxyStats {
				return haproxy_client.HaproxyStats{
					{
						ProxyName:            "fake_pxname1_9000",
						CurrentQueued:        10,
						ErrorConnecting:      20,
						AverageQueueTimeMs:   30,
						AverageConnectTimeMs: 25,
						CurrentSessions:      15,
						AverageSessionTimeMs: 9,
					},
				}
			}
		})

		AfterEach(func() {
			signals <- syscall.SIGTERM
		})

		It("emits metrics", func() {
			Eventually(fakeClient.GetStatsCallCount, "30ms", "10ms").Should(BeNumerically(">=", 2))
			Eventually(fakeEmitter.EmitCallCount, "30ms", "10ms").Should(BeNumerically(">=", 2))
		})
	})

})
