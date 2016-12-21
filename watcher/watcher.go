package watcher

import (
	"os"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/routing_table"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/routing-api"
	uaaclient "code.cloudfoundry.org/uaa-go-client"
)

type Watcher struct {
	routingAPIClient          routing_api.Client
	updater                   routing_table.Updater
	uaaClient                 uaaclient.Client
	subscriptionRetryInterval int
	syncChannel               chan struct{}
	eventChan                 chan routing_api.TcpEvent
	stopEventSource           chan struct{}
	eventSource               atomic.Value
	logger                    lager.Logger
}

func New(
	routingAPIClient routing_api.Client,
	updater routing_table.Updater,
	uaaClient uaaclient.Client,
	subscriptionRetryInterval int,
	syncChannel chan struct{},
	logger lager.Logger,
) *Watcher {
	return &Watcher{
		routingAPIClient:          routingAPIClient,
		updater:                   updater,
		uaaClient:                 uaaClient,
		subscriptionRetryInterval: subscriptionRetryInterval,
		syncChannel:               syncChannel,
		eventChan:                 make(chan routing_api.TcpEvent),
		stopEventSource:           make(chan struct{}),
		logger:                    logger.Session("watcher"),
	}
}

func (w *Watcher) Run(signal <-chan os.Signal, ready chan<- struct{}) error {
	w.logger.Debug("starting")
	defer w.logger.Debug("finished")

	go w.subscribe()
	close(ready)
	w.logger.Debug("started")
	w.handleEvent(signal)
	return nil
}

func (w *Watcher) subscribe() {
	canUseCachedToken := true
	for {
		select {
		case <-w.stopEventSource:
			return
		default:
			token, err := w.uaaClient.FetchToken(!canUseCachedToken)
			if err != nil {
				w.logger.Error("error-fetching-token", err)
				time.Sleep(time.Duration(w.subscriptionRetryInterval) * time.Second)
				continue
			}
			w.routingAPIClient.SetToken(token.AccessToken)

			w.logger.Info("Subscribing-to-routing-api-event-stream")
			es, err := w.routingAPIClient.SubscribeToTcpEvents()
			if err != nil {
				if err.Error() == "unauthorized" {
					w.logger.Error("invalid-oauth-token", err)
					canUseCachedToken = false
				} else {
					canUseCachedToken = true
				}
				w.logger.Error("failed-subscribing-to-routing-api-event-stream", err)
				time.Sleep(time.Duration(w.subscriptionRetryInterval) * time.Second)
				continue
			} else {
				canUseCachedToken = true
			}
			w.logger.Info("Successfully-subscribed-to-routing-api-event-stream")

			w.eventSource.Store(es)
			w.nextEvent(es)
		}
	}
}

func (w *Watcher) handleEvent(signal <-chan os.Signal) {
	for {
		select {
		case event := <-w.eventChan:
			w.updater.HandleEvent(event)

		case <-w.syncChannel:
			go w.updater.Sync()

		case <-signal:
			w.logger.Info("stopping")
			close(w.stopEventSource)
			if es := w.eventSource.Load(); es != nil {
				err := es.(routing_api.TcpEventSource).Close()
				if err != nil {
					w.logger.Error("failed-closing-routing-api-event-source", err)
				}
			}
		}
	}
}

func (w *Watcher) nextEvent(es routing_api.TcpEventSource) {
	for {
		event, err := es.Next()
		if err != nil {
			w.logger.Error("failed-to-get-next-routing-api-event", err)
			err = es.Close()
			if err != nil {
				w.logger.Error("failed-closing-routing-api-event-source", err)
			}
			break
		}
		w.eventChan <- event
	}
}
