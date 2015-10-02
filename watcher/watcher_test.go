package watcher_test

import (
	"errors"
	"os"
	"time"

	fake_routing_table "github.com/cloudfoundry-incubator/cf-tcp-router/routing_table/fakes"
	"github.com/cloudfoundry-incubator/cf-tcp-router/watcher"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	"github.com/cloudfoundry-incubator/uaa-token-fetcher"
	testTokenFetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher/fakes"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
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
		tokenFetcher     *testTokenFetcher.FakeTokenFetcher
		testWatcher      *watcher.Watcher
		process          ifrit.Process
		eventChannel     chan routing_api.TcpEvent
		errorChannel     chan error
		syncChannel         chan struct{}
		updater          *fake_routing_table.FakeUpdater
	)

	BeforeEach(func() {
		eventSource = new(fake_routing_api.FakeTcpEventSource)
		routingApiClient = new(fake_routing_api.FakeClient)
		updater = new(fake_routing_table.FakeUpdater)
		tokenFetcher = &testTokenFetcher.FakeTokenFetcher{}
		token := &token_fetcher.Token{
			AccessToken: "access_token",
			ExpireTime:  5,
		}
		tokenFetcher.FetchTokenReturns(token, nil)

		routingApiClient.SubscribeToTcpEventsReturns(eventSource, nil)
		syncChannel = make(chan struct{})
		testWatcher = watcher.New(routingApiClient, updater, tokenFetcher, 1, syncChannel, logger)

		eventChannel = make(chan routing_api.TcpEvent)
		errorChannel = make(chan error)

		eventSource.CloseStub = func() error {
			errorChannel <- errors.New("closed")
			return nil
		}

		eventSource.NextStub = func() (routing_api.TcpEvent, error) {
			select {
			case event := <-eventChannel:
				return event, nil
			case err := <-errorChannel:
				return routing_api.TcpEvent{}, err
			}
			return routing_api.TcpEvent{}, nil
		}
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

		JustBeforeEach(func() {
			tcpEvent = routing_api.TcpEvent{
				TcpRouteMapping: db.TcpRouteMapping{
					TcpRoute: db.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    61000,
					},
					HostPort: 5222,
					HostIP:   "some-ip-1",
				},
				Action: "Upsert",
			}
			eventChannel <- tcpEvent
		})

		It("calls updater HandleEvent", func() {
			Eventually(updater.HandleEventCallCount, 5*time.Second, 300*time.Millisecond).Should(Equal(1))
			upsertEvent := updater.HandleEventArgsForCall(0)
			Expect(upsertEvent).Should(Equal(tcpEvent))
		})
	})

	Context("handle DeleteEvent", func() {
		var (
			tcpEvent routing_api.TcpEvent
		)

		JustBeforeEach(func() {
			tcpEvent = routing_api.TcpEvent{
				TcpRouteMapping: db.TcpRouteMapping{
					TcpRoute: db.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    61000,
					},
					HostPort: 5222,
					HostIP:   "some-ip-1",
				},
				Action: "Delete",
			}
			eventChannel <- tcpEvent
		})

		It("calls updater HandleEvent", func() {
			Eventually(updater.HandleEventCallCount, 5*time.Second, 300*time.Millisecond).Should(Equal(1))
			deleteEvent := updater.HandleEventArgsForCall(0)
			Expect(deleteEvent).Should(Equal(tcpEvent))
		})
	})

	Context("handle Sync Event", func() {
		JustBeforeEach(func() {
			syncChannel <- struct{}{}
		})

		It("calls updater Sync", func() {
			Eventually(updater.SyncCallCount, 5*time.Second, 300*time.Millisecond).Should(Equal(1))
		})
	})

	Context("when eventSource returns error", func() {
		JustBeforeEach(func() {
			Eventually(routingApiClient.SubscribeToTcpEventsCallCount).Should(Equal(1))
			errorChannel <- errors.New("buzinga...")
		})

		It("resubscribes to SSE from routing api", func() {
			Eventually(routingApiClient.SubscribeToTcpEventsCallCount, 5*time.Second, 300*time.Millisecond).Should(Equal(2))
			Eventually(logger).Should(gbytes.Say("test.watcher.failed-getting-next-tcp-routing-event"))
		})
	})

	Context("when subscribe to events fails", func() {
		var (
			routingApiErrChannel chan error
		)
		BeforeEach(func() {
			routingApiErrChannel = make(chan error)

			routingApiClient.SubscribeToTcpEventsStub = func() (routing_api.TcpEventSource, error) {
				select {
				case err := <-routingApiErrChannel:
					if err != nil {
						return nil, err
					}
				}
				return eventSource, nil
			}

			testWatcher = watcher.New(routingApiClient, updater, tokenFetcher, 1, syncChannel, logger)
		})

		JustBeforeEach(func() {
			routingApiErrChannel <- errors.New("kaboom")
		})

		It("retries to subscribe", func() {
			close(routingApiErrChannel)
			Eventually(routingApiClient.SubscribeToTcpEventsCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
			Eventually(logger).Should(gbytes.Say("test.watcher.failed-subscribing-to-tcp-routing-events"))
		})
	})

	Context("when the token fetcher returns an error", func() {
		BeforeEach(func() {
			tokenFetcher.FetchTokenReturns(nil, errors.New("token fetcher error"))
		})

		It("returns an error", func() {
			Eventually(logger).Should(gbytes.Say("test.watcher.error-fetching-token"))
			Eventually(tokenFetcher.FetchTokenCallCount, 5*time.Second, 1*time.Second).Should(Equal(2))
		})
	})

})
