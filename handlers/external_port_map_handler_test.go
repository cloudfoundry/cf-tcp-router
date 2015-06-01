package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/configurer/fakes"
	"github.com/GESoftware-CF/cf-tcp-router/handlers"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("ExternalPortMapHandler", func() {
	var (
		handler          *handlers.ExternalPortMapHandler
		logger           lager.Logger
		responseRecorder *httptest.ResponseRecorder
		fakeConfigurer   *fakes.FakeRouterConfigurer
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		responseRecorder = httptest.NewRecorder()
		fakeConfigurer = new(fakes.FakeRouterConfigurer)
		handler = handlers.NewExternalPortMapHandler(logger, fakeConfigurer)
	})

	Describe("MapExternalPort", func() {
		var (
			backendHostInfos cf_tcp_router.BackendHostInfos
		)
		BeforeEach(func() {
			backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 1234)
			backendHostInfos = cf_tcp_router.BackendHostInfos{backendHostInfo}
		})

		JustBeforeEach(func() {
			handler.MapExternalPort(responseRecorder, newTestRequest(backendHostInfos))
		})

		Context("when request is valid", func() {
			var expectedRouterHostInfo cf_tcp_router.RouterHostInfo
			BeforeEach(func() {
				expectedRouterHostInfo = cf_tcp_router.RouterHostInfo{
					Address: "some-ip",
					Port:    23456,
				}
				fakeConfigurer.MapBackendHostsToAvailablePortReturns(expectedRouterHostInfo, nil)
			})

			It("responds with 201 CREATED", func() {
				Expect(responseRecorder.Code).To(Equal(http.StatusCreated))
			})

			It("returns router host info", func() {
				payload, err := json.Marshal(expectedRouterHostInfo)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(responseRecorder.Body.String()).To(MatchJSON(string(payload)))
			})

			Context("when configurer returns an error", func() {
				BeforeEach(func() {
					fakeConfigurer.MapBackendHostsToAvailablePortReturns(cf_tcp_router.RouterHostInfo{}, errors.New("Kabooom"))
				})

				It("responds with 500 INTERNAL_SERVER_ERROR", func() {
					Expect(responseRecorder.Code).To(Equal(http.StatusInternalServerError))
					Eventually(logger).Should(gbytes.Say("test.map_external_port.failed-to-configure"))
				})
			})
		})

		Context("when request is invalid", func() {
			Context("when payload is not a valid json", func() {
				BeforeEach(func() {
					handler.MapExternalPort(responseRecorder, newTestRequest(`{abcd`))
				})

				It("responds with 400 BAD REQUEST", func() {
					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Eventually(logger).Should(gbytes.Say("test.map_external_port.failed-to-unmarshal"))
				})
			})

			Context("when payload has invalid values", func() {
				BeforeEach(func() {
					backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 0)
					backendHostInfos := cf_tcp_router.BackendHostInfos{backendHostInfo}
					fakeConfigurer.MapBackendHostsToAvailablePortReturns(cf_tcp_router.RouterHostInfo{}, errors.New(cf_tcp_router.ErrInvalidBackendHostInfo))
					handler.MapExternalPort(responseRecorder, newTestRequest(backendHostInfos))
				})

				It("responds with 400 BAD REQUEST", func() {
					Expect(responseRecorder.Code).To(Equal(http.StatusBadRequest))
					Eventually(logger).Should(gbytes.Say("test.map_external_port.invalid-payload"))
				})
			})
		})
	})
})
