package configurer_test

import (
	"github.com/GESoftware-CF/cf-tcp-router/configurer"
	"github.com/GESoftware-CF/cf-tcp-router/configurer/haproxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configurer", func() {

	Describe("NewConfigurer", func() {
		Context("when 'haproxy' tcp load balancer is passed", func() {
			It("should return haproxy configurer", func() {
				routeConfigurer := configurer.NewConfigurer(logger, configurer.HaProxyConfigurer)
				Expect(routeConfigurer).ShouldNot(BeNil())
				Expect(routeConfigurer).To(BeAssignableToTypeOf(haproxy.HaProxyConfigurer{}))
			})
		})

		Context("when non-supported tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "not-supported")
				}).Should(Panic())
			})
		})

		Context("when empty tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "")
				}).Should(Panic())
			})
		})

	})
})
