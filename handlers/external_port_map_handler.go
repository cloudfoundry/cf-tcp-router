package handlers

import (
	"encoding/json"
	"net/http"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"
	"github.com/pivotal-golang/lager"
)

type ExternalPortMapHandler struct {
	updater routing_table.Updater
	logger  lager.Logger
}

func NewExternalPortMapHandler(logger lager.Logger, updater routing_table.Updater) *ExternalPortMapHandler {
	return &ExternalPortMapHandler{
		logger:  logger,
		updater: updater,
	}
}

func (h *ExternalPortMapHandler) MapExternalPort(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("map_external_port")
	logger.Info("map-external-port")

	var mappingRequest cf_tcp_router.MappingRequests
	err := json.NewDecoder(r.Body).Decode(&mappingRequest)
	if err != nil {
		logger.Error("failed-to-unmarshal", err)
		writeInvalidJSONResponse(w, err)
		return
	}

	err = h.updater.Update(mappingRequest)
	if err != nil {
		if err.Error() == cf_tcp_router.ErrInvalidMapingRequest {
			logger.Error("invalid-payload", err)
			writeInvalidJSONResponse(w, err)
		} else {
			logger.Error("failed-to-update", err)
			writeInternalErrorJSONResponse(w, err)
		}
		return
	}

	writeStatusOKResponse(w, nil)
}
