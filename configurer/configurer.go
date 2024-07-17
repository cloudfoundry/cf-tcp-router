package configurer

import (
	"errors"

	"code.cloudfoundry.org/cf-tcp-router/config"
	"code.cloudfoundry.org/cf-tcp-router/configurer/haproxy"
	"code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/cf-tcp-router/monitor"
	"code.cloudfoundry.org/lager/v3"
)

const (
	HaProxyConfigurer = "HAProxy"
)

//go:generate counterfeiter -o fakes/fake_configurer.go . RouterConfigurer
type RouterConfigurer interface {
	Configure(routingTable models.RoutingTable, forceHealthCheckToFail bool) error
}

func NewConfigurer(logger lager.Logger, tcpLoadBalancer string, tcpLoadBalancerBaseCfg string, tcpLoadBalancerCfg string, monitor monitor.Monitor, scriptRunner haproxy.ScriptRunner, backendTlsCfg config.BackendTLSConfig) RouterConfigurer {
	switch tcpLoadBalancer {
	case HaProxyConfigurer:
		routerHostInfo, err := haproxy.NewHaProxyConfigurer(
			logger,
			haproxy.NewConfigMarshaller(logger),
			tcpLoadBalancerBaseCfg,
			tcpLoadBalancerCfg,
			monitor,
			scriptRunner,
			backendTlsCfg,
		)

		if err != nil {
			logger.Fatal("could not create tcp load balancer",
				err,
				lager.Data{"tcp_load_balancer": tcpLoadBalancer})
			return nil
		}
		return routerHostInfo
	default:
		logger.Fatal("not-supported", errors.New("unsupported tcp load balancer"), lager.Data{"tcp_load_balancer": tcpLoadBalancer})
		return nil
	}
}
