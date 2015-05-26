package haproxy

import (
	"math/rand"
	"net"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/pivotal-golang/lager"
)

type HaProxyConfigurer struct {
	logger lager.Logger
}

func NewHaProxyConfigurer(logger lager.Logger) HaProxyConfigurer {
	return HaProxyConfigurer{
		logger: logger,
	}
}

func (h HaProxyConfigurer) MapBackendHostsToAvailablePort(backendHostInfos cf_tcp_router.BackendHostInfos) (cf_tcp_router.RouterHostInfo, error) {
	err := backendHostInfos.Validate()
	if err != nil {
		h.logger.Error("invalid-backendhostinfo", err)
		return cf_tcp_router.RouterHostInfo{}, err
	}
	externalIP, err := h.getExternalIP()
	if err != nil {
		return cf_tcp_router.RouterHostInfo{}, err
	}
	// This is dummy implementation and needs to be changed once we integrate with haproxy
	return cf_tcp_router.NewRouterHostInfo(externalIP, h.getExternalPort()), nil
}

// This is dummy implementation and will change once we integrate with haproxy
func (h HaProxyConfigurer) getExternalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		h.logger.Error("error-getting-interfaces", err)
		return "", err
	}
	var externalIP string
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ipv4 := ip.To4(); ipv4 != nil {
			if ipv4.IsLoopback() == false {
				externalIP = ipv4.String()
				break
			}
		}
	}
	return externalIP, nil
}

// This is dummy implementation and will change once we integrate with haproxy
func (h HaProxyConfigurer) getExternalPort() uint16 {
	randomPort := rand.Int31n(65536)
	return uint16(randomPort)
}
