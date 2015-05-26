package testutil

import (
	"net"

	. "github.com/onsi/gomega"
)

func GetExternalIP() string {
	var externalIP string
	addrs, err := net.InterfaceAddrs()
	Expect(err).ShouldNot(HaveOccurred())
	for _, addr := range addrs {
		ip, _, _ := net.ParseCIDR(addr.String())
		if ipv4 := ip.To4(); ipv4 != nil {
			if ipv4.IsLoopback() == false {
				externalIP = ipv4.String()
				break
			}
		}
	}
	return externalIP
}
