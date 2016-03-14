package routing_table

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/routing-api"
	apimodels "github.com/cloudfoundry-incubator/routing-api/models"
	uaaclient "github.com/cloudfoundry-incubator/uaa-go-client"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_updater.go . Updater
type Updater interface {
	HandleEvent(event routing_api.TcpEvent) error
	Sync()
	Syncing() bool
}

type updater struct {
	logger           lager.Logger
	routingTable     *models.RoutingTable
	configurer       configurer.RouterConfigurer
	syncing          bool
	routingAPIClient routing_api.Client
	uaaClient        uaaclient.Client
	cachedEvents     []routing_api.TcpEvent
	lock             *sync.Mutex
}

func NewUpdater(logger lager.Logger, routingTable *models.RoutingTable, configurer configurer.RouterConfigurer,
	routingAPIClient routing_api.Client, uaaClient uaaclient.Client) Updater {
	return &updater{
		logger:           logger,
		routingTable:     routingTable,
		configurer:       configurer,
		lock:             new(sync.Mutex),
		syncing:          false,
		routingAPIClient: routingAPIClient,
		uaaClient:        uaaClient,
		cachedEvents:     nil,
	}
}

func (u *updater) Sync() {
	logger := u.logger.Session("handle-sync")
	logger.Debug("starting")

	defer func() {
		u.lock.Lock()
		u.applyCachedEvents(logger)
		u.configurer.Configure(*u.routingTable)
		logger.Debug("applied-fetched-routes-to-routing-table", lager.Data{"size": u.routingTable.Size()})
		u.syncing = false
		u.cachedEvents = nil
		u.lock.Unlock()
		logger.Debug("complete")
	}()

	u.lock.Lock()
	u.syncing = true
	u.cachedEvents = []routing_api.TcpEvent{}
	u.lock.Unlock()

	useCachedToken := true
	var err error
	var tcpRouteMappings []apimodels.TcpRouteMapping
	for count := 0; count < 2; count++ {
		token, tokenErr := u.uaaClient.FetchToken(useCachedToken)
		if tokenErr != nil {
			logger.Error("error-fetching-token", tokenErr)
			return
		}
		u.routingAPIClient.SetToken(token.AccessToken)
		tcpRouteMappings, err = u.routingAPIClient.TcpRouteMappings()
		if err != nil {
			logger.Error("error-fetching-routes", err)
			if err.Error() == "unauthorized" {
				useCachedToken = false
				logger.Info("retrying-sync")
			} else {
				return
			}
		} else {
			break
		}
	}
	logger.Debug("fetched-tcp-routes", lager.Data{"num-routes": len(tcpRouteMappings)})
	if err == nil {
		// Create a new map and populate using tcp route mappings we got from routing api
		u.routingTable.Entries = make(map[models.RoutingKey]models.RoutingTableEntry)
		for _, routeMapping := range tcpRouteMappings {
			routingKey, backendServerInfo := u.toRoutingTableEntry(logger, routeMapping)
			logger.Debug("creating-routing-table-entry", lager.Data{"key": routingKey, "value": backendServerInfo})
			u.routingTable.UpsertBackendServerInfo(routingKey, backendServerInfo)
		}
	}
}

func (u *updater) applyCachedEvents(logger lager.Logger) {
	logger.Debug("applying-cached-events")
	defer logger.Debug("applied-cached-events")
	for _, e := range u.cachedEvents {
		u.handleEvent(e)
	}
}

func (u *updater) Syncing() bool {
	u.lock.Lock()
	defer u.lock.Unlock()
	return u.syncing
}

func (u *updater) HandleEvent(event routing_api.TcpEvent) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	if u.syncing {
		u.logger.Debug("caching-events")
		u.cachedEvents = append(u.cachedEvents, event)
	} else {
		return u.handleEvent(event)
	}
	return nil
}

func (u *updater) handleEvent(event routing_api.TcpEvent) error {
	logger := u.logger.Session("handle-event", lager.Data{"event": event})
	action := event.Action
	switch action {
	case "Upsert":
		return u.handleUpsert(logger, event.TcpRouteMapping)
	case "Delete":
		return u.handleDelete(logger, event.TcpRouteMapping)
	default:
		logger.Info("unknown-event-action")
		return errors.New("unknown-event-action:" + action)
	}
}

func (u *updater) toRoutingTableEntry(logger lager.Logger, routeMapping apimodels.TcpRouteMapping) (models.RoutingKey, models.BackendServerInfo) {
	logger.Debug("converting-tcp-route-mapping", lager.Data{"tcp-route": routeMapping})
	routingKey := models.RoutingKey{Port: routeMapping.TcpRoute.ExternalPort}
	backendServerInfo := models.BackendServerInfo{
		Address: routeMapping.HostIP,
		Port:    routeMapping.HostPort,
	}
	return routingKey, backendServerInfo
}

func (u *updater) handleUpsert(logger lager.Logger, routeMapping apimodels.TcpRouteMapping) error {
	defer logger.Debug("handle-upsert-done")
	routingKey, backendServerInfo := u.toRoutingTableEntry(logger, routeMapping)
	logger.Debug("creating-routing-table-entry", lager.Data{"key": routingKey})
	if u.routingTable.UpsertBackendServerInfo(routingKey, backendServerInfo) && !u.syncing {
		logger.Debug("calling-configurer")
		return u.configurer.Configure(*u.routingTable)
	}

	return nil
}

func (u *updater) handleDelete(logger lager.Logger, routeMapping apimodels.TcpRouteMapping) error {
	defer logger.Debug("handle-delete-done")
	routingKey, backendServerInfo := u.toRoutingTableEntry(logger, routeMapping)
	logger.Debug("deleting-routing-table-entry", lager.Data{"key": routingKey})
	if u.routingTable.DeleteBackendServerInfo(routingKey, backendServerInfo) && !u.syncing {
		logger.Debug("calling-configurer")
		return u.configurer.Configure(*u.routingTable)
	}

	return nil
}
