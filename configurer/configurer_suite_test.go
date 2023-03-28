package configurer_test

import (
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	logger lager.Logger
)

func TestConfigurer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configurer Suite")
}

var _ = BeforeEach(func() {
	logger = lagertest.NewTestLogger("test")
})
