package haproxy_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/configurer/haproxy"
	"github.com/GESoftware-CF/cf-tcp-router/testutil"
	"github.com/GESoftware-CF/cf-tcp-router/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaproxyConfigurer", func() {
	Describe("CreateExternalPortMappings", func() {
		const (
			haproxyCfgTemplate = "fixtures/haproxy.cfg.template"
			startPort          = 61000
		)

		var (
			externalIP        string
			haproxyConfigurer *haproxy.HaProxyConfigurer
		)

		verifyHaProxyConfigContent := func(haproxyFileName, expectedContent string) {
			data, err := ioutil.ReadFile(haproxyFileName)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(data)).Should(ContainSubstring(expectedContent))
		}

		BeforeEach(func() {
			externalIP = testutil.GetExternalIP()
		})

		Context("when invalid maping request is passed", func() {
			BeforeEach(func() {
				var err error
				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, haproxyCfgTemplate, startPort)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("returns error", func() {
				err := haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal(cf_tcp_router.ErrInvalidMapingRequest))
			})
		})

		Context("when empty configuration file is passed", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, "", startPort)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when invalid start frontend port is passed", func() {
			Context("when the frontend port is zero", func() {
				It("returns error", func() {
					_, err := haproxy.NewHaProxyConfigurer(logger, haproxyCfgTemplate, 0)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrInvalidStartFrontendPort))
				})
			})
			Context("when the frontend port is less than 1024", func() {
				It("returns error", func() {
					_, err := haproxy.NewHaProxyConfigurer(logger, haproxyCfgTemplate, 80)
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrInvalidStartFrontendPort))
				})
			})
		})
		Context("when configuration file does not exist", func() {
			It("returns a ErrRouterConfigFileNotFound error", func() {
				_, err := haproxy.NewHaProxyConfigurer(logger, "file/path/does/not/exists", startPort)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(cf_tcp_router.ErrRouterConfigFileNotFound))
			})
		})

		Context("when valid mapping request and config file is passed", func() {
			var (
				haproxyCfgFile            string
				haproxyCfgBackupFile      string
				err                       error
				haproxyCfgTemplateContent []byte
			)

			BeforeEach(func() {
				rand.Seed(17 * time.Now().UTC().UnixNano())
				haproxyCfgFile = fmt.Sprintf("fixtures/haproxy_%d.cfg", rand.Int31())
				haproxyCfgBackupFile = fmt.Sprintf("%s.bak", haproxyCfgFile)
				utils.CopyFile(haproxyCfgTemplate, haproxyCfgFile)

				haproxyCfgTemplateContent, err = ioutil.ReadFile(haproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, haproxyCfgFile, startPort)
				Expect(err).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Remove(haproxyCfgFile)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(utils.FileExists(haproxyCfgBackupFile)).To(BeTrue())
				err = os.Remove(haproxyCfgBackupFile)
				Expect(err).ShouldNot(HaveOccurred())
			})

			Context("when CreateExternalPortMappings is called once", func() {
				Context("when only one mapping is provided as part of request", func() {

					BeforeEach(func() {
						backends := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
						}
						err = haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(2222, backends),
						})
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("appends the haproxy config with new listen configuration", func() {
						listenCfg :=
							"\nlisten listen_cfg_2222\n  mode tcp\n  bind :2222\n  server server_some-ip-1_0 some-ip-1:1234\n  server server_some-ip-2_1 some-ip-2:1235\n"
						verifyHaProxyConfigContent(haproxyCfgFile, listenCfg)
					})
				})

				Context("when multiple mappings are provided as part of one request", func() {

					BeforeEach(func() {
						backends1 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
						}
						backends2 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
						}
						err = haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(2222, backends1),
							cf_tcp_router.NewMappingRequest(3333, backends2),
						})
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("appends the haproxy config with new listen configuration", func() {
						listenCfg1 := `
listen listen_cfg_2222
  mode tcp
  bind :2222
  server server_some-ip-1_0 some-ip-1:1234
  server server_some-ip-2_1 some-ip-2:1235
`
						listenCfg2 := `
listen listen_cfg_3333
  mode tcp
  bind :3333
  server server_some-ip-3_0 some-ip-3:1234
  server server_some-ip-4_1 some-ip-4:1235
`
						verifyHaProxyConfigContent(haproxyCfgFile, listenCfg1)
						verifyHaProxyConfigContent(haproxyCfgFile, listenCfg2)
						verifyHaProxyConfigContent(haproxyCfgFile, string(haproxyCfgTemplateContent))
					})
				})

				Context("when multiple mappings with same external port are provided as part of one request", func() {

					BeforeEach(func() {
						backends1 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
						}
						backends2 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
						}
						err = haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(2222, backends1),
							cf_tcp_router.NewMappingRequest(2222, backends2),
						})
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("appends the haproxy config with new listen configuration", func() {
						listenCfg := `
listen listen_cfg_2222
  mode tcp
  bind :2222
  server server_some-ip-3_0 some-ip-3:1234
  server server_some-ip-4_1 some-ip-4:1235
  server server_some-ip-1_2 some-ip-1:1234
  server server_some-ip-2_3 some-ip-2:1235
`
						verifyHaProxyConfigContent(haproxyCfgFile, listenCfg)
						verifyHaProxyConfigContent(haproxyCfgFile, string(haproxyCfgTemplateContent))
					})
				})
			})

			Context("when CreateExternalPortMappings is called multiple times", func() {
				BeforeEach(func() {

					backends1 := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
					}

					err = haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{
						cf_tcp_router.NewMappingRequest(2222, backends1),
					})
					Expect(err).ShouldNot(HaveOccurred())

					backends2 := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-3", 2345),
						cf_tcp_router.NewBackendHostInfo("some-ip-4", 3456),
					}
					err = haproxyConfigurer.CreateExternalPortMappings(cf_tcp_router.MappingRequests{
						cf_tcp_router.NewMappingRequest(3333, backends2),
					})
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("appends the haproxy config with new listen configuration for all the calls", func() {
					listenCfg := `
listen listen_cfg_2222
  mode tcp
  bind :2222
  server server_some-ip-1_0 some-ip-1:1234
  server server_some-ip-2_1 some-ip-2:1235

listen listen_cfg_3333
  mode tcp
  bind :3333
  server server_some-ip-3_0 some-ip-3:2345
  server server_some-ip-4_1 some-ip-4:3456
`
					verifyHaProxyConfigContent(haproxyCfgFile, listenCfg)
					verifyHaProxyConfigContent(haproxyCfgFile, string(haproxyCfgTemplateContent))
				})
			})

			Context("when CreateExternalPortMappings is called multiple times concurrently", func() {
				type cfgInfo struct {
					port uint16
					cfg  string
				}
				var (
					numCalls   int
					cfgChannel chan cfgInfo
				)

				BeforeEach(func() {
					numCalls = 5
					cfgChannel = make(chan cfgInfo, numCalls)

					for i := 0; i < numCalls; i++ {
						go func(indx int) {
							defer GinkgoRecover()
							ip1 := fmt.Sprintf("some-ip-%d", 2*indx)
							ip2 := fmt.Sprintf("some-ip-%d", 2*indx+1)
							externalPort := uint16(2220 + indx)
							err := haproxyConfigurer.CreateExternalPortMappings(
								cf_tcp_router.MappingRequests{
									cf_tcp_router.NewMappingRequest(externalPort, cf_tcp_router.BackendHostInfos{
										cf_tcp_router.NewBackendHostInfo(ip1, 1234),
										cf_tcp_router.NewBackendHostInfo(ip2, 1235),
									}),
								})
							Expect(err).ShouldNot(HaveOccurred())

							listenCfg := fmt.Sprintf(
								"listen listen_cfg_%d\n  mode tcp\n  bind :%d\n  server server_%s_0 %s:1234\n  server server_%s_1 %s:1235",
								externalPort, externalPort,
								ip1, ip1, ip2, ip2)
							cfg := cfgInfo{
								port: externalPort,
								cfg:  listenCfg,
							}
							cfgChannel <- cfg
						}(i)
					}
				})

				It("does not handout duplicate frontend ports and persists listen configuration", func() {
					portMap := make(map[uint16]string)
					for i := 0; i < numCalls; i++ {
						select {
						case cfg := <-cfgChannel:
							Expect(portMap).ToNot(HaveKey(cfg.port))
							portMap[cfg.port] = cfg.cfg
						}
					}
					data, err := ioutil.ReadFile(haproxyCfgFile)
					Expect(err).ShouldNot(HaveOccurred())
					for _, listenCfg := range portMap {
						Expect(string(data)).To(ContainSubstring(listenCfg))
					}
					Expect(string(data)).To(ContainSubstring(string(haproxyCfgTemplateContent)))
				})
			})
		})
	})
})
