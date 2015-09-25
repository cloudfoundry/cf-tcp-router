package routing_table

import (
	"errors"
	"sync"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_updater.go . Updater
type Updater interface {
	Update(mappingRequests cf_tcp_router.MappingRequests) error
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

func (u *updater) Update(mappingRequests cf_tcp_router.MappingRequests) error {
	logger := u.logger.WithData(lager.Data{"request": mappingRequests})
	logger.Debug("start-update")
	defer logger.Debug("end-update")
	err := mappingRequests.Validate()
	if err != nil {
		logger.Error("invalid-mapping-request", err)
		return errors.New(cf_tcp_router.ErrInvalidMapingRequest)
	}
	u.lock.Lock()
	defer u.lock.Unlock()

	externalPortMap := make(map[uint16]cf_tcp_router.MappingRequest)

	for _, mappingRequest := range mappingRequests {
		if existingMappingRequest, ok := externalPortMap[mappingRequest.ExternalPort]; ok {
			mappingRequest.Backends = append(mappingRequest.Backends, existingMappingRequest.Backends...)
		}
		externalPortMap[mappingRequest.ExternalPort] = mappingRequest
	}

	changed := false
	for _, mappingRequest := range externalPortMap {
		routingKey, routingTableEntry := models.ToRoutingTableEntry(mappingRequest)
		logger.Debug("creating-routing-table-entry", lager.Data{"key": routingKey})
		if u.routingTable.Set(routingKey, routingTableEntry) {
			changed = true
		}
	}
	if changed {
		logger.Debug("calling-configurer")
		return u.configurer.Configure(u.routingTable)
	}
	return nil
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
