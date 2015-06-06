package configurer_test

import (
	"reflect"

	"github.com/GESoftware-CF/cf-tcp-router/configurer"
	"github.com/GESoftware-CF/cf-tcp-router/configurer/haproxy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Configurer", func() {

	const (
		startPort = 62000
	)
	Describe("NewConfigurer", func() {
		Context("when 'haproxy' tcp load balancer is passed", func() {
			It("should return haproxy configurer", func() {
				routeConfigurer := configurer.NewConfigurer(logger,
					configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", startPort)
				Expect(routeConfigurer).ShouldNot(BeNil())
				expectedType := reflect.PtrTo(reflect.TypeOf(haproxy.HaProxyConfigurer{}))
				value := reflect.ValueOf(routeConfigurer)
				Expect(value.Type()).To(Equal(expectedType))
			})

			Context("when invalid config file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "", startPort)
					}).Should(Panic())
				})
			})
		})

		Context("when non-supported tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "not-supported", "some-config-file", startPort)
				}).Should(Panic())
			})
		})

		Context("when empty tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "", "some-config-file", startPort)
				}).Should(Panic())
			})
		})

		Context("when invalid start front end port is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", 0)
				}).Should(Panic())
			})
		})

	})
})
