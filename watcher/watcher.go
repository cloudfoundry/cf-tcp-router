package watcher

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"
	"github.com/cloudfoundry-incubator/routing-api"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	"github.com/pivotal-golang/lager"
)

type Watcher struct {
	routingAPIClient          routing_api.Client
	updater                   routing_table.Updater
	tokenFetcher              token_fetcher.TokenFetcher
	subscriptionRetryInterval int
	syncChannel               chan struct{}
	logger                    lager.Logger
}

func New(
	routingAPIClient routing_api.Client,
	updater routing_table.Updater,
	tokenFetcher token_fetcher.TokenFetcher,
	subscriptionRetryInterval int,
	syncChannel chan struct{},
	logger lager.Logger,
) *Watcher {
	return &Watcher{
		routingAPIClient:          routingAPIClient,
		updater:                   updater,
		tokenFetcher:              tokenFetcher,
		subscriptionRetryInterval: subscriptionRetryInterval,
		syncChannel:               syncChannel,
		logger:                    logger.Session("watcher"),
	}
}

func (watcher *Watcher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	watcher.logger.Debug("starting")

	close(ready)
	watcher.logger.Debug("started")
	defer watcher.logger.Debug("finished")

	eventChan := make(chan routing_api.TcpEvent)

	var eventSource atomic.Value
	var stopEventSource int32
	canUseCachedToken := true
	go func() {
		var es routing_api.TcpEventSource

		for {
			if atomic.LoadInt32(&stopEventSource) == 1 {
				return
			}
			token, err := watcher.tokenFetcher.FetchToken(canUseCachedToken)
			if err != nil {
				watcher.logger.Error("error-fetching-token", err)
				time.Sleep(time.Duration(watcher.subscriptionRetryInterval) * time.Second)
				continue
			}
			watcher.routingAPIClient.SetToken(token.AccessToken)

			watcher.logger.Info("subscribing-to-tcp-routing-events")
			es, err = watcher.routingAPIClient.SubscribeToTcpEvents()
			if err != nil {
				if err.Error() == "unauthorized" {
					watcher.logger.Error("invalid-oauth-token", err)
					canUseCachedToken = false
				} else {
					canUseCachedToken = true
				}
				watcher.logger.Error("failed-subscribing-to-tcp-routing-events", err)
				time.Sleep(time.Duration(watcher.subscriptionRetryInterval) * time.Second)
				continue
			} else {
				canUseCachedToken = true
			}
			watcher.logger.Info("subscribed-to-tcp-routing-events")

			eventSource.Store(es)

			var event routing_api.TcpEvent
			for {
				event, err = es.Next()
				if err != nil {
					watcher.logger.Error("failed-getting-next-tcp-routing-event", err)
					break
				}
				eventChan <- event
			}
		}
	}()

	for {
		select {
		case event := <-eventChan:
			watcher.updater.HandleEvent(event)

		case <-watcher.syncChannel:
			watcher.updater.Sync()

		case <-signals:
			watcher.logger.Info("stopping")
			atomic.StoreInt32(&stopEventSource, 1)
			if es := eventSource.Load(); es != nil {
				err := es.(routing_api.TcpEventSource).Close()
				if err != nil {
					watcher.logger.Error("failed-closing-event-source", err)
				}
			}
			return nil
		}
	}
}
