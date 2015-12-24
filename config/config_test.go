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
				RoutingAPI: config.RoutingAPIConfig{
					URI:          "http://routing-api.service.cf.internal",
					Port:         3000,
					AuthDisabled: false,
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

	Context("when oauth section is  missing", func() {
		It("loads only routing api section", func() {
			expectedCfg := config.Config{
				RoutingAPI: config.RoutingAPIConfig{
					URI:  "http://routing-api.service.cf.internal",
					Port: 3000,
				},
			}
			cfg, err := config.New("fixtures/no_oauth.yml")
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(expectedCfg))
		})
	})

	Context("when oauth section has some missing fields", func() {
		It("loads config and defaults missing fields", func() {
			expectedCfg := config.Config{
				OAuth: token_fetcher.OAuthConfig{
					TokenEndpoint: "http://uaa.service.cf.internal",
					ClientName:    "",
					ClientSecret:  "",
					Port:          8080,
				},
				RoutingAPI: config.RoutingAPIConfig{
					URI:  "http://routing-api.service.cf.internal",
					Port: 3000,
				},
			}
			cfg, err := config.New("fixtures/missing_oauth_fields.yml")
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(expectedCfg))
		})
	})
})
