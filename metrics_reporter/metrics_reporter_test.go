package metrics_reporter_test

import (
	"os"
	"syscall"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/metrics_reporter"
	emitter_fakes "code.cloudfoundry.org/cf-tcp-router/metrics_reporter/fakes"
	"code.cloudfoundry.org/cf-tcp-router/metrics_reporter/haproxy_client"
	haproxy_fakes "code.cloudfoundry.org/cf-tcp-router/metrics_reporter/haproxy_client/fakes"
	"code.cloudfoundry.org/clock/fakeclock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Metrics Reporter", func() {
	var (
		fakeClient      *haproxy_fakes.FakeHaproxyClient
		fakeEmitter     *emitter_fakes.FakeMetricsEmitter
		metricsReporter *metrics_reporter.MetricsReporter
		clock           *fakeclock.FakeClock
		process         ifrit.Process
		syncInterval    time.Duration
	)

	BeforeEach(func() {
		fakeClient = &haproxy_fakes.FakeHaproxyClient{}
		fakeEmitter = &emitter_fakes.FakeMetricsEmitter{}
		clock = fakeclock.NewFakeClock(time.Now())
		syncInterval = 1 * time.Second
		metricsReporter = metrics_reporter.NewMetricsReporter(clock, fakeClient, fakeEmitter, syncInterval, logger)
	})

	Context("on specified interval", func() {
		Context("when haproxy client returns stats data", func() {
			BeforeEach(func() {
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
				process.Signal(os.Interrupt)
				Eventually(process.Wait()).Should(Receive(BeNil()))
			})

			It("emits metrics", func() {

				process = ifrit.Invoke(metricsReporter)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(0))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(0))

				clock.Increment(syncInterval + 100*time.Millisecond)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(1))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(1))

				clock.Increment(syncInterval + 100*time.Millisecond)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(2))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(2))
			})
		})
		Context("when haproxy client returns no stats data", func() {
			BeforeEach(func() {
				fakeClient.GetStatsReturns(haproxy_client.HaproxyStats{})
			})

			AfterEach(func() {
				process.Signal(os.Interrupt)
				Eventually(process.Wait()).Should(Receive(BeNil()))
			})

			It("emits metrics", func() {

				process = ifrit.Invoke(metricsReporter)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(0))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(0))

				clock.Increment(syncInterval + 100*time.Millisecond)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(1))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(0))

				clock.Increment(syncInterval + 100*time.Millisecond)

				Eventually(fakeClient.GetStatsCallCount).Should(Equal(2))
				Eventually(fakeEmitter.EmitCallCount).Should(Equal(0))
			})
		})
	})

	Context("when signaled with SIGUSR2", func() {
		BeforeEach(func() {
			process = ifrit.Invoke(metricsReporter)
		})
		It("does not shut down", func() {
			process.Signal(syscall.SIGUSR2)

			Consistently(process.Wait()).ShouldNot(Receive())
		})
	})
})
