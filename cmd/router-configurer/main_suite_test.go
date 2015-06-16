package main_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/cloudfoundry-incubator/cf-tcp-router/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"

	"testing"
)

const (
	haproxyCfgTemplate = "configurer/haproxy/fixtures/haproxy.cfg.template"
	startExternalPort  = 64000
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
	routerConfigurer, err := gexec.Build("github.com/cloudfoundry-incubator/cf-tcp-router/cmd/router-configurer", "-race")
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

})

var _ = AfterEach(func() {
	err := os.Remove(haproxyConfigFile)
	Expect(err).ShouldNot(HaveOccurred())

	os.Remove(haproxyConfigBackupFile)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
