package watcher

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/routing_table"
	"code.cloudfoundry.org/lager/v3"
	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/uaaclient"
	"github.com/tedsuo/ifrit"
)

type Watcher struct {
	routingAPIClient          routing_api.Client
	updater                   routing_table.Updater
	uaaTokenFetcher           uaaclient.TokenFetcher
	subscriptionRetryInterval int
	syncChannel               chan struct{}
	logger                    lager.Logger
	process                   ifrit.Process
}

func New(
	routingAPIClient routing_api.Client,
	updater routing_table.Updater,
	uaaTokenFetcher uaaclient.TokenFetcher,
	subscriptionRetryInterval int,
	syncChannel chan struct{},
	logger lager.Logger,
) *Watcher {
	return &Watcher{
		routingAPIClient:          routingAPIClient,
		updater:                   updater,
		uaaTokenFetcher:           uaaTokenFetcher,
		subscriptionRetryInterval: subscriptionRetryInterval,
		syncChannel:               syncChannel,
		logger:                    logger.Session("watcher"),
	}
}

func (watcher *Watcher) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	watcher.logger.Debug("starting")
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
			token, err := watcher.uaaTokenFetcher.FetchToken(context.Background(), !canUseCachedToken)
			if err != nil {
				watcher.logger.Error("error-fetching-token", err)
				time.Sleep(time.Duration(watcher.subscriptionRetryInterval) * time.Second)
				continue
			}
			watcher.routingAPIClient.SetToken(token.AccessToken)

			watcher.logger.Info("Subscribing-to-routing-api-event-stream")
			es, err = watcher.routingAPIClient.SubscribeToTcpEvents()
			if err != nil {
				if err.Error() == "unauthorized" {
					watcher.logger.Error("invalid-oauth-token", err)
					canUseCachedToken = false
				} else {
					canUseCachedToken = true
				}
				watcher.logger.Error("failed-subscribing-to-routing-api-event-stream", err)
				time.Sleep(time.Duration(watcher.subscriptionRetryInterval) * time.Second)
				continue
			} else {
				canUseCachedToken = true
			}
			watcher.logger.Info("Successfully-subscribed-to-routing-api-event-stream")

			eventSource.Store(es)

			var event routing_api.TcpEvent
			for {
				event, err = es.Next()
				if err != nil {
					watcher.logger.Error("failed-to-get-next-routing-api-event", err)
					err = es.Close()
					if err != nil {
						watcher.logger.Error("failed-closing-routing-api-event-source", err)
					}
					break
				}
				eventChan <- event
			}
		}
	}()

	close(ready)
	watcher.logger.Debug("started")

	for {
		select {
		case event := <-eventChan:
			watcher.updater.HandleEvent(event)

		case <-watcher.syncChannel:
			go watcher.updater.Sync()

		case sig := <-signals:
			if sig == syscall.SIGUSR2 {
				go func() {
					watcher.logger.Info("drain-requested")
					err := watcher.updater.Drain()
					if err != nil {
						watcher.logger.Error("failed-draining", err)
					}
					if watcher.process != nil {
						watcher.process.Signal(os.Interrupt)
					}
				}()
			} else {
				watcher.logger.Info("stopping")
				atomic.StoreInt32(&stopEventSource, 1)
				if es := eventSource.Load(); es != nil {
					err := es.(routing_api.TcpEventSource).Close()
					if err != nil {
						watcher.logger.Error("failed-closing-routing-api-event-source", err)
					}
				}
				return nil
			}

		}
	}
}

func (watcher *Watcher) SetProcess(proc ifrit.Process) {
	watcher.process = proc
}
