package haproxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/cf-tcp-router/monitor"
	"code.cloudfoundry.org/cf-tcp-router/utils"
	"code.cloudfoundry.org/lager"
)

const (
	ErrRouterConfigFileNotFound = "Configuration file not found"
	ErrNoChildProcesses         = "waitid: no child processes"
)

type Configurer struct {
	logger             lager.Logger
	baseConfigFilePath string
	configFilePath     string
	configFileLock     *sync.Mutex
	monitor            monitor.Monitor
	scriptRunner       ScriptRunner
}

func NewHaProxyConfigurer(logger lager.Logger, baseConfigFilePath string, configFilePath string, monitor monitor.Monitor, scriptRunner ScriptRunner) (*Configurer, error) {
	if !utils.FileExists(baseConfigFilePath) {
		return nil, fmt.Errorf("%s: [%s]", ErrRouterConfigFileNotFound, baseConfigFilePath)
	}
	if !utils.FileExists(configFilePath) {
		return nil, fmt.Errorf("%s: [%s]", ErrRouterConfigFileNotFound, configFilePath)
	}
	return &Configurer{
		logger:             logger,
		baseConfigFilePath: baseConfigFilePath,
		configFilePath:     configFilePath,
		configFileLock:     new(sync.Mutex),
		monitor:            monitor,
		scriptRunner:       scriptRunner,
	}, nil
}

func (h *Configurer) Configure(routingTable models.RoutingTable) error {
	h.monitor.StopWatching()
	h.configFileLock.Lock()
	defer h.configFileLock.Unlock()

	err := h.createConfigBackup()
	if err != nil {
		return err
	}

	cfgContent, err := ioutil.ReadFile(h.baseConfigFilePath)
	if err != nil {
		h.logger.Error("failed-reading-base-config-file", err, lager.Data{"base-config-file": h.baseConfigFilePath})
		return err
	}
	var buff bytes.Buffer
	_, err = buff.Write(cfgContent)
	if err != nil {
		h.logger.Error("failed-copying-config-file", err, lager.Data{"config-file": h.configFilePath})
		return err
	}

	for key, entry := range routingTable.Entries {
		cfgContent, err = h.getListenConfiguration(key, entry)
		if err != nil {
			continue
		}
		_, err = buff.Write(cfgContent)
		if err != nil {
			h.logger.Error("failed-writing-to-buffer", err)
			return err
		}
	}

	h.logger.Info("writing-config", lager.Data{"num-bytes": buff.Len()})
	err = h.writeToConfig(buff.Bytes())
	if err != nil {
		return err
	}

	if h.scriptRunner != nil {
		h.logger.Info("running-script")

		signalChannel := make(chan os.Signal, 2)
		signal.Notify(signalChannel, syscall.SIGCHLD)
		go func() {
			sig := <-signalChannel
			if sig == syscall.SIGCHLD {
				r := syscall.Rusage{}
				for {
					pid, waitErr := syscall.Wait4(-1, nil, 0, &r)
					pidstring := strconv.Itoa(pid)
					if waitErr != nil {
						h.logger.Debug("wait4-failed", lager.Data{"pid": pidstring, "message": waitErr})
					} else {
						h.logger.Debug("wait4-suceeded", lager.Data{"pid": pidstring})
					}
				}
			}
		}()

		err = h.scriptRunner.Run()
		if err != nil && err.Error() != ErrNoChildProcesses {
			h.logger.Error("failed-to-run-script", err)
			return err
		}
		h.monitor.StartWatching()
	}
	return nil
}

func (h *Configurer) getListenConfiguration(key models.RoutingKey, entry models.RoutingTableEntry) ([]byte, error) {
	var buff bytes.Buffer
	_, err := buff.WriteString("\n")
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}

	var listenCfgStr string
	listenCfgStr, err = RoutingTableEntryToHaProxyConfig(key, entry)
	if err != nil {
		h.logger.Error("failed-marshaling-routing-table-entry", err)
		return nil, err
	}

	_, err = buff.WriteString(listenCfgStr)
	if err != nil {
		h.logger.Error("failed-writing-to-buffer", err)
		return nil, err
	}
	return buff.Bytes(), nil
}

func (h *Configurer) createConfigBackup() error {
	h.logger.Debug("reading-config-file", lager.Data{"config-file": h.configFilePath})
	cfgContent, err := ioutil.ReadFile(h.configFilePath)
	if err != nil {
		h.logger.Error("failed-reading-base-config-file", err, lager.Data{"config-file": h.configFilePath})
		return err
	}
	backupConfigFileName := fmt.Sprintf("%s.bak", h.configFilePath)
	err = utils.WriteToFile(cfgContent, backupConfigFileName)
	if err != nil {
		h.logger.Error("failed-to-backup-config", err, lager.Data{"config-file": h.configFilePath})
		return err
	}
	return nil
}

func (h *Configurer) writeToConfig(cfgContent []byte) error {
	tmpConfigFileName := fmt.Sprintf("%s.tmp", h.configFilePath)
	err := utils.WriteToFile(cfgContent, tmpConfigFileName)
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
