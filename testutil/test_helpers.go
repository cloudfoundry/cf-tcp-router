package testutil

import (
	"fmt"
	"net"

	. "github.com/onsi/gomega"

	uuid "github.com/nu7hatch/gouuid"
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

func RandomFileName(prefix string, suffix string) string {
	guid, err := uuid.NewV4()
	Expect(err).ShouldNot(HaveOccurred())
	return fmt.Sprintf("%s%s%s", prefix, guid, suffix)
}
