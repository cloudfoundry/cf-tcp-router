package haproxy_test

import (
	"github.com/GESoftware-CF/cf-tcp-router/configurer/haproxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaproxyConfiguration", func() {
	Describe("BackendServerInfo", func() {
		Context("when configuration is valid", func() {
			It("returns a valid haproxy configuration representation", func() {
				bs := haproxy.NewBackendServerInfo("some-name", "some-ip", 1234)
				str, err := bs.ToHaProxyConfig()
				Expect(err).ShouldNot(HaveOccurred())
				Expect(str).Should(Equal("server some-name some-ip:1234\n"))
			})
		})

		Context("when configuration is invalid", func() {
			Context("when name is empty", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("", "some-ip", 1234)
					_, err := bs.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("backend_server.name"))
				})
			})

			Context("when address is empty", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("some-name", "", 1234)
					_, err := bs.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("backend_server.address"))
				})
			})

			Context("when port is invalid", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("some-name", "some-ip", 0)
					_, err := bs.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("backend_server.port"))
				})
			})
		})
	})

	Describe("ListenConfigurationInfo", func() {
		Context("when configuration is valid", func() {
			Context("when single backend server info is provided", func() {
				It("returns a valid haproxy configuration representation", func() {
					bs := haproxy.NewBackendServerInfo("some-name", "some-ip", 1234)
					lc := haproxy.NewListenConfigurationInfo("listen-1", 8880, []haproxy.BackendServerInfo{bs})
					str, err := lc.ToHaProxyConfig()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(str).Should(Equal("listen listen-1\n  mode tcp\n  bind :8880\n  server some-name some-ip:1234\n"))
				})
			})

			Context("when multiple backend server infos are provided", func() {
				It("returns a valid haproxy configuration representation", func() {
					bs1 := haproxy.NewBackendServerInfo("some-name-1", "some-ip-1", 1234)
					bs2 := haproxy.NewBackendServerInfo("some-name-2", "some-ip-2", 1235)
					lc := haproxy.NewListenConfigurationInfo("listen-1", 8880, []haproxy.BackendServerInfo{bs1, bs2})
					str, err := lc.ToHaProxyConfig()
					Expect(err).ShouldNot(HaveOccurred())
					Expect(str).Should(Equal("listen listen-1\n  mode tcp\n  bind :8880\n  server some-name-1 some-ip-1:1234\n  server some-name-2 some-ip-2:1235\n"))
				})
			})
		})

		Context("when configuration is invalid", func() {
			Context("when name is empty", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("some-name", "some-ip", 1234)
					lc := haproxy.NewListenConfigurationInfo("", 8880, []haproxy.BackendServerInfo{bs})
					_, err := lc.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("listen_configuration.name"))
				})
			})

			Context("when front end port is invalid", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("some-name", "some-ip", 1234)
					lc := haproxy.NewListenConfigurationInfo("some-name", 0, []haproxy.BackendServerInfo{bs})
					_, err := lc.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("listen_configuration.port"))
				})
			})

			Context("when backend server is invalid", func() {
				It("returns an error", func() {
					bs := haproxy.NewBackendServerInfo("", "some-ip", 1234)
					lc := haproxy.NewListenConfigurationInfo("some-name", 8880, []haproxy.BackendServerInfo{bs})
					_, err := lc.ToHaProxyConfig()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("backend_server.name"))
				})
			})
		})
	})
})
