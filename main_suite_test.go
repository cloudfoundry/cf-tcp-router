package main_test

import (
	testHelpers "code.cloudfoundry.org/routing-api/test_helpers"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	tlsHelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	"code.cloudfoundry.org/cf-tcp-router/config"
	"code.cloudfoundry.org/cf-tcp-router/testutil"
	"code.cloudfoundry.org/cf-tcp-router/utils"
	"code.cloudfoundry.org/localip"
	locketConfig "code.cloudfoundry.org/locket/cmd/locket/config"
	"code.cloudfoundry.org/locket/cmd/locket/testrunner"
	routingAPI "code.cloudfoundry.org/routing-api"
	routingAPIConfig "code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	tcpRouterPath                  string
	routingAPIBinPath              string
	haproxyConfigFile              string
	haproxyConfigBackupFile        string
	haproxyBaseConfigFile          string
	dbAllocator                    testHelpers.DbAllocator
	dbId                           string
	locketDBAllocator              testHelpers.DbAllocator
	locketBinPath                  string
	locketProcess                  ifrit.Process
	locketPort                     uint16
	locketDbConfig                 *routingAPIConfig.SqlDB
	routingAPIAddress              string
	routingAPIArgs                 testHelpers.Args
	routingAPIPort                 int
	routingAPIMTLSPort             int
	routingAPIIP                   string
	routingApiClient               routingAPI.Client
	routingAPICAFileName           string
	routingAPICAPrivateKey         *rsa.PrivateKey
	routingAPIClientCertPath       string
	routingAPIClientPrivateKeyPath string
	longRunningProcessPidFile      string
	catCmd                         *exec.Cmd
)

func nextAvailPort() int {
	port, err := localip.LocalPort()
	Expect(err).ToNot(HaveOccurred())

	return int(port)
}

func TestTCPRouter(test *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(test, "TCPRouter Suite")
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

	tcpRouterPath = context["tcp-router"]
	routingAPIBinPath = context["routing-api"]
	locketBinPath = context["locket"]

	setupDB()
	locketPort = uint16(nextAvailPort())
	locketDBAllocator = testHelpers.NewDbAllocator()

	locketDbConfig, err = locketDBAllocator.Create()
	Expect(err).NotTo(HaveOccurred())

})

func setupDB() {
	dbAllocator = testHelpers.NewDbAllocator()

	dbConfig, err := dbAllocator.Create()
	Expect(err).NotTo(HaveOccurred())
	dbId = dbConfig.Schema
}

func setupLongRunningProcess() {
	catCmd = exec.Command("cat")
	err := catCmd.Start()
	Expect(err).ToNot(HaveOccurred())
	pid := catCmd.Process.Pid

	file, err := os.CreateTemp(os.TempDir(), "test-pid-file")
	Expect(err).ToNot(HaveOccurred())
	_, err = file.WriteString(fmt.Sprintf("%d", pid))
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	longRunningProcessPidFile = file.Name()
}

