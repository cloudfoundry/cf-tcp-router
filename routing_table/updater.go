package routing_table

import (
	"errors"
	"sync"

	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_updater.go . Updater
type Updater interface {
	HandleEvent(event routing_api.TcpEvent) error
}

type updater struct {
	logger       lager.Logger
	routingTable models.RoutingTable
	configurer   configurer.RouterConfigurer
	lock         *sync.Mutex
}

func NewUpdater(logger lager.Logger, routingTable models.RoutingTable, configurer configurer.RouterConfigurer) Updater {
	return &updater{
		logger:       logger,
		routingTable: routingTable,
		configurer:   configurer,
		lock:         new(sync.Mutex),
	}
}

func (u *updater) HandleEvent(event routing_api.TcpEvent) error {
	logger := u.logger.Session("handle-event", lager.Data{"event": event})
	action := event.Action
	switch action {
	case "Upsert":
		return u.HandleUpsert(logger, event.TcpRouteMapping)
	case "Delete":
		return u.HandleDelete(logger, event.TcpRouteMapping)
	default:
		logger.Info("unknown-event-action")
		return errors.New("unknown-event-action:" + action)
	}
	return nil
}

func (u *updater) HandleUpsert(logger lager.Logger, routeMapping db.TcpRouteMapping) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	routingKey := models.RoutingKey{Port: routeMapping.TcpRoute.ExternalPort}
	backendServerInfo := models.BackendServerInfo{
		Address: routeMapping.HostIP,
		Port:    routeMapping.HostPort,
	}
	logger.Debug("creating-routing-table-entry", lager.Data{"key": routingKey})
	if u.routingTable.UpsertBackendServerInfo(routingKey, backendServerInfo) {
		logger.Debug("calling-configurer")
		return u.configurer.Configure(u.routingTable)
	}

	return nil
}

func (u *updater) HandleDelete(logger lager.Logger, routeMapping db.TcpRouteMapping) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	routingKey := models.RoutingKey{Port: routeMapping.TcpRoute.ExternalPort}
	backendServerInfo := models.BackendServerInfo{
		Address: routeMapping.HostIP,
		Port:    routeMapping.HostPort,
	}
	logger.Debug("deleting-routing-table-entry", lager.Data{"key": routingKey})
	if u.routingTable.DeleteBackendServerInfo(routingKey, backendServerInfo) {
		logger.Debug("calling-configurer")
		return u.configurer.Configure(u.routingTable)
	}

	return nil
}
