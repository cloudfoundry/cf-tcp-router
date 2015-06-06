package haproxy

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync"

	"github.com/GESoftware-CF/cf-tcp-router/utils"

	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"
	"github.com/pivotal-golang/lager"
)

type HaProxyConfigurer struct {
	logger           lager.Logger
	frontendAddress  string
	nextFrontendPort uint16
	configFilePath   string
	portLock         *sync.Mutex
	configFileLock   *sync.Mutex
}

func NewHaProxyConfigurer(logger lager.Logger, configFilePath string, configStartFrontendPort uint16) (*HaProxyConfigurer, error) {
	ip, err := getExternalIP()
	if err != nil {
		return nil, err
	}

	if !utils.FileExists(configFilePath) {
		return nil, errors.New(fmt.Sprintf("%s: [%s]", cf_tcp_router.ErrRouterConfigFileNotFound, configFilePath))
	}

	if configStartFrontendPort == 0 || configStartFrontendPort < cf_tcp_router.LowerBoundStartFrontendPort {
		return nil, errors.New(fmt.Sprintf("%s: [%d]", cf_tcp_router.ErrInvalidStartFrontendPort, configStartFrontendPort))
	}
	return &HaProxyConfigurer{
		logger:           logger,
		frontendAddress:  ip,
		nextFrontendPort: configStartFrontendPort,
		configFilePath:   configFilePath,
		portLock:         new(sync.Mutex),
		configFileLock:   new(sync.Mutex),
	}, nil
}

func (h *HaProxyConfigurer) MapBackendHostsToAvailablePort(backendHostInfos cf_tcp_router.BackendHostInfos) (cf_tcp_router.RouterHostInfo, error) {
	err := backendHostInfos.Validate()
	if err != nil {
		h.logger.Error("invalid-backendhostinfo", err)
		return cf_tcp_router.RouterHostInfo{}, errors.New(cf_tcp_router.ErrInvalidBackendHostInfo)
	}
	frontendPort := h.getFrontendPort()

	listenConfiguration := h.getListenConfiguration(backendHostInfos, frontendPort)

	err = h.applyListenConfiguration(listenConfiguration)
	if err != nil {
		h.logger.Error("failed-applying-configuration", err)
		return cf_tcp_router.RouterHostInfo{}, nil
	}

	// This is dummy implementation and needs to be changed once we integrate with haproxy
	return cf_tcp_router.NewRouterHostInfo(h.frontendAddress, frontendPort), nil
}

func (h *HaProxyConfigurer) getListenConfiguration(
	backendHostInfos cf_tcp_router.BackendHostInfos,
	frontendPort uint16) ListenConfigurationInfo {
	backendServers := make([]BackendServerInfo, len(backendHostInfos))
	for i, backendHost := range backendHostInfos {
		bs := NewBackendServerInfo(
			fmt.Sprintf("server_%s_%d", backendHost.Address, i),
			backendHost.Address,
			backendHost.Port)
		backendServers[i] = bs
	}
	return NewListenConfigurationInfo(fmt.Sprintf("listen_cfg_%d", frontendPort), frontendPort, backendServers)
}

func (h *HaProxyConfigurer) appendListenConfiguration(listenCfg ListenConfigurationInfo, cfgContent []byte) ([]byte, error) {
	var buff bytes.Buffer
	_, err := buff.Write(cfgContent)
	if err != nil {
		h.logger.Error("failed-copying-config-file", err, lager.Data{"config-file": h.configFilePath})
		return nil, err
	}

	_, err = buff.WriteString("\n")
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}

	var listenCfgStr string
	listenCfgStr, err = listenCfg.ToHaProxyConfig()
	if err != nil {
		h.logger.Error("failed-marshaling-listen-config", err)
		return nil, err
	}

	_, err = buff.WriteString(listenCfgStr)
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}
	return buff.Bytes(), nil
}

func (h *HaProxyConfigurer) applyListenConfiguration(listenCfg ListenConfigurationInfo) error {
	h.configFileLock.Lock()
	defer h.configFileLock.Unlock()

	h.logger.Debug("reading-config-file", lager.Data{"config-file": h.configFilePath})
	cfgContent, err := ioutil.ReadFile(h.configFilePath)
	if err != nil {
		h.logger.Error("failed-reading-config-file", err, lager.Data{"config-file": h.configFilePath})
		return err
	}

	backupConfigFileName := fmt.Sprintf("%s.bak", h.configFilePath)
	err = utils.WriteToFile(cfgContent, backupConfigFileName)
	if err != nil {
		h.logger.Error("failed-to-backup-config", err, lager.Data{"config-file": h.configFilePath})
		return err
	}

	newCfgContent, err := h.appendListenConfiguration(listenCfg, cfgContent)
	if err != nil {
		return err
	}

	tmpConfigFileName := fmt.Sprintf("%s.tmp", h.configFilePath)
	err = utils.WriteToFile(newCfgContent, tmpConfigFileName)
	if err != nil {
		h.logger.Error("failed-to-write-temp-config", err, lager.Data{"temp-config-file": tmpConfigFileName})
		return err
	}

	err = os.Rename(tmpConfigFileName, h.configFilePath)
	if err != nil {
		h.logger.Error(
			"failed-renaming-temp-config-file",
			err,
			lager.Data{"config-file": h.configFilePath, "temp-config-file": tmpConfigFileName})
		return err
	}
	return nil
}

// This is dummy implementation and will change once we integrate with haproxy
func getExternalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
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

// This will change once routing table is implemented
func (h *HaProxyConfigurer) getFrontendPort() uint16 {
	h.portLock.Lock()
	defer h.portLock.Unlock()
	port := h.nextFrontendPort
	h.nextFrontendPort++
	return port
}
