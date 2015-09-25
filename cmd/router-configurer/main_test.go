package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/cmd/router-configurer/testrunner"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
	"github.com/cloudfoundry-incubator/cf-tcp-router/utils"
	"github.com/cloudfoundry-incubator/routing-api/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/vito/go-sse/sse"
)

var _ = Describe("Main", func() {

	Describe("ExternalPortMapHandler", func() {
		var (
			externalIP string
			server     *ghttp.Server
			event      sse.Event
			logger     lager.Logger
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("test")
			externalIP = testutil.GetExternalIP()
			tcpRoute1 := db.NewTcpRouteMapping("rguid1", 52000, "1.1.1.1", 60000)

			data, err := json.Marshal(tcpRoute1)
			Expect(err).ToNot(HaveOccurred())
			event = sse.Event{
				ID:   "1",
				Name: "Upsert",
				Data: data,
			}
		})

		Context("when valid arguments are passed", func() {
			BeforeEach(func() {
				server = ghttp.NewServer()
				server.AllowUnhandledRequests = true
				routingApiEndpoints := strings.Split(server.URL(), ":")
				Expect(routingApiEndpoints).To(HaveLen(3))

				randomConfigFileName := testutil.RandomFileName("router_configurer", ".yml")
				configFile := path.Join(os.TempDir(), randomConfigFileName)

				cfg := fmt.Sprintf("%s\n  port: %s\n%s\n  port: %s\n", `oauth:
  token_endpoint: "http://127.0.0.1"
  client_name: "someclient"
  client_secret: "somesecret"`, routingApiEndpoints[2],
					`routing_api:
  uri: http://127.0.0.1`, routingApiEndpoints[2])
				err := utils.WriteToFile([]byte(cfg), configFile)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(utils.FileExists(configFile)).To(BeTrue())

				routerConfigurerArgs := testrunner.Args{
					Address: fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
					BaseLoadBalancerConfigFilePath: haproxyCfgTemplate,
					LoadBalancerConfigFilePath:     haproxyConfigFile,
					ConfigFilePath:                 configFile,
				}

				logger.Info("starting-server", lager.Data{"address": server.URL()})
				runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/oauth/token"),
						func(w http.ResponseWriter, req *http.Request) {
							jsonBytes := []byte(`{"access_token":"some-token", "expires_in":10}`)
							w.Write(jsonBytes)
						},
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routing/v1/tcp_routes/events"),
						ghttp.VerifyHeader(http.Header{
							"Authorization": []string{"bearer some-token"},
						}),
						func(w http.ResponseWriter, req *http.Request) {
							event.Write(w)
						},
					),
				)
				routerConfigurerProcess = ifrit.Invoke(runner)
			})

			AfterEach(func() {
				logger.Info("shutting-down")
				routerConfigurerProcess.Signal(os.Interrupt)
				Eventually(routerConfigurerProcess.Wait(), 5*time.Second).Should(Receive())
				server.Close()
			})

			It("Starts an SSE connection to the server", func() {
				Eventually(func() int {
					requests := make([]*http.Request, 0)
					receivedRequests := server.ReceivedRequests()
					for _, req := range receivedRequests {
						if strings.Contains(req.RequestURI, "routing/v1/tcp_routes/events") {
							requests = append(requests, req)
						}
					}
					return len(requests)
				}, 5*time.Second).Should(BeNumerically(">=", 1))
			})

			Context("when valid backend host info is passed", func() {
				It("should return valid external IP and Port", func() {
					backendHostInfos := `[
					{"external_port":2222,
					"backends":[
						{"ip": "some-ip", "port":1234},
						{"ip": "some-ip-1", "port":12345}
					]}]`
					payload := []byte(backendHostInfos)
					resp, err := http.Post(
						fmt.Sprintf("http://127.0.0.1:%d/v0/external_ports", routerConfigurerPort),
						"application/json", bytes.NewBuffer(payload))
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resp.StatusCode).Should(Equal(http.StatusOK))
				})
			})

			Context("when malformed json is passed", func() {
				It("should return 400", func() {
					backendHostInfos := `{abcd`
					payload := []byte(backendHostInfos)
					resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/v0/external_ports",
						routerConfigurerPort), "application/json", bytes.NewBuffer(payload))
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
				})
			})
		})
	})
})
