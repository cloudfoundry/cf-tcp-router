package handlers

import (
	"net/http"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/rata"
)

func New(logger lager.Logger, configurer configurer.RouterConfigurer) http.Handler {
	externalPortMapHandler := NewExternalPortMapHandler(logger, configurer)
	actions := rata.Handlers{
		// External port mapping
		cf_tcp_router.MapExternalPortRoute: route(externalPortMapHandler.MapExternalPort),
	}

	handler, err := rata.NewRouter(cf_tcp_router.Routes, actions)
	if err != nil {
		panic("unable to create router: " + err.Error())
	}

	return handler
}

func route(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(f)
}
