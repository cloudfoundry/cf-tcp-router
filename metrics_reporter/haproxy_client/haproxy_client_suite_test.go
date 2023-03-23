package haproxy_client_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHaproxyClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HaproxyClient Suite")
}
