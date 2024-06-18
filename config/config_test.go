package config_test

import (
	"fmt"
	"os"

	tlshelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	"code.cloudfoundry.org/cf-tcp-router/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	caFile := "fixtures/ca.pem"
	certAndKeyFile := "fixtures/cert_and_key.pem"
	mismatchedCertAndKeyFile := "fixtures/mismatched_cert_and_key.pem"

	BeforeEach(func() {
		// Generate a CA and move it into the correct location for the fixture
		tmpCAFile, _ := tlshelpers.GenerateCa()
		err := os.Rename(tmpCAFile, caFile)
		Expect(err).ToNot(HaveOccurred())

		// Generate a second trusted CA and add it to the fixture's CA file
		tmpCAFile2, _ := tlshelpers.GenerateCa()
		caBytes, err := os.ReadFile(tmpCAFile2)
		Expect(err).ToNot(HaveOccurred())
		f, err := os.OpenFile(caFile, os.O_RDWR|os.O_APPEND, 0644)
		Expect(err).ToNot(HaveOccurred())
		_, err = f.Write(caBytes)
		Expect(err).ToNot(HaveOccurred())
		err = os.Remove(tmpCAFile2)
		Expect(err).ToNot(HaveOccurred())

		// Generate a client cert + key, and move it into the correct location for the fixture
		_, tmpCertFile1, tmpKeyFile1, _ := tlshelpers.GenerateCaAndMutualTlsCerts()
		cert1Bytes, err := os.ReadFile(tmpCertFile1)
		Expect(err).NotTo(HaveOccurred())
		key1Bytes, err := os.ReadFile(tmpKeyFile1)
		Expect(err).NotTo(HaveOccurred())
		os.WriteFile(certAndKeyFile, []byte(fmt.Sprintf("%s%s", string(cert1Bytes), string(key1Bytes))), 0644)
		Expect(err).NotTo(HaveOccurred())

		// Generate a second client cert + key, and move it into the correct location for the fixture
		// used for the invalid key-pair combo to fail if a key and cert do not go together
		_, _, tmpKeyFile2, _ := tlshelpers.GenerateCaAndMutualTlsCerts()
		key2Bytes, err := os.ReadFile(tmpKeyFile2)
		Expect(err).NotTo(HaveOccurred())
		os.WriteFile(mismatchedCertAndKeyFile, []byte(fmt.Sprintf("%s%s", string(cert1Bytes), string(key2Bytes))), 0644)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		os.Remove(caFile)
		os.Remove(certAndKeyFile)
		os.Remove(mismatchedCertAndKeyFile)
	})

	Context("when a valid config", func() {
		It("loads the config", func() {
			expectedCfg := config.Config{
				OAuth: config.OAuthConfig{
					TokenEndpoint:     "uaa.service.cf.internal",
					ClientName:        "someclient",
					ClientSecret:      "somesecret",
					Port:              8443,
					SkipSSLValidation: true,
					CACerts:           "some-ca-cert",
				},
				RoutingAPI: config.RoutingAPIConfig{
					URI:          "http://routing-api.service.cf.internal",
					Port:         3000,
					AuthDisabled: false,

					ClientCertificatePath: "/a/client_cert",
					ClientPrivateKeyPath:  "/b/private_key",
					CACertificatePath:     "/c/ca_cert",
				},
				HaProxyPidFile:               "/path/to/pid/file",
				IsolationSegments:            []string{"foo-iso-seg"},
				ReservedSystemComponentPorts: []int{8080, 8081},
				BackendTLS: config.BackendTLSConfig{
					CACertificatePath:    caFile,
					ClientCertAndKeyPath: certAndKeyFile,
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

	Context("when the CA path is not a valid CA", func() {
		It("returns an error", func() {
			_, err := config.New("fixtures/bad_ca_config.yml")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the Client Cert/key pair are not valid", func() {
		It("returns an error", func() {
			_, err := config.New("fixtures/bad_client_cert_config.yml")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the Client Cert/key pair are mismatched", func() {
		It("returns an error", func() {
			_, err := config.New("fixtures/mismatched_client_cert_config.yml")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when haproxy pid file is missing", func() {
		It("return error", func() {
			_, err := config.New("fixtures/no_haproxy.yml")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when oauth section is  missing", func() {
		It("loads only routing api section", func() {
			expectedCfg := config.Config{
				RoutingAPI: config.RoutingAPIConfig{
					URI:  "http://routing-api.service.cf.internal",
					Port: 3000,
				},
				HaProxyPidFile: "/path/to/pid/file",
			}
			cfg, err := config.New("fixtures/no_oauth.yml")
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(expectedCfg))
		})
	})

	Context("when oauth section has some missing fields", func() {
		It("loads config and defaults missing fields", func() {
			expectedCfg := config.Config{
				OAuth: config.OAuthConfig{
					TokenEndpoint:     "uaa.service.cf.internal",
					ClientName:        "",
					ClientSecret:      "",
					Port:              8443,
					SkipSSLValidation: true,
				},
				RoutingAPI: config.RoutingAPIConfig{
					URI:  "http://routing-api.service.cf.internal",
					Port: 3000,
				},
				HaProxyPidFile: "/path/to/pid/file",
			}
			cfg, err := config.New("fixtures/missing_oauth_fields.yml")
			Expect(err).NotTo(HaveOccurred())
			Expect(*cfg).To(Equal(expectedCfg))
		})
	})
})
