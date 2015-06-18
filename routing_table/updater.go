package routing_table

import (
	"errors"
	"sync"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_updater.go . Updater
type Updater interface {
	Update(mappingRequests cf_tcp_router.MappingRequests) error
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
