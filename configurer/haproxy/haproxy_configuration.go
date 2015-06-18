package haproxy

import (
	"bytes"
	"fmt"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
)

func BackendServerInfoToHaProxyConfig(bs models.BackendServerInfo) (string, error) {
	if bs.Address == "" {
		return "", cf_tcp_router.ErrInvalidField{"backend_server.address"}
	}
	if bs.Port == 0 {
		return "", cf_tcp_router.ErrInvalidField{"backend_server.port"}
	}
	name := fmt.Sprintf("server_%s_%d", bs.Address, bs.Port)
	return fmt.Sprintf("server %s %s:%d\n", name, bs.Address, bs.Port), nil
}

func RoutingTableEntryToHaProxyConfig(routingKey models.RoutingKey, routingTableEntry models.RoutingTableEntry) (string, error) {
	if routingKey.Port == 0 {
		return "", cf_tcp_router.ErrInvalidField{"listen_configuration.port"}
	}
	name := fmt.Sprintf("listen_cfg_%d", routingKey.Port)
	var buff bytes.Buffer

	buff.WriteString(fmt.Sprintf("listen %s\n  mode tcp\n  bind :%d\n", name, routingKey.Port))
	for bs, _ := range routingTableEntry.Backends {
		str, err := BackendServerInfoToHaProxyConfig(bs)
		if err != nil {
			return "", err
		}
		buff.WriteString(fmt.Sprintf("  %s", str))
	}
	return buff.String(), nil
}
