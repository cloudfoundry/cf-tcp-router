package main

import (
	"errors"
	"flag"
	"math"
	"os"

	"github.com/GESoftware-CF/cf-tcp-router/configurer"
	"github.com/GESoftware-CF/cf-tcp-router/handlers"
	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var serverAddress = flag.String(
	"address",
	"",
	"The host:port that the server is bound to.",
)

var tcpLoadBalancer = flag.String(
	"tcpLoadBalancer",
	configurer.HaProxyConfigurer,
	"The tcp load balancer to use.",
)

var tcpLoadBalancerCfg = flag.String(
	"tcpLoadBalancerConfig",
	"",
	"The tcp load balancer configuration file name.",
)

var startExternalPort = flag.Uint(
	"startExternalPort",
	defaultStartExternalPort,
	"The port number from which the router will start allocating ports when non particular port is requested.",
)

const (
	dropsondeDestination     = "localhost:3457"
	dropsondeOrigin          = "receptor"
	defaultStartExternalPort = 60000
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, reconfigurableSink := cf_lager.New("router-configurer")
	logger.Info("starting")

	initializeDropsonde(logger)

	if *startExternalPort > math.MaxUint16 {
		logger.Fatal("invalid-start-externalport", errors.New("Start ExternalPort must be within the range of 1024...65535"))
	}

	handler := handlers.New(logger, configurer.NewConfigurer(logger,
		*tcpLoadBalancer, *tcpLoadBalancerCfg, uint16(*startExternalPort)))

	members := grouper.Members{
		{"server", http_server.New(*serverAddress, handler)},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		members = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	logger.Info("started")

	err := <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func initializeDropsonde(logger lager.Logger) {
	err := dropsonde.Initialize(dropsondeDestination, dropsondeOrigin)
	if err != nil {
		logger.Error("failed to initialize dropsonde: %v", err)
	}
}
