package haproxy

import (
	"bytes"
	"fmt"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
)

type BackendServerInfo struct {
	Name    string
	Address string
	Port    uint16
}

type ListenConfigurationInfo struct {
	Name           string
	FrontendPort   uint16
	BackendServers []BackendServerInfo
}

func NewBackendServerInfo(name, address string, port uint16) BackendServerInfo {
	return BackendServerInfo{
		Name:    name,
		Address: address,
		Port:    port,
	}
}

func NewListenConfigurationInfo(name string, port uint16, backendServers []BackendServerInfo) ListenConfigurationInfo {
	return ListenConfigurationInfo{
		Name:           name,
		FrontendPort:   port,
		BackendServers: backendServers,
	}
}

func (bs BackendServerInfo) ToHaProxyConfig() (string, error) {
	if bs.Name == "" {
		return "", cf_tcp_router.ErrInvalidField{"backend_server.name"}
	}
	if bs.Address == "" {
		return "", cf_tcp_router.ErrInvalidField{"backend_server.address"}
	}
	if bs.Port == 0 {
		return "", cf_tcp_router.ErrInvalidField{"backend_server.port"}
	}
	return fmt.Sprintf("server %s %s:%d\n", bs.Name, bs.Address, bs.Port), nil
}

func (lc ListenConfigurationInfo) ToHaProxyConfig() (string, error) {
	if lc.Name == "" {
		return "", cf_tcp_router.ErrInvalidField{"listen_configuration.name"}
	}
	if lc.FrontendPort == 0 {
		return "", cf_tcp_router.ErrInvalidField{"listen_configuration.port"}
	}
	var buff bytes.Buffer

	buff.WriteString(fmt.Sprintf("listen %s\n  mode tcp\n  bind :%d\n", lc.Name, lc.FrontendPort))
	for _, bs := range lc.BackendServers {
		str, err := bs.ToHaProxyConfig()
		if err != nil {
			return "", err
		}
		buff.WriteString(fmt.Sprintf("  %s", str))
	}
	return buff.String(), nil
}
