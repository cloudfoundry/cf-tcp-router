package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/testutil"
	"code.cloudfoundry.org/cf-tcp-router/utils"
	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/localip"
	"code.cloudfoundry.org/locket/cmd/locket/config"
	"code.cloudfoundry.org/locket/cmd/locket/testrunner"
	routing_api "code.cloudfoundry.org/routing-api"
	routingtestrunner "code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	routing_api_config "code.cloudfoundry.org/routing-api/config"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	tcpRouterPath           string
	routingAPIBinPath       string
	tcpRouterPort           int
	haproxyConfigFile       string
	haproxyConfigBackupFile string
	haproxyBaseConfigFile   string

	consulRunner *consulrunner.ClusterRunner
	dbAllocator  routingtestrunner.DbAllocator

	dbId string

	locketDBAllocator routingtestrunner.DbAllocator
	locketBinPath     string
	locketProcess     ifrit.Process
	locketPort        uint16
	locketDbConfig    *routing_api_config.SqlDB

	routingAPIAddress string
	routingAPIArgs    routingtestrunner.Args
	routingAPIPort    uint16
	routingAPIIP      string
	routingApiClient  routing_api.Client

	longRunningProcessPidFile string
	catCmd                    *exec.Cmd
)

func nextAvailPort() int {
	port, err := localip.LocalPort()
	Expect(err).ToNot(HaveOccurred())

	return int(port)
}

func TestTCPRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TCPRouter Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	tcpRouter, err := gexec.Build("code.cloudfoundry.org/cf-tcp-router", "-race")
	Expect(err).NotTo(HaveOccurred())
	routingAPIBin, err := gexec.Build("code.cloudfoundry.org/routing-api/cmd/routing-api", "-race")
	Expect(err).NotTo(HaveOccurred())
	locketBin, err := gexec.Build("code.cloudfoundry.org/locket/cmd/locket", "-race")
	Expect(err).NotTo(HaveOccurred())

	payload, err := json.Marshal(map[string]string{
		"tcp-router":  tcpRouter,
		"routing-api": routingAPIBin,
		"locket":      locketBin,
	})
	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	context := map[string]string{}

	err := json.Unmarshal(payload, &context)
	Expect(err).NotTo(HaveOccurred())

	tcpRouterPort = nextAvailPort()
	tcpRouterPath = context["tcp-router"]
	routingAPIBinPath = context["routing-api"]
	locketBinPath = context["locket"]

	setupDB()
	locketPort = uint16(nextAvailPort())
	locketDBAllocator = routingtestrunner.NewDbAllocator()

	locketDbConfig, err = locketDBAllocator.Create()
	Expect(err).NotTo(HaveOccurred())

})

func setupDB() {
	dbAllocator = routingtestrunner.NewDbAllocator()

	dbConfig, err := dbAllocator.Create()
	Expect(err).NotTo(HaveOccurred())
	dbId = dbConfig.Schema
}

func setupLongRunningProcess() {
	catCmd = exec.Command("cat")
	err := catCmd.Start()
	Expect(err).ToNot(HaveOccurred())
	pid := catCmd.Process.Pid

	file, err := ioutil.TempFile(os.TempDir(), "test-pid-file")
	Expect(err).ToNot(HaveOccurred())
	_, err = file.WriteString(fmt.Sprintf("%d", pid))
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	longRunningProcessPidFile = file.Name()
}

func killLongRunningProcess() {
	err := os.Remove(longRunningProcessPidFile)
	Expect(err).ToNot(HaveOccurred())
	if catCmd.ProcessState == nil {
		err := catCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())
	}
}

var _ = BeforeEach(func() {
	setupLocket()

	randomFileName := testutil.RandomFileName("haproxy_", ".cfg")
	randomBackupFileName := fmt.Sprintf("%s.bak", randomFileName)
	randomBaseFileName := testutil.RandomFileName("haproxy_base_", ".cfg")
	haproxyConfigFile = path.Join(os.TempDir(), randomFileName)
	haproxyConfigBackupFile = path.Join(os.TempDir(), randomBackupFileName)
	haproxyBaseConfigFile = path.Join(os.TempDir(), randomBaseFileName)

	err := utils.WriteToFile(
		[]byte(
			`global maxconn 4096
defaults
  log global
  timeout connect 300000
  timeout client 300000
  timeout server 300000
  maxconn 2000`),
		haproxyBaseConfigFile)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(utils.FileExists(haproxyBaseConfigFile)).To(BeTrue())

	err = utils.CopyFile(haproxyBaseConfigFile, haproxyConfigFile)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(utils.FileExists(haproxyConfigFile)).To(BeTrue())

	routingAPIPort = uint16(nextAvailPort())
	routingAPIIP = "127.0.0.1"
	routingAPIAddress = fmt.Sprintf("http://%s:%d", routingAPIIP, routingAPIPort)

	dbCACert := os.Getenv("SQL_SERVER_CA_CERT")
	routingAPIArgs, err = routingtestrunner.NewRoutingAPIArgs(
		routingAPIIP,
		routingAPIPort,
		dbId,
		dbCACert,
		fmt.Sprintf("localhost:%d", locketPort),
	)
	Expect(err).NotTo(HaveOccurred())

	routingApiClient = routing_api.NewClient(routingAPIAddress, false)

	setupLongRunningProcess()
})

var _ = AfterEach(func() {
	err := os.Remove(haproxyConfigFile)
	Expect(err).ShouldNot(HaveOccurred())

	os.Remove(haproxyConfigBackupFile)

	teardownLocket()
	dbAllocator.Reset()
	locketDBAllocator.Reset()
	killLongRunningProcess()
})

var _ = SynchronizedAfterSuite(func() {
	dbAllocator.Delete()
	locketDBAllocator.Delete()
}, func() {
	gexec.CleanupBuildArtifacts()
})

func setupLocket() {
	locketRunner := testrunner.NewLocketRunner(locketBinPath, func(c *config.LocketConfig) {
		c.DatabaseConnectionString = "root:password@/" + locketDbConfig.Schema
		c.ListenAddress = fmt.Sprintf("localhost:%d", locketPort)
	})
	locketProcess = ginkgomon.Invoke(locketRunner)
}

func teardownLocket() {
	ginkgomon.Interrupt(locketProcess, 5*time.Second)
}

func setupConsul() {
	consulRunner = consulrunner.NewClusterRunner(consulrunner.ClusterRunnerConfig{
		StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
		NumNodes:     1,
		Scheme:       "http",
	})
	consulRunner.Start()
	consulRunner.WaitUntilReady()
}

func teardownConsul() {
	consulRunner.Stop()
}
