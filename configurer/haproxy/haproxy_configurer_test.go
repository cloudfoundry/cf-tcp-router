package haproxy_test

import (
	"bytes"
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
	Describe("MapBackendHostsToAvailablePort", func() {
		const (
			haproxyCfgTemplate = "fixtures/haproxy.cfg.template"
			startPort          = 61000
		)

		var (
			externalIP        string
			haproxyConfigurer *haproxy.HaProxyConfigurer
		)

		verifyRouterHostInfo := func(routerHostInfo cf_tcp_router.RouterHostInfo) {
			Expect(routerHostInfo.Address).Should(Equal(externalIP))
			Expect(routerHostInfo.Port).Should(BeNumerically("<", 65536))
			Expect(routerHostInfo.Port).Should(BeNumerically(">=", startPort))
		}

		verifyHaProxyConfigContent := func(haproxyFileName, expectedContent string) {
			data, err := ioutil.ReadFile(haproxyFileName)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(data)).Should(Equal(expectedContent))
		}

		BeforeEach(func() {
			externalIP = testutil.GetExternalIP()
		})

		Context("when invalid backend host info is passed", func() {
			BeforeEach(func() {
				var err error
				haproxyConfigurer, err = haproxy.NewHaProxyConfigurer(logger, haproxyCfgTemplate, startPort)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("returns error", func() {
				_, err := haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal(cf_tcp_router.ErrInvalidBackendHostInfo))
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

		Context("when valid backend host info and config file is passed", func() {
			var (
				haproxyCfgFile            string
				haproxyCfgBackupFile      string
				routerHostInfo            cf_tcp_router.RouterHostInfo
				err                       error
				expectedContent           string
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

			Context("when MapBackendHostsToAvailablePort is called once", func() {
				BeforeEach(func() {
					routerHostInfo, err = haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
					})
					Expect(err).ShouldNot(HaveOccurred())

					var buff bytes.Buffer
					_, err = buff.Write(haproxyCfgTemplateContent)
					Expect(err).ShouldNot(HaveOccurred())
					listenCfg := fmt.Sprintf(
						"\nlisten listen_cfg_%d\n  mode tcp\n  bind :%d\n  server server_some-ip-1_0 some-ip-1:1234\n  server server_some-ip-2_1 some-ip-2:1235\n",
						routerHostInfo.Port, routerHostInfo.Port)
					_, err = buff.WriteString(listenCfg)
					Expect(err).ShouldNot(HaveOccurred())

					expectedContent = buff.String()
				})

				It("returns valid router host info and appends the haproxy config with new listen configuration", func() {
					verifyRouterHostInfo(routerHostInfo)
					verifyHaProxyConfigContent(haproxyCfgFile, expectedContent)
					data, err := ioutil.ReadFile(haproxyCfgFile)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(data)).Should(Equal(expectedContent))
				})
			})

			Context("when MapBackendHostsToAvailablePort is called multiple times", func() {
				BeforeEach(func() {
					var routerHostInfo1, routerHostInfo2 cf_tcp_router.RouterHostInfo

					routerHostInfo1, err = haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
					})
					Expect(err).ShouldNot(HaveOccurred())
					verifyRouterHostInfo(routerHostInfo1)

					routerHostInfo2, err = haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-3", 2345),
						cf_tcp_router.NewBackendHostInfo("some-ip-4", 3456),
					})
					Expect(err).ShouldNot(HaveOccurred())
					verifyRouterHostInfo(routerHostInfo2)

					Expect(routerHostInfo1.Port).ShouldNot(Equal(routerHostInfo2.Port))

					var buff bytes.Buffer
					_, err = buff.Write(haproxyCfgTemplateContent)
					Expect(err).ShouldNot(HaveOccurred())

					listenCfg1 := fmt.Sprintf(
						"\nlisten listen_cfg_%d\n  mode tcp\n  bind :%d\n  server server_some-ip-1_0 some-ip-1:1234\n  server server_some-ip-2_1 some-ip-2:1235\n",
						routerHostInfo1.Port, routerHostInfo1.Port)
					listenCfg2 := fmt.Sprintf(
						"\nlisten listen_cfg_%d\n  mode tcp\n  bind :%d\n  server server_some-ip-3_0 some-ip-3:2345\n  server server_some-ip-4_1 some-ip-4:3456\n",
						routerHostInfo2.Port, routerHostInfo2.Port)

					_, err = buff.WriteString(listenCfg1)
					Expect(err).ShouldNot(HaveOccurred())

					_, err = buff.WriteString(listenCfg2)
					Expect(err).ShouldNot(HaveOccurred())

					expectedContent = buff.String()
				})

				It("appends the haproxy config with new listen configuration for all the calls", func() {
					verifyHaProxyConfigContent(haproxyCfgFile, expectedContent)
				})
			})

			Context("when MapBackendHostsToAvailablePort is called multiple times concurrently", func() {
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
							routerHostInfo, err := haproxyConfigurer.MapBackendHostsToAvailablePort(cf_tcp_router.BackendHostInfos{
								cf_tcp_router.NewBackendHostInfo(ip1, 1234),
								cf_tcp_router.NewBackendHostInfo(ip2, 1235),
							})
							Expect(err).ShouldNot(HaveOccurred())
							verifyRouterHostInfo(routerHostInfo)
							listenCfg := fmt.Sprintf(
								"listen listen_cfg_%d\n  mode tcp\n  bind :%d\n  server server_%s_0 %s:1234\n  server server_%s_1 %s:1235",
								routerHostInfo.Port, routerHostInfo.Port,
								ip1, ip1, ip2, ip2)
							cfg := cfgInfo{
								port: routerHostInfo.Port,
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
				})
			})
		})
	})
})
