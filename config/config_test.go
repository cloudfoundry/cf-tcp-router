package config_test

import (
	"github.com/cloudfoundry-incubator/cf-tcp-router/config"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("when a valid config", func() {
		It("loads the config", func() {
			expectedCfg := config.Config{
				OAuth: token_fetcher.OAuthConfig{
					TokenEndpoint: "http://uaa.service.cf.internal",
					ClientName:    "someclient",
					ClientSecret:  "somesecret",
					Port:          8080,
				},
				RoutingApi: config.RoutingApiConfig{
					Uri:  "http://routing-api.service.cf.internal",
					Port: 3000,
				},
			}
			cfg, err := config.New("fixtures/valid_config.yml")
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(expectedCfg))
		})
	})
	Context("when given an invalid config", func() {
		Context("non existing config", func() {
			It("return error", func() {
				_, err := config.New("fixtures/non_existing_config.yml")
				Expect(err).To(HaveOccurred())
			})
		})
		Context("malformed YAML config", func() {
			It("return error", func() {
				_, err := config.New("fixtures/malformed_config.yml")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
