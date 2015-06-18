package configurer_test

import (
	"reflect"

	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer/haproxy"

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
					configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "haproxy/fixtures/haproxy.cfg")
				Expect(routeConfigurer).ShouldNot(BeNil())
				expectedType := reflect.PtrTo(reflect.TypeOf(haproxy.HaProxyConfigurer{}))
				value := reflect.ValueOf(routeConfigurer)
				Expect(value.Type()).To(Equal(expectedType))
			})

			Context("when invalid config file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "haproxy/fixtures/haproxy.cfg.template", "")
					}).Should(Panic())
				})
			})

			Context("when invalid base config file is passed", func() {
				It("should panic", func() {
					Expect(func() {
						configurer.NewConfigurer(logger, configurer.HaProxyConfigurer, "", "haproxy/fixtures/haproxy.cfg")
					}).Should(Panic())
				})
			})
		})

		Context("when non-supported tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "not-supported", "some-base-config-file", "some-config-file")
				}).Should(Panic())
			})
		})

		Context("when empty tcp load balancer is passed", func() {
			It("should panic", func() {
				Expect(func() {
					configurer.NewConfigurer(logger, "", "some-base-config-file", "some-config-file")
				}).Should(Panic())
			})
		})

	})
})
