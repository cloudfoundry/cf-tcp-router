package metrics_reporter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"
)

var (
	logger lager.Logger
)

func TestConfigurer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Reporter Suite")
}

var _ = BeforeEach(func() {
	logger = lagertest.NewTestLogger("test")
})
