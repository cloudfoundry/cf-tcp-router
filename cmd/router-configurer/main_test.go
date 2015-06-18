package main_test

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/cmd/router-configurer/testrunner"
	"github.com/cloudfoundry-incubator/cf-tcp-router/testutil"
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
					Address:            fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
					BaseConfigFilePath: haproxyCfgTemplate,
					ConfigFilePath:     haproxyConfigFile,
				}

				runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
				routerConfigurerProcess = ifrit.Invoke(runner)
			})

			AfterEach(func() {
				ginkgomon.Kill(routerConfigurerProcess, 5*time.Second)
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
