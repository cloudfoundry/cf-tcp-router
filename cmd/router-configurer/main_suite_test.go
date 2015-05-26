package main_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/GESoftware-CF/cf-tcp-router/cmd/router-configurer/testrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
)

var (
	routerConfigurerPath    string
	routerConfigurerPort    int
	routerConfigurerProcess ifrit.Process
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
	routerConfigurerArgs := testrunner.Args{
		Address: fmt.Sprintf("127.0.0.1:%d", routerConfigurerPort),
	}

	runner := testrunner.New(routerConfigurerPath, routerConfigurerArgs)
	routerConfigurerProcess = ifrit.Invoke(runner)
})

var _ = AfterEach(func() {
	ginkgomon.Kill(routerConfigurerProcess, 5*time.Second)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
