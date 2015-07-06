package haproxy_test

import (
	"fmt"
	"io/ioutil"
	"os"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer/haproxy"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
	"github.com/cloudfoundry-incubator/cf-tcp-router/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaproxyConfigurer", func() {
	Describe("Configure", func() {
		const (
			haproxyConfigTemplate = "fixtures/haproxy.cfg.template"
			haproxyConfigFile     = "fixtures/haproxy.cfg"
		)

		var (
			haproxyConfigurer *haproxy.HaProxyConfigurer
		)

		verifyHaProxyConfigContent := func(haproxyFileName, expectedContent string, present bool) {
			data, err := ioutil.ReadFile(haproxyFileName)
			Expect(err).ShouldNot(HaveOccurred())
			if present {
				Expect(string(data)).Should(ContainSubstring(expectedContent))
			} else {
				Expect(string(data)).ShouldNot(ContainSubstring(expectedContent))
			}
		}

		Context("when empty base configuration file is passed", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, "", haproxyConfigFile)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when empty configuration file is passed", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxyConfigTemplate, "")
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when base configuration file does not exist", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, "file/path/does/not/exists", haproxyConfigFile)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when configuration file does not exist", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, haproxyConfigTemplate, "file/path/does/not/exists")
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when invalid routing table is passed", func() {
			var (
				expectedContent []byte
			)
			BeforeEach(func() {
				var err error
				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, haproxyConfigTemplate, haproxyConfigFile)
				Expect(err).ShouldNot(HaveOccurred())
				expectedContent, err = ioutil.ReadFile(haproxyConfigFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("returns error and doesn't update config file", func() {
				routingKey := models.RoutingKey{Port: 0}
				routingTableEntry := models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						models.BackendServerInfo{"some-ip", 1234}: struct{}{},
					},
				}
				routingTable := models.NewRoutingTable()
				ok := routingTable.Set(routingKey, routingTableEntry)
				Expect(ok).To(BeTrue())
				err := haproxyConfigurer.Configure(routingTable)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("listen_configuration.port"))
				content, err := ioutil.ReadFile(haproxyConfigFile)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(content).Should(Equal(expectedContent))
			})
		})

		Context("when valid routing table and config file is passed", func() {
			var (
				generatedHaproxyCfgFile      string
				haproxyCfgBackupFile         string
				err                          error
				haproxyConfigTemplateContent []byte
			)

			BeforeEach(func() {

				generatedHaproxyCfgFile = testutil.RandomFileName("fixtures/haproxy_", ".cfg")
				haproxyCfgBackupFile = fmt.Sprintf("%s.bak", generatedHaproxyCfgFile)
				utils.CopyFile(haproxyConfigTemplate, generatedHaproxyCfgFile)

				haproxyConfigTemplateContent, err = ioutil.ReadFile(generatedHaproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, haproxyConfigTemplate, generatedHaproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(generatedHaproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(utils.FileExists(haproxyCfgBackupFile)).To(BeTrue())
				err = os.Remove(haproxyCfgBackupFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			Context("when CreateExternalPortMappings is called once", func() {
				Context("when only one mapping is provided as part of request", func() {

					BeforeEach(func() {
						routingTable := models.NewRoutingTable()
						routingTableEntry := models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1235},
							},
						)
						routinTableKey := models.RoutingKey{Port: 2222}
						ok := routingTable.Set(routinTableKey, routingTableEntry)
						Expect(ok).To(BeTrue())
						err = haproxyConfigurer.Configure(routingTable)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("appends the haproxy config with new listen configuration", func() {
						listenCfg :=
							"\nlisten listen_cfg_2222\n  mode tcp\n  bind :2222\n"
						serverConfig1 := "server server_some-ip-1_1234 some-ip-1:1234"
						serverConfig2 := "server server_some-ip-2_1235 some-ip-2:1235"
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, listenCfg, true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, serverConfig1, true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, serverConfig2, true)
					})
				})

				Context("when multiple mappings are provided as part of one request", func() {

					BeforeEach(func() {
						routingTable := models.NewRoutingTable()
						routingTableEntry := models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1235},
							},
						)
						routinTableKey := models.RoutingKey{Port: 2222}
						ok := routingTable.Set(routinTableKey, routingTableEntry)
						Expect(ok).To(BeTrue())
						routingTableEntry = models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-3", 1234},
								models.BackendServerInfo{"some-ip-4", 1235},
							},
						)
						routinTableKey = models.RoutingKey{Port: 3333}
						ok = routingTable.Set(routinTableKey, routingTableEntry)
						Expect(ok).To(BeTrue())

						err = haproxyConfigurer.Configure(routingTable)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("appends the haproxy config with new listen configuration", func() {
						listenCfg1 := `
listen listen_cfg_2222
  mode tcp
  bind :2222
`
						listenCfg2 := `
listen listen_cfg_3333
  mode tcp
  bind :3333
`
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, listenCfg1, true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-1_1234 some-ip-1:1234", true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-2_1235 some-ip-2:1235", true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, listenCfg2, true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-3_1234 some-ip-3:1234", true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-4_1235 some-ip-4:1235", true)
						verifyHaProxyConfigContent(generatedHaproxyCfgFile, string(haproxyConfigTemplateContent), true)
					})
				})
			})

			Context("when Configure is called multiple times", func() {
				BeforeEach(func() {
					routingTable := models.NewRoutingTable()
					routingTableEntry := models.NewRoutingTableEntry(
						models.BackendServerInfos{
							models.BackendServerInfo{"some-ip-1", 1234},
							models.BackendServerInfo{"some-ip-2", 1235},
						},
					)
					routinTableKey := models.RoutingKey{Port: 2222}
					ok := routingTable.Set(routinTableKey, routingTableEntry)
					Expect(ok).To(BeTrue())
					err = haproxyConfigurer.Configure(routingTable)
					Expect(err).ShouldNot(HaveOccurred())

					routingTable = models.NewRoutingTable()
					routingTableEntry = models.NewRoutingTableEntry(
						models.BackendServerInfos{
							models.BackendServerInfo{"some-ip-3", 2345},
							models.BackendServerInfo{"some-ip-4", 3456},
						},
					)
					routinTableKey = models.RoutingKey{Port: 3333}
					ok = routingTable.Set(routinTableKey, routingTableEntry)
					Expect(ok).To(BeTrue())
					err = haproxyConfigurer.Configure(routingTable)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("persists the last routing table in haproxy config", func() {
					listenCfg := `
listen listen_cfg_3333
  mode tcp
  bind :3333
`
					notPresentCfg := `
listen listen_cfg_2222
  mode tcp
  bind :2222
`
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, listenCfg, true)
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-3_2345 some-ip-3:2345", true)
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-4_3456 some-ip-4:3456", true)

					verifyHaProxyConfigContent(generatedHaproxyCfgFile, notPresentCfg, false)
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-1_1234 some-ip-1:1234", false)
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, "server server_some-ip-2_1235 some-ip-2:1235", false)
					verifyHaProxyConfigContent(generatedHaproxyCfgFile, string(haproxyConfigTemplateContent), true)
				})
			})
		})
	})
})
