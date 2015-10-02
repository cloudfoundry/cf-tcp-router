package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/cf-tcp-router/config"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"
	"github.com/cloudfoundry-incubator/cf-tcp-router/syncer"
	"github.com/cloudfoundry-incubator/cf-tcp-router/watcher"
	"github.com/cloudfoundry-incubator/routing-api"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	"github.com/cloudfoundry/dropsonde"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

var tcpLoadBalancer = flag.String(
	"tcpLoadBalancer",
	configurer.HaProxyConfigurer,
	"The tcp load balancer to use.",
)

var tcpLoadBalancerBaseCfg = flag.String(
	"tcpLoadBalancerBaseConfig",
	"",
	"The tcp load balancer base configuration file name. This contains the basic header information.",
)

var tcpLoadBalancerCfg = flag.String(
	"tcpLoadBalancerConfig",
	"",
	"The tcp load balancer configuration file name.",
)

var subscriptionRetryInterval = flag.Int(
	"subscriptionRetryInterval",
	5,
	"Retry interval between retries to subscribe for tcp events from routing api (in seconds)",
)

var configFile = flag.String(
	"config",
	"/var/vcap/jobs/router_configurer/config/router_configurer.yml",
	"The Router configurer yml config.",
)

var syncInterval = flag.Duration(
	"syncInterval",
	time.Minute,
	"The interval between syncs of the routing table from routing api.",
)

const (
	dropsondeDestination = "localhost:3457"
	dropsondeOrigin      = "router-configurer"
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, reconfigurableSink := cf_lager.New("router-configurer")
	logger.Info("starting")

	initializeDropsonde(logger)

	routingTable := models.NewRoutingTable()
	configurer := configurer.NewConfigurer(logger,
		*tcpLoadBalancer, *tcpLoadBalancerBaseCfg, *tcpLoadBalancerCfg)

	cfg, err := config.New(*configFile)
	if err != nil {
		logger.Error("failed-to-unmarshal-config-file", err)
		os.Exit(1)
	}
	tokenFetcher := token_fetcher.NewTokenFetcher(&cfg.OAuth)

	routingApiAddress := fmt.Sprintf("%s:%d", cfg.RoutingApi.Uri, cfg.RoutingApi.Port)
	logger.Debug("creating-routing-api-client", lager.Data{"api-location": routingApiAddress})
	routingApiClient := routing_api.NewClient(routingApiAddress)

	updater := routing_table.NewUpdater(logger, &routingTable, configurer, routingApiClient, tokenFetcher)
	syncChannel := make(chan struct{})
	clock := clock.NewClock()
	syncRunner := syncer.New(clock, *syncInterval, syncChannel, logger)
	watcher := watcher.New(routingApiClient, updater, tokenFetcher, *subscriptionRetryInterval, syncChannel, logger)

	members := grouper.Members{
		{"watcher", watcher},
		{"syncer", syncRunner},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		members = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	logger.Info("started")

	err = <-monitor.Wait()
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
