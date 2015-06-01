package main_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/GESoftware-CF/cf-tcp-router/cmd/router-configurer/testrunner"
	"github.com/GESoftware-CF/cf-tcp-router/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
)

const (
	haproxyCfgTemplate = "configurer/haproxy/fixtures/haproxy.cfg.template"
)

var (
	routerConfigurerPath    string
	routerConfigurerPort    int
	routerConfigurerProcess ifrit.Process
	haproxyConfigFile       string
	haproxyConfigBackupFile string
)

func TestRouterConfigurer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RouterConfigurer Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	routerConfigurer, err := gexec.Build("github.com/GESoftware-CF/cf-tcp-router/cmd/router-configurer", "-race")
	Expect(err).NotTo(HaveOccurred())
	payload, err := json.Marshal(map[string]string{
		"router-configurer": routerConfigurer,
	})

	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	context := map[string]string{}

	err := json.Unmarshal(payload, &context)
	Expect(err).NotTo(HaveOccurred())

	routerConfigurerPort = 7000 + GinkgoParallelNode()
	routerConfigurerPath = context["router-configurer"]
})

var _ = BeforeEach(func() {
	rand.Seed(17 * time.Now().UTC().UnixNano())
	randomFileName := fmt.Sprintf("haproxy_%d.cfg", rand.Int31())
	randomBackupFileName := fmt.Sprintf("%s.bak", randomFileName)
	haproxyConfigFile = path.Join(os.TempDir(), randomFileName)
	haproxyConfigBackupFile = path.Join(os.TempDir(), randomBackupFileName)
	err := utils.WriteToFile(
		[]byte(
			`global maxconn 4096
defaults
  log global
  timeout connect 300000
  timeout client 300000
  timeout server 300000
  maxconn 2000`),
		haproxyConfigFile)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(utils.FileExists(haproxyConfigFile)).To(BeTrue())

	routerConfigurerArgs := testrunner.Args{
		Address:        fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
		ConfigFilePath: haproxyConfigFile,
	}

	runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
	routerConfigurerProcess = ifrit.Invoke(runner)
})

var _ = AfterEach(func() {
	ginkgomon.Kill(routerConfigurerProcess, 5*time.Second)
	err := os.Remove(haproxyConfigFile)
	Expect(err).ShouldNot(HaveOccurred())

	os.Remove(haproxyConfigBackupFile)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
