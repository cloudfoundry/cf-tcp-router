package configurer

import (
	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
)

type RouterConfigurer interface {
	MapBackendHostsToAvailablePort(backendHostInfos cf_tcp_router.BackendHostInfos) (cf_tcp_router.RouterHostInfo, error)
}
