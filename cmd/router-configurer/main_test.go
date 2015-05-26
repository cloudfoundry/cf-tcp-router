package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Main", func() {

	Describe("ExternalPortMapHandler", func() {
		var externalIP string

		BeforeEach(func() {
			externalIP = testutil.GetExternalIP()
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
				Expect(routerHostInfo.Port).To(BeNumerically(">=", 0))
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

})
