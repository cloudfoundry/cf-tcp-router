package watcher_test

import (
	"errors"
	"os"
	"time"

	fake_routing_table "code.cloudfoundry.org/cf-tcp-router/routing_table/fakes"
	"code.cloudfoundry.org/cf-tcp-router/watcher"
	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/fake_routing_api"
	"code.cloudfoundry.org/routing-api/models"
	test_uaa_client "code.cloudfoundry.org/routing-api/uaaclient/fakes"
	"github.com/tedsuo/ifrit"
	"golang.org/x/oauth2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Watcher", func() {
	const (
		routerGroupGuid = "rtrGrp0001"
	)
	var (
		eventSource      *fake_routing_api.FakeTcpEventSource
		routingApiClient *fake_routing_api.FakeClient
		uaaTokenFetcher  *test_uaa_client.FakeTokenFetcher
		testWatcher      *watcher.Watcher
		process          ifrit.Process
		syncChannel      chan struct{}
		updater          *fake_routing_table.FakeUpdater
	)

	BeforeEach(func() {
		eventSource = new(fake_routing_api.FakeTcpEventSource)
		routingApiClient = new(fake_routing_api.FakeClient)
		updater = new(fake_routing_table.FakeUpdater)
		uaaTokenFetcher = &test_uaa_client.FakeTokenFetcher{}
		token := &oauth2.Token{
			AccessToken: "access_token",
			Expiry:      time.Now().Add(5 * time.Second),
		}
		uaaTokenFetcher.FetchTokenReturns(token, nil)

		routingApiClient.SubscribeToTcpEventsReturns(eventSource, nil)
		syncChannel = make(chan struct{})
		testWatcher = watcher.New(routingApiClient, updater, uaaTokenFetcher, 1, syncChannel, logger)
	})

	JustBeforeEach(func() {
		process = ifrit.Invoke(testWatcher)
	})

	AfterEach(func() {
		process.Signal(os.Interrupt)
		Eventually(process.Wait()).Should(Receive())
		Eventually(logger).Should(gbytes.Say("test.watcher.stopping"))
	})

	Context("handle UpsertEvent", func() {
		var (
			tcpEvent routing_api.TcpEvent
		)

		BeforeEach(func() {
			tcpEvent = routing_api.TcpEvent{
				TcpRouteMapping: models.NewTcpRouteMapping(
					routerGroupGuid,
					61000,
					"some-ip-1",
					5222,
					0,
					"",
					nil,
					0,
					models.ModificationTag{},
				),
				Action: "Upsert",
			}
			eventSource.NextReturns(tcpEvent, nil)
		})

		It("calls updater HandleEvent", func() {
			Eventually(updater.HandleEventCallCount).Should(BeNumerically(">=", 1))
			upsertEvent := updater.HandleEventArgsForCall(0)
			Expect(upsertEvent).Should(Equal(tcpEvent))
		})
	})

	Context("handle DeleteEvent", func() {
		var (
			tcpEvent routing_api.TcpEvent
		)

		BeforeEach(func() {
			tcpEvent = routing_api.TcpEvent{
				TcpRouteMapping: models.NewTcpRouteMapping(
					routerGroupGuid,
					61000,
					"some-ip-1",
					5222,
					0,
					"",
					nil,
					0,
					models.ModificationTag{},
				),
				Action: "Delete",
			}
			eventSource.NextReturns(tcpEvent, nil)
		})

		It("calls updater HandleEvent", func() {
			Eventually(updater.HandleEventCallCount).Should(BeNumerically(">=", 1))
			deleteEvent := updater.HandleEventArgsForCall(0)
			Expect(deleteEvent).Should(Equal(tcpEvent))
		})
	})

	Context("handle Sync Event", func() {
		JustBeforeEach(func() {
			syncChannel <- struct{}{}
		})

		It("calls updater Sync", func() {
			Eventually(updater.SyncCallCount).Should(Equal(1))
		})
	})

	Context("when eventSource returns error", func() {
		BeforeEach(func() {
			eventSource.NextReturns(routing_api.TcpEvent{}, errors.New("buzinga.."))
		})

		It("resubscribes to SSE from routing api", func() {
			Eventually(routingApiClient.SubscribeToTcpEventsCallCount).Should(BeNumerically(">=", 2))
			Eventually(logger).Should(gbytes.Say("failed-to-get-next-routing-api-event"))
		})

		It("closes the current eventSource", func() {
			Eventually(eventSource.CloseCallCount).Should(BeNumerically(">=", 1))
		})
	})

	Context("when subscribe to events fails", func() {
		var (
			routingApiErrChannel chan error
		)
		BeforeEach(func() {
			routingApiErrChannel = make(chan error)

			routingApiClient.SubscribeToTcpEventsStub = func() (routing_api.TcpEventSource, error) {
				err := <-routingApiErrChannel
				if err != nil {
					return nil, err
				}
				return eventSource, nil
			}

			testWatcher = watcher.New(routingApiClient, updater, uaaTokenFetcher, 1, syncChannel, logger)
		})

		Context("with error other than unauthorized", func() {
			It("uses the cached token and retries to subscribe", func() {
				Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(1))
				_, forceUpdate := uaaTokenFetcher.FetchTokenArgsForCall(0)
				Expect(forceUpdate).To(BeFalse())
				routingApiErrChannel <- errors.New("kaboom")
				close(routingApiErrChannel)
				Eventually(routingApiClient.SubscribeToTcpEventsCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
				Eventually(logger).Should(gbytes.Say("failed-subscribing-to-routing-api-event-stream"))
				Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
				_, forceUpdate = uaaTokenFetcher.FetchTokenArgsForCall(1)
				Expect(forceUpdate).To(BeFalse())
			})
		})

		Context("with unauthorized error", func() {
			It("fetches a new token and retries to subscribe", func() {
				Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(1))
				_, forceUpdate := uaaTokenFetcher.FetchTokenArgsForCall(0)
				Expect(forceUpdate).To(BeFalse())
				routingApiErrChannel <- errors.New("unauthorized")
				Eventually(routingApiClient.SubscribeToTcpEventsCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
				Eventually(logger).Should(gbytes.Say("failed-subscribing-to-routing-api-event-stream"))
				Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
				_, forceUpdate = uaaTokenFetcher.FetchTokenArgsForCall(1)
				Expect(forceUpdate).To(BeTrue())

				By("resumes to use cache token for subsequent errors")
				routingApiErrChannel <- errors.New("kaboom")
				close(routingApiErrChannel)
				Eventually(routingApiClient.SubscribeToTcpEventsCallCount, 5*time.Second, 1*time.Second).Should(Equal(3))
				Eventually(logger).Should(gbytes.Say("failed-subscribing-to-routing-api-event-stream"))
				Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(3))
				_, forceUpdate = uaaTokenFetcher.FetchTokenArgsForCall(2)
				Expect(forceUpdate).To(BeFalse())
			})
		})
	})

	Context("when the token fetcher returns an error", func() {
		BeforeEach(func() {
			uaaTokenFetcher.FetchTokenReturns(nil, errors.New("token fetcher error"))
		})

		It("returns an error", func() {
			Eventually(logger).Should(gbytes.Say("error-fetching-token"))
			Eventually(uaaTokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(BeNumerically(">", 2))
		})
	})

})
