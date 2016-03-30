package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/cmd/router-configurer/testrunner"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
	"github.com/cloudfoundry-incubator/cf-tcp-router/utils"
	"github.com/cloudfoundry-incubator/routing-api"
	routingtestrunner "github.com/cloudfoundry-incubator/routing-api/cmd/routing-api/testrunner"
	"github.com/cloudfoundry-incubator/routing-api/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Main", func() {

	var (
		routerGroupGuid string
	)

	getServerPort := func(serverURL string) string {
		endpoints := strings.Split(serverURL, ":")
		Expect(endpoints).To(HaveLen(3))
		return endpoints[2]
	}

	oAuthServer := func(logger lager.Logger) *ghttp.Server {
		server := ghttp.NewTLSServer()
		server.AllowUnhandledRequests = true
		server.RouteToHandler("POST", "/oauth/token",
			func(w http.ResponseWriter, req *http.Request) {
				jsonBytes := []byte(`{"access_token":"some-token", "expires_in":10}`)
				w.Write(jsonBytes)
			},
		)
		logger.Info("starting-oauth-server", lager.Data{"address": server.URL()})
		return server
	}

	getRouterGroupGuid := func(port uint16) string {
		client := routing_api.NewClient(fmt.Sprintf("http://127.0.0.1:%d", port))
		routerGroups, err := client.RouterGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(routerGroups).ToNot(HaveLen(0))
		return routerGroups[0].Guid
	}

	routingApiServer := func(logger lager.Logger) ifrit.Process {
		server := routingtestrunner.New(routingAPIBinPath, routingAPIArgs)
		logger.Info("starting-routing-api-server")
		process := ifrit.Invoke(server)
		routerGroupGuid = getRouterGroupGuid(routingAPIArgs.Port)
		return process
	}

	generateConfigFile := func(oauthServerPort, routingApiServerPort string, routingApiAuthDisabled bool) string {
		randomConfigFileName := testutil.RandomFileName("router_configurer", ".yml")
		configFile := path.Join(os.TempDir(), randomConfigFileName)

		cfg := fmt.Sprintf("%s\n  port: %s\n%s\n  auth_disabled: %t\n  %s\n  port: %s\n", `oauth:
  token_endpoint: "127.0.0.1"
  skip_oauth_tls_verification: true
  client_name: "someclient"
  client_secret: "somesecret"`, oauthServerPort,
			`routing_api:`,
			routingApiAuthDisabled,
			`uri: http://127.0.0.1`, routingApiServerPort)
		err := utils.WriteToFile([]byte(cfg), configFile)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(utils.FileExists(configFile)).To(BeTrue())
		return configFile
	}

	verifyHaProxyConfigContent := func(haproxyFileName, expectedContent string) {
		data, err := ioutil.ReadFile(haproxyFileName)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(string(data)).Should(ContainSubstring(expectedContent))
	}

	var (
		externalIP  string
		oauthServer *ghttp.Server
		server      ifrit.Process
		logger      *lagertest.TestLogger
		session     *gexec.Session
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		externalIP = testutil.GetExternalIP()
	})

	Context("when both oauth and routing api servers are up and running", func() {
		BeforeEach(func() {
			oauthServer = oAuthServer(logger)
			server = routingApiServer(logger)
			oauthServerPort := getServerPort(oauthServer.URL())
			configFile := generateConfigFile(oauthServerPort, fmt.Sprintf("%d", routingAPIPort), false)
			routerConfigurerArgs := testrunner.Args{
				BaseLoadBalancerConfigFilePath: haproxyBaseConfigFile,
				LoadBalancerConfigFilePath:     haproxyConfigFile,
				ConfigFilePath:                 configFile,
			}

			tcpRouteMapping := models.TcpRouteMapping{
				TcpRoute: models.TcpRoute{
					RouterGroupGuid: routerGroupGuid,
					ExternalPort:    5222,
				},
				HostPort: 61000,
				HostIP:   "some-ip-1",
			}
			err := routingApiClient.UpsertTcpRouteMappings([]models.TcpRouteMapping{tcpRouteMapping})
			Expect(err).ToNot(HaveOccurred())

			tcpRouteMappings, err := routingApiClient.TcpRouteMappings()
			Expect(err).NotTo(HaveOccurred())
			Expect(tcpRouteMappings).To(ContainElement(tcpRouteMapping))

			allOutput := logger.Buffer()
			runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
			session, err = gexec.Start(runner.Command, allOutput, allOutput)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			logger.Info("shutting-down")
			session.Signal(os.Interrupt)
			Eventually(session.Exited, 5*time.Second).Should(BeClosed())
			server.Signal(os.Interrupt)
			Eventually(server.Wait(), 5*time.Second).Should(Receive())
			oauthServer.Close()
		})

		It("syncs with routing api", func() {
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("applied-fetched-routes-to-routing-table"))
			expectedConfigEntry := "\nlisten listen_cfg_5222\n  mode tcp\n  bind :5222\n"
			serverConfigEntry := "server server_some-ip-1_61000 some-ip-1:61000"
			verifyHaProxyConfigContent(haproxyConfigFile, expectedConfigEntry)
			verifyHaProxyConfigContent(haproxyConfigFile, serverConfigEntry)
		})

		It("starts an SSE connection to the server", func() {
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("Subscribing-to-routing-api-event-stream"))
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("Successfully-subscribed-to-routing-api-event-stream"))
			tcpRouteMapping := models.TcpRouteMapping{
				TcpRoute: models.TcpRoute{
					RouterGroupGuid: routerGroupGuid,
					ExternalPort:    5222,
				},
				HostPort: 61000,
				HostIP:   "some-ip-2",
			}
			err := routingApiClient.UpsertTcpRouteMappings([]models.TcpRouteMapping{tcpRouteMapping})
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("handle-upsert-done"))
			expectedConfigEntry := "\nlisten listen_cfg_5222\n  mode tcp\n  bind :5222\n"
			verifyHaProxyConfigContent(haproxyConfigFile, expectedConfigEntry)
			oldServerConfigEntry := "server server_some-ip-1_61000 some-ip-1:61000"
			verifyHaProxyConfigContent(haproxyConfigFile, oldServerConfigEntry)
			newServerConfigEntry := "server server_some-ip-2_61000 some-ip-2:61000"
			verifyHaProxyConfigContent(haproxyConfigFile, newServerConfigEntry)
		})

	})

	Context("Oauth server is down", func() {
		var (
			routerConfigurerArgs testrunner.Args
			configFile           string
			oauthServerPort      string
		)
		BeforeEach(func() {
			server = routingApiServer(logger)
			oauthServerPort = "1111"

		})

		AfterEach(func() {
			logger.Info("shutting-down")
			server.Signal(os.Interrupt)
			Eventually(server.Wait(), 5*time.Second).Should(Receive())
		})

		JustBeforeEach(func() {
			allOutput := logger.Buffer()
			runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
			var err error
			session, err = gexec.Start(runner.Command, allOutput, allOutput)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("routing api auth is enabled", func() {
			BeforeEach(func() {
				configFile = generateConfigFile(oauthServerPort, fmt.Sprintf("%d", routingAPIPort), false)
				routerConfigurerArgs = testrunner.Args{
					BaseLoadBalancerConfigFilePath: haproxyBaseConfigFile,
					LoadBalancerConfigFilePath:     haproxyConfigFile,
					ConfigFilePath:                 configFile,
				}
			})

			It("exits with error", func() {
				Eventually(session.Out, 5*time.Second).Should(gbytes.Say("error-fetching-oauth-token"))
				Eventually(session.Exited).Should(BeClosed())
			})
		})

		Context("routing api auth is disabled", func() {
			BeforeEach(func() {
				configFile = generateConfigFile(oauthServerPort, fmt.Sprintf("%d", routingAPIPort), true)
				routerConfigurerArgs = testrunner.Args{
					BaseLoadBalancerConfigFilePath: haproxyBaseConfigFile,
					LoadBalancerConfigFilePath:     haproxyConfigFile,
					ConfigFilePath:                 configFile,
				}
			})

			AfterEach(func() {
				session.Signal(os.Interrupt)
				Eventually(session.Exited, 5*time.Second).Should(BeClosed())
			})

			It("does not call oauth server to get auth token and starts SSE connection with routing api", func() {
				Eventually(session.Out, 5*time.Second).Should(gbytes.Say("creating-noop-uaa-client"))
				Eventually(session.Out, 5*time.Second).Should(gbytes.Say("Successfully-subscribed-to-routing-api-event-stream"))
			})
		})
	})

	Context("Routing API server is down", func() {
		BeforeEach(func() {
			oauthServer = oAuthServer(logger)
			oauthServerPort := getServerPort(oauthServer.URL())
			configFile := generateConfigFile(oauthServerPort, fmt.Sprintf("%d", routingAPIPort), false)
			routerConfigurerArgs := testrunner.Args{
				BaseLoadBalancerConfigFilePath: haproxyBaseConfigFile,
				LoadBalancerConfigFilePath:     haproxyConfigFile,
				ConfigFilePath:                 configFile,
			}
			allOutput := logger.Buffer()
			runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
			var err error
			session, err = gexec.Start(runner.Command, allOutput, allOutput)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			logger.Info("shutting-down")
			session.Signal(os.Interrupt)
			Eventually(session.Exited, 5*time.Second).Should(BeClosed())
			oauthServer.Close()
			server.Signal(os.Interrupt)
			Eventually(server.Wait(), 5*time.Second).Should(Receive())
		})

		It("keeps trying to connect and doesn't blow up", func() {
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("Subscribing-to-routing-api-event-stream"))
			Consistently(session.Exited).ShouldNot(BeClosed())
			Consistently(session.Out, 5*time.Second).ShouldNot(gbytes.Say("Successfully-subscribed-to-routing-api-event-stream"))
			By("starting routing api server")
			server = routingApiServer(logger)
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("Successfully-subscribed-to-routing-api-event-stream"))
			tcpRouteMapping := models.TcpRouteMapping{
				TcpRoute: models.TcpRoute{
					RouterGroupGuid: routerGroupGuid,
					ExternalPort:    5222,
				},
				HostPort: 61000,
				HostIP:   "some-ip-3",
			}
			err := routingApiClient.UpsertTcpRouteMappings([]models.TcpRouteMapping{tcpRouteMapping})
			Expect(err).ToNot(HaveOccurred())
			Eventually(session.Out, 5*time.Second).Should(gbytes.Say("handle-upsert-done"))
			expectedConfigEntry := "\nlisten listen_cfg_5222\n  mode tcp\n  bind :5222\n"
			verifyHaProxyConfigContent(haproxyConfigFile, expectedConfigEntry)
			newServerConfigEntry := "server server_some-ip-3_61000 some-ip-3:61000"
			verifyHaProxyConfigContent(haproxyConfigFile, newServerConfigEntry)
		})
	})

})