func killLongRunningProcess() {
	isAlive := catCmd.ProcessState == nil
	if isAlive {
		err := catCmd.Process.Kill()
		Expect(err).ToNot(HaveOccurred())
	}
	Expect(os.Remove(longRunningProcessPidFile)).To(Succeed())
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

	routingAPIPort = nextAvailPort()
	routingAPIMTLSPort = nextAvailPort()
	routingAPIIP = "127.0.0.1"
	routingAPIAddress = fmt.Sprintf("https://%s:%d", routingAPIIP, routingAPIMTLSPort)

	dbCACert := os.Getenv("SQL_SERVER_CA_CERT")

	routingAPICAFileName, routingAPICAPrivateKey = tlsHelpers.GenerateCa()
	routingAPIServerCertPath, routingAPIServerKeyPath, _ := tlsHelpers.GenerateCertAndKey(routingAPICAFileName, routingAPICAPrivateKey)

	routingAPIArgs, err = testHelpers.NewRoutingAPIArgs(
		routingAPIIP,
		routingAPIPort,
		routingAPIMTLSPort,
		dbId,
		dbCACert,
		fmt.Sprintf("localhost:%d", locketPort),
		routingAPICAFileName,
		routingAPIServerCertPath,
		routingAPIServerKeyPath,
	)
	Expect(err).NotTo(HaveOccurred())

	routingAPIClientCertPath, routingAPIClientPrivateKeyPath, _ = tlsHelpers.GenerateCertAndKey(routingAPICAFileName, routingAPICAPrivateKey)

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(routingAPIClientCertPath, routingAPIClientPrivateKeyPath),
	).Client(
		tlsconfig.WithAuthorityFromFile(routingAPICAFileName),
	)
	Expect(err).NotTo(HaveOccurred())
	routingApiClient = routingAPI.NewClientWithTLSConfig(routingAPIAddress, tlsConfig)

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
	locketRunner := testrunner.NewLocketRunner(locketBinPath, func(config *locketConfig.LocketConfig) {
		switch testHelpers.Database {
		case testHelpers.Postgres:
			config.DatabaseConnectionString = fmt.Sprintf(
				"user=%s password=%s host=%s dbname=%s",
				testHelpers.PostgresUsername,
				testHelpers.PostgresPassword,
				testHelpers.Host,
				locketDbConfig.Schema,
			)
			config.DatabaseDriver = testHelpers.Postgres
		default:
			config.DatabaseConnectionString = fmt.Sprintf(
				"%s:%s@/%s",
				testHelpers.MySQLUserName,
				testHelpers.MySQLPassword,
				locketDbConfig.Schema,
			)
			config.DatabaseDriver = testHelpers.MySQL
		}
		config.ListenAddress = fmt.Sprintf("%s:%d", testHelpers.Host, locketPort)
	})
	locketProcess = ginkgomon.Invoke(locketRunner)
}

func teardownLocket() {
	ginkgomon.Interrupt(locketProcess, 5*time.Second)
}

func getRouterGroupGuid(routingApiClient routingAPI.Client) string {
	var routerGroups []models.RouterGroup
	Eventually(func() error {
		var err error
		routerGroups, err = routingApiClient.RouterGroups()
		return err
	}, "30s", "1s").ShouldNot(HaveOccurred(), "Failed to connect to Routing API server after 30s.")
	Expect(routerGroups).ToNot(HaveLen(0))
	return routerGroups[0].Guid
}

func generateTCPRouterConfigFile(oauthServerPort int, uaaCACertsPath string, routingApiAuthDisabled bool, reserved_routing_ports ...int) string {
	tcpRouterConfig := config.Config{
		ReservedSystemComponentPorts: reserved_routing_ports,
		OAuth: config.OAuthConfig{
			TokenEndpoint:     testHelpers.RoutingAPIIP,
			SkipSSLValidation: false,
			CACerts:           uaaCACertsPath,
			ClientName:        "someclient",
			ClientSecret:      "somesecret",
			Port:              oauthServerPort,
		},
		DrainWaitDuration: 3 * time.Second,
		RoutingAPI: config.RoutingAPIConfig{
			AuthDisabled: routingApiAuthDisabled,
		},
		HaProxyPidFile: longRunningProcessPidFile,
		IsolationSegments: []string{
			"foo-iso-seg",
		},
	}

	tcpRouterConfig.RoutingAPI.URI = fmt.Sprintf("https://%s", testHelpers.RoutingAPIIP)
	tcpRouterConfig.RoutingAPI.Port = routingAPIMTLSPort
	tcpRouterConfig.RoutingAPI.ClientCertificatePath = routingAPIClientCertPath
	tcpRouterConfig.RoutingAPI.ClientPrivateKeyPath = routingAPIClientPrivateKeyPath
	tcpRouterConfig.RoutingAPI.CACertificatePath = routingAPICAFileName

	bs, err := yaml.Marshal(tcpRouterConfig)
	Expect(err).NotTo(HaveOccurred())

	randomConfigFile, err := os.CreateTemp("", "tcp_router")
	Expect(err).ShouldNot(HaveOccurred())
	// Close file because we write using path instead of file handle
	randomConfigFile.Close()

	configFilePath := randomConfigFile.Name()
	Expect(utils.FileExists(configFilePath)).To(BeTrue())

	err = utils.WriteToFile(bs, configFilePath)
	Expect(err).ShouldNot(HaveOccurred())
	return configFilePath
}
