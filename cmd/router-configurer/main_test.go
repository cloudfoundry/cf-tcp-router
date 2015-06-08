package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/cmd/router-configurer/testrunner"
	"github.com/GESoftware-CF/cf-tcp-router/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Main", func() {

	Describe("ExternalPortMapHandler", func() {
		var externalIP string

		BeforeEach(func() {
			externalIP = testutil.GetExternalIP()
		})

		Context("when valid arguments are passed", func() {
			BeforeEach(func() {
				routerConfigurerArgs := testrunner.Args{
					Address:           fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
					ConfigFilePath:    haproxyConfigFile,
					StartFrontendPort: startFrontendPort,
				}

				runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
				routerConfigurerProcess = ifrit.Invoke(runner)
			})

			AfterEach(func() {
				ginkgomon.Kill(routerConfigurerProcess, 5*time.Second)
			})

			Context("when valid backend host info is passed", func() {
				It("should return valid external IP and Port", func() {
					backendHostInfos := `[{"backend_ip": "some-ip", "backend_port":1234}, {"backend_ip": "some-ip-1", "backend_port":12345}]`
					payload := []byte(backendHostInfos)
					resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/v0/external_ports", routerConfigurerPort), "application/json", bytes.NewBuffer(payload))
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
					responseBody, err := ioutil.ReadAll(resp.Body)
					Expect(err).ShouldNot(HaveOccurred())
					var routerHostInfo cf_tcp_router.RouterHostInfo
					err = json.Unmarshal(responseBody, &routerHostInfo)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(routerHostInfo.Address).Should(Equal(externalIP))
					Expect(routerHostInfo.Port).To(BeNumerically(">=", startFrontendPort))
					Expect(routerHostInfo.Port).To(BeNumerically("<", 65536))
				})
			})

			Context("when malformed json is passed", func() {
				It("should return 400", func() {
					backendHostInfos := `{abcd`
					payload := []byte(backendHostInfos)
					resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/v0/external_ports", routerConfigurerPort), "application/json", bytes.NewBuffer(payload))
					Expect(err).ShouldNot(HaveOccurred())
					Expect(resp.StatusCode).Should(Equal(http.StatusBadRequest))
				})
			})
		})

		Context("when start frontend port is invalid", func() {
			var routerConfigurerArgs testrunner.Args
			var readyChan <-chan struct{}
			var errorChan <-chan error

			JustBeforeEach(func() {
				runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
				routerConfigurerProcess = ifrit.Invoke(runner)
				errorChan = routerConfigurerProcess.Wait()
				readyChan = routerConfigurerProcess.Ready()
			})

			Context("when start frontend port is greater than 65535", func() {
				BeforeEach(func() {
					routerConfigurerArgs = testrunner.Args{
						Address:           fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
						ConfigFilePath:    haproxyConfigFile,
						StartFrontendPort: 70000,
					}
				})

				AfterEach(func() {
					ginkgomon.Kill(routerConfigurerProcess, 5*time.Second)
				})

				It("should fail starting the process", func() {
					Eventually(errorChan).Should(Receive())
					Expect(readyChan).ShouldNot(BeClosed())
				})
			})

		})
	})
})
