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
	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter"
	"github.com/cloudfoundry-incubator/cf-tcp-router/metrics_reporter/haproxy_client"
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

const (
	defaultTokenFetchRetryInterval = 5 * time.Second
	defaultTokenFetchNumRetries    = uint(3)
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

var tcpLoadBalancerStatsUnixSocket = flag.String(
	"tcpLoadBalancerStatsUnixSocket",
	"/var/vcap/jobs/haproxy/config/haproxy.sock",
	"Unix domain socket for tcp load balancer",
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

var tokenFetchMaxRetries = flag.Uint(
	"tokenFetchMaxRetries",
	defaultTokenFetchNumRetries,
	"Maximum number of retries the Token Fetcher will use every time FetchToken is called",
)

var tokenFetchRetryInterval = flag.Duration(
	"tokenFetchRetryInterval",
	defaultTokenFetchRetryInterval,
	"interval to wait before TokenFetcher retries to fetch a token",
)

var tokenFetchExpirationBufferTime = flag.Uint64(
	"tokenFetchExpirationBufferTime",
	30,
	"Buffer time in seconds before the actual token expiration time, when TokenFetcher consider a token expired",
)

var statsCollectionInterval = flag.Duration(
	"statsCollectionInterval",
	time.Minute,
	"The interval between collection of stats from tcp load balancer.",
)

var dropsondePort = flag.Int(
	"dropsondePort",
	3457,
	"Port the local metron agent is listening on",
)

const (
	dropsondeOrigin        = "router-configurer"
	statsConnectionTimeout = 10 * time.Second
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, reconfigurableSink := cf_lager.New("router-configurer")
	logger.Info("starting")
	clock := clock.NewClock()

	initializeDropsonde(logger)

	routingTable := models.NewRoutingTable()
	configurer := configurer.NewConfigurer(logger,
		*tcpLoadBalancer, *tcpLoadBalancerBaseCfg, *tcpLoadBalancerCfg)

	cfg, err := config.New(*configFile)
	if err != nil {
		logger.Error("failed-to-unmarshal-config-file", err)
		os.Exit(1)
	}
	tokenFetcher, err := createTokenFetcher(logger, cfg, clock)
	if err != nil {
		logger.Error("failed-to-get-token-fetcher", err)
		os.Exit(1)
	}
	_, err = tokenFetcher.FetchToken(false)
	if err != nil {
		logger.Error("error-fetching-oauth-token", err)
		os.Exit(1)
	}

	routingAPIAddress := fmt.Sprintf("%s:%d", cfg.RoutingAPI.URI, cfg.RoutingAPI.Port)
	logger.Debug("creating-routing-api-client", lager.Data{"api-location": routingAPIAddress})
	routingAPIClient := routing_api.NewClient(routingAPIAddress)

	updater := routing_table.NewUpdater(logger, &routingTable, configurer, routingAPIClient, tokenFetcher)
	syncChannel := make(chan struct{})
	syncRunner := syncer.New(clock, *syncInterval, syncChannel, logger)
	watcher := watcher.New(routingAPIClient, updater, tokenFetcher, *subscriptionRetryInterval, syncChannel, logger)

	haproxyClient := haproxy_client.NewClient(logger, *tcpLoadBalancerStatsUnixSocket, statsConnectionTimeout)
	metricsEmitter := metrics_reporter.NewMetricsEmitter()
	metricsReporter := metrics_reporter.NewMetricsReporter(clock, haproxyClient, metricsEmitter, *statsCollectionInterval)

	members := grouper.Members{
		{"watcher", watcher},
		{"syncer", syncRunner},
		{"metricsReporter", metricsReporter},
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

func createTokenFetcher(logger lager.Logger, cfg *config.Config, klok clock.Clock) (token_fetcher.TokenFetcher, error) {
	if cfg.RoutingAPI.AuthDisabled {
		logger.Debug("creating-noop-token-fetcher")
		tknFetcher := token_fetcher.NewNoOpTokenFetcher()
		return tknFetcher, nil
	}
	logger.Debug("creating-uaa-token-fetcher")

	tokenFetcherConfig := token_fetcher.TokenFetcherConfig{
		MaxNumberOfRetries:   uint32(*tokenFetchMaxRetries),
		RetryInterval:        *tokenFetchRetryInterval,
		ExpirationBufferTime: int64(*tokenFetchExpirationBufferTime),
	}
	return token_fetcher.NewTokenFetcher(logger, &cfg.OAuth, tokenFetcherConfig, klok)
}

func initializeDropsonde(logger lager.Logger) {
	dropsondeDestination := fmt.Sprintf("localhost:%d", *dropsondePort)
	err := dropsonde.Initialize(dropsondeDestination, dropsondeOrigin)
	if err != nil {
		logger.Error("failed-to-initialize-dropsonde", err)
	}
}
