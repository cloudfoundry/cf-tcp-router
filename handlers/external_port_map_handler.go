package handlers

import (
	"encoding/json"
	"net/http"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/GESoftware-CF/cf-tcp-router/configurer"
	"github.com/pivotal-golang/lager"
)

type ExternalPortMapHandler struct {
	configurer configurer.RouterConfigurer
	logger     lager.Logger
}

func NewExternalPortMapHandler(logger lager.Logger, configurer configurer.RouterConfigurer) *ExternalPortMapHandler {
	return &ExternalPortMapHandler{
		logger:     logger,
		configurer: configurer,
	}
}

func (h *ExternalPortMapHandler) MapExternalPort(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("map_external_port")
	logger.Info("map-external-port")

	var backendHostInfos cf_tcp_router.BackendHostInfos
	err := json.NewDecoder(r.Body).Decode(&backendHostInfos)
	if err != nil {
		logger.Error("failed-to-unmarshal", err)
		writeInvalidJSONResponse(w, err)
		return
	}

	routerHostInfo, err := h.configurer.MapBackendHostsToAvailablePort(backendHostInfos)
	if err != nil {
		if err.Error() == cf_tcp_router.ErrInvalidBackendHostInfo {
			logger.Error("invalid-payload", err)
			writeInvalidJSONResponse(w, err)
		} else {
			logger.Error("failed-to-configure", err)
			writeInternalErrorJSONResponse(w, err)
		}
		return
	}

	writeStatusCreatedResponse(w, routerHostInfo)
}
