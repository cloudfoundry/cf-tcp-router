package haproxy

import (
	"fmt"
	"sort"
	"strings"

	"code.cloudfoundry.org/cf-tcp-router/config"
	"code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/lager/v3"
)

//go:generate counterfeiter -o fakes/fake_config_marshaller.go . ConfigMarshaller
type ConfigMarshaller interface {
	Marshal(models.HAProxyConfig, config.BackendTLSConfig) string
}

type configMarshaller struct {
	logger lager.Logger
}

func NewConfigMarshaller(l lager.Logger) ConfigMarshaller {
	return configMarshaller{logger: l}
}

func (cm configMarshaller) Marshal(conf models.HAProxyConfig, backendTlsCfg config.BackendTLSConfig) string {
	var output strings.Builder
	sortedPorts := sortedHAProxyInboundPorts(conf)
	for inboundPortIdx := range sortedPorts {
		port := sortedPorts[inboundPortIdx]
		frontend := conf[port]

		output.WriteString(cm.marshalHAProxyFrontend(port, frontend, backendTlsCfg))
	}
	return output.String()
}

func (cm configMarshaller) marshalHAProxyFrontend(port models.HAProxyInboundPort, frontend models.HAProxyFrontend, backendTlsCfg config.BackendTLSConfig) string {
	var (
		frontendStanza strings.Builder
		backendStanzas strings.Builder
	)
	frontendStanza.WriteString(fmt.Sprintf("\nfrontend frontend_%d", port))
	frontendStanza.WriteString("\n  mode tcp")
	frontendStanza.WriteString(fmt.Sprintf("\n  bind :%d", port))

	if frontend.ContainsSNIRoutes() {
		frontendStanza.WriteString("\n  tcp-request inspect-delay 5s")
		frontendStanza.WriteString("\n  tcp-request content accept if { req.ssl_hello_type gt 0 }")
	}

	sortedHostnames := sortedSniHostnames(frontend)
	for hostnameIdx := range sortedHostnames {
		var backendCfgName string
		hostname := sortedHostnames[hostnameIdx]

		if hostname == "" { // The non-SNI route gets a default_backend because none of the `use_backend if {...}` predicates will succeed
			backendCfgName = fmt.Sprintf("backend_%d", port)
			frontendStanza.WriteString(fmt.Sprintf("\n  default_backend %s", backendCfgName))

		} else { // SNI routes use named backends
			backendCfgName = fmt.Sprintf("backend_%d_%s", port, hostname)
			frontendStanza.WriteString(fmt.Sprintf("\n  use_backend %s if { req.ssl_sni %s }", backendCfgName, hostname))
		}

		backend := frontend[hostname]

		haProxyBackendString := cm.marshalHAProxyBackend(backendCfgName, backend, backendTlsCfg)
		backendStanzas.WriteString(haProxyBackendString)
	}

	frontendStanza.WriteString("\n")
	frontendStanza.WriteString(backendStanzas.String())
	return frontendStanza.String()
}

// This might result in malformed lines since we always write the opening stanza, but conditionally write others...
func (cm configMarshaller) marshalHAProxyBackend(backendName string, backend models.HAProxyBackend, backendTlsCfg config.BackendTLSConfig) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("\nbackend %s", backendName))
	output.WriteString("\n  mode tcp")

	for _, server := range backend {
		if server.TLSPort > 0 && !backendTlsCfg.Enabled {
			cm.logger.Error("backend-tls-not-enabled", fmt.Errorf("Backend TLS Port was set, but backend_tls has not been enabled for tcp-router"), lager.Data{"backend": backend})
			//skip this endpoint, but there may be other backends with tlsport <= 0 that we should still set
			continue
		}

		if server.TLSPort > 0 {
			output.WriteString(fmt.Sprintf("\n  server server_%s_%d %s:%d ssl verify required verifyhost %s ca-file %s", server.Address, server.TLSPort, server.Address, server.TLSPort, server.InstanceID, backendTlsCfg.CACertificatePath))

			if backendTlsCfg.ClientCertAndKeyPath != "" {
				output.WriteString(fmt.Sprintf(" crt %s", backendTlsCfg.ClientCertAndKeyPath))
			}
		} else {
			if server.TLSPort == 0 && backendTlsCfg.Enabled {
				cm.logger.Error("route-missing-tls-information", fmt.Errorf("Backend TLSPort was set to 0. If TLS is intentionally off for this backend, set this to -1 to suppress this message"), lager.Data{"backend": server})
			}
			output.WriteString(fmt.Sprintf("\n  server server_%s_%d %s:%d", server.Address, server.Port, server.Address, server.Port))
		}
	}

	output.WriteString("\n")
	return output.String()
}

func sortedHAProxyInboundPorts(conf models.HAProxyConfig) []models.HAProxyInboundPort {
	keys := make([]models.HAProxyInboundPort, len(conf))
	i := 0
	for k := range conf {
		keys[i] = k
		i++
	}

	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func sortedSniHostnames(frontend models.HAProxyFrontend) []models.SniHostname {
	keys := make([]models.SniHostname, len(frontend))
	i := 0
	for k := range frontend {
		keys[i] = k
		i++
	}

	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
