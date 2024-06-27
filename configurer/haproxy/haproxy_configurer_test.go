package haproxy_test

import (
	"fmt"
	"os"

	tlshelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	"code.cloudfoundry.org/cf-tcp-router/config"
	"code.cloudfoundry.org/cf-tcp-router/configurer/haproxy"
	"code.cloudfoundry.org/cf-tcp-router/configurer/haproxy/fakes"
	"code.cloudfoundry.org/cf-tcp-router/models"
	monitorFakes "code.cloudfoundry.org/cf-tcp-router/monitor/fakes"
	"code.cloudfoundry.org/cf-tcp-router/testutil"
	"code.cloudfoundry.org/cf-tcp-router/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaproxyConfigurer", func() {
	Describe("Configure", func() {
		const (
			haproxyConfigTemplate = "fixtures/haproxy.cfg.template"
			haproxyConfigFile     = "fixtures/haproxy.cfg"
		)
		var (
			haproxyConfigurer *haproxy.Configurer
			fakeMonitor       *monitorFakes.FakeMonitor
			backendTlsCfg     config.BackendTLSConfig
		)

		BeforeEach(func() {
			fakeMonitor = &monitorFakes.FakeMonitor{}
			caFile, _ := tlshelpers.GenerateCa()
			backendTlsCfg = config.BackendTLSConfig{
				CACertificatePath: caFile,
			}

		})

		Context("when empty base configuration file is passed", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), "", haproxyConfigFile, fakeMonitor, nil, backendTlsCfg)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(haproxy.ErrRouterConfigFileNotFound))
			})
		})

		Context("when empty configuration file is passed", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), haproxyConfigTemplate, "", fakeMonitor, nil, backendTlsCfg)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(haproxy.ErrRouterConfigFileNotFound))
			})
		})

		Context("when an empty CA file path is passed", func() {
			It("does not return a ErrRouterCAFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), haproxyConfigTemplate, haproxyConfigFile, fakeMonitor, nil, config.BackendTLSConfig{})
				Expect(err).ShouldNot(HaveOccurred())
			})

		})

		Context("when base configuration file does not exist", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), "file/path/does/not/exist", haproxyConfigFile, fakeMonitor, nil, backendTlsCfg)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(haproxy.ErrRouterConfigFileNotFound))
			})
		})

		Context("when the CA file path does not exist", func() {
			It("returns a ErrRouterCAFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), haproxyConfigTemplate, haproxyConfigFile, fakeMonitor, nil, config.BackendTLSConfig{CACertificatePath: "file/path/does/not/exist"})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(haproxy.ErrRouterCAFileNotFound))
			})

		})

		Context("when configuration file does not exist", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxy.NewConfigMarshaller(logger), haproxyConfigTemplate, "file/path/does/not/exist", fakeMonitor, nil, backendTlsCfg)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(haproxy.ErrRouterConfigFileNotFound))
			})
		})

		Context("when necessary files exist", func() {
			var (
				originalConfigTemplateContent []byte
				currentConfigTemplateContent  []byte
				routingTable                  models.RoutingTable
				fakeMarshaller                *fakes.FakeConfigMarshaller
				fakeScriptRunner              *fakes.FakeScriptRunner
				generatedHaproxyCfgFile       string
				haproxyCfgBackupFile          string
				err                           error
			)

			marshallerContent := "whatever the marshaller generates to represent the HAProxyConfig"

			BeforeEach(func() {
				currentConfigTemplateContent = []byte{}
				routingTable = models.NewRoutingTable(logger)

				generatedHaproxyCfgFile = testutil.RandomFileName("fixtures/haproxy_", ".cfg")
				haproxyCfgBackupFile = fmt.Sprintf("%s.bak", generatedHaproxyCfgFile)
				_ = utils.CopyFile(haproxyConfigTemplate, generatedHaproxyCfgFile)

				originalConfigTemplateContent, err = os.ReadFile(generatedHaproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				fakeMarshaller = new(fakes.FakeConfigMarshaller)
				fakeScriptRunner = new(fakes.FakeScriptRunner)
				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, fakeMarshaller, haproxyConfigTemplate, generatedHaproxyCfgFile, fakeMonitor, fakeScriptRunner, backendTlsCfg)
				Expect(err).ShouldNot(HaveOccurred())

				fakeMarshaller.MarshalCalls(func(haproxyConf models.HAProxyConfig, backendTlsCfg config.BackendTLSConfig) string {
					return fmt.Sprintf("%s\nca-file-path: %s", marshallerContent, backendTlsCfg.CACertificatePath)
				})
			})

			AfterEach(func() {
				err := os.Remove(generatedHaproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(utils.FileExists(haproxyCfgBackupFile)).To(BeTrue())
				err = os.Remove(haproxyCfgBackupFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			Context("when Configure is called once", func() {
				It("writes contents to file", func() {
					err = haproxyConfigurer.Configure(routingTable)
					Expect(err).ToNot(HaveOccurred())

					currentConfigTemplateContent, err = os.ReadFile(generatedHaproxyCfgFile)
					Expect(err).ToNot(HaveOccurred())

					expected := fmt.Sprintf("%s%s\nca-file-path: %s", string(originalConfigTemplateContent), marshallerContent, backendTlsCfg.CACertificatePath)
					Expect(string(currentConfigTemplateContent)).To(Equal(expected))

					Expect(fakeMonitor.StopWatchingCallCount()).To(Equal(1))
					Expect(fakeScriptRunner.RunCallCount()).To(Equal(1))
					Expect(fakeMonitor.StartWatchingCallCount()).To(Equal(1))
				})
			})

			Context("when Configure is called twice", func() {
				It("overwrites the file every time (does not accumulate marshalled contents)", func() {
					err = haproxyConfigurer.Configure(routingTable)
					Expect(err).ToNot(HaveOccurred())

					err = haproxyConfigurer.Configure(routingTable)
					Expect(err).ToNot(HaveOccurred())

					currentConfigTemplateContent, err = os.ReadFile(generatedHaproxyCfgFile)
					Expect(err).ToNot(HaveOccurred())

					// File contains only the most recent copy of marshallerContent
					expected := fmt.Sprintf("%s%s\nca-file-path: %s", string(originalConfigTemplateContent), marshallerContent, backendTlsCfg.CACertificatePath)
					Expect(string(currentConfigTemplateContent)).To(Equal(expected))

					// Restarts after each call, though
					Expect(fakeMonitor.StopWatchingCallCount()).To(Equal(2))
					Expect(fakeScriptRunner.RunCallCount()).To(Equal(2))
					Expect(fakeMonitor.StartWatchingCallCount()).To(Equal(2))
				})
			})
		})
	})
})
