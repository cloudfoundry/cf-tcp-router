package cf_tcp_router_test

import (
	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate", func() {

	Describe("BackendHostInfo", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12)
				Expect(backendHostInfo.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					backendHostInfo := cf_tcp_router.NewBackendHostInfo("", 12)
					err := backendHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("backend_ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 0)
					err := backendHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("backend_port"))
				})
			})
		})
	})

	Describe("BackendHostInfos", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				backendHostInfos := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
					cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
				}
				Expect(backendHostInfos.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					backendHostInfos := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("", 12),
						cf_tcp_router.NewBackendHostInfo("", 13),
					}
					err := backendHostInfos.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("backend_ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					backendHostInfos := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("1.2.3.4", 0),
						cf_tcp_router.NewBackendHostInfo("1.2.3.5", 12),
					}
					err := backendHostInfos.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("backend_port"))
				})
			})
		})
	})

	Describe("RouterHostInfo", func() {
		Context("when values are valid", func() {

			It("does not raise an error", func() {
				routerHostInfo := cf_tcp_router.NewRouterHostInfo("1.2.3.4", 12)
				Expect(routerHostInfo.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					routerHostInfo := cf_tcp_router.NewRouterHostInfo("", 12)
					err := routerHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("router_ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					routerHostInfo := cf_tcp_router.NewRouterHostInfo("1.2.3.4", 0)
					err := routerHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("router_port"))
				})
			})
		})
	})
})
