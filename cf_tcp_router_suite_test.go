package cf_tcp_router_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfTcpRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfTcpRouter Suite")
}
