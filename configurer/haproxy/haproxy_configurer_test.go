package haproxy_test

import (
	"github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/configurer/haproxy"
	"github.com/GESoftware-CF/cf-tcp-router/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaproxyConfigurer", func() {
	Describe("MapBackendHostsToAvailablePort", func() {
		var (
			externalIP        string
			haproxyConfigurer haproxy.HaProxyConfigurer
		)

		BeforeEach(func() {
			haproxyConfigurer = haproxy.NewHaProxyConfigurer(logger)
			externalIP = testutil.GetExternalIP()
		})

		Context("when invalid backend host info is passed", func() {
			It("should return error", func() {
				_, err := haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal(cf_tcp_router.ErrInvalidBackendHostInfo))
			})
		})

		Context("when valid backend host info is passed", func() {
			It("should return error", func() {
				routerHostInfo, err := haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
				})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(routerHostInfo.Address).Should(Equal(externalIP))
				Expect(routerHostInfo.Port).Should(BeNumerically("<", 65536))
				Expect(routerHostInfo.Port).Should(BeNumerically(">=", 0))
			})
		})
	})

})
