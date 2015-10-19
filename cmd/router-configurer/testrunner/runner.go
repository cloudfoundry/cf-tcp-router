package testrunner

import (
	"os/exec"
	"strconv"
	"time"

	"github.com/tedsuo/ifrit/ginkgomon"
)

type Args struct {
	BaseLoadBalancerConfigFilePath string
	LoadBalancerConfigFilePath     string
	ConfigFilePath                 string
	RoutingApiAuthEnabled          bool
}

func (args Args) ArgSlice() []string {
	return []string{
		"-tcpLoadBalancerConfig=" + args.LoadBalancerConfigFilePath,
		"-tcpLoadBalancerBaseConfig=" + args.BaseLoadBalancerConfigFilePath,
		"-config=" + args.ConfigFilePath,
		"-logLevel=debug",
		"-routingApiAuthEnabled=" + strconv.FormatBool(args.RoutingApiAuthEnabled),
	}
}

func New(binPath string, args Args) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:              "router-configurer",
		AnsiColorCode:     "1;97m",
		StartCheck:        "router-configurer.started",
		StartCheckTimeout: 10 * time.Second,
		Command:           exec.Command(binPath, args.ArgSlice()...),
	})
}
