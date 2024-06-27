package models_test

import (
	. "code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("HAProxyConfig", func() {
	Describe("NewHAProxyConfig", func() {
		var (
			logger       lager.Logger
			routingTable RoutingTable
		)

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("haproxy-config-test")
			routingTable = NewRoutingTable(logger)
		})

		Context("when a frontend is invalid", func() {
			validRoutingTableEntry := RoutingTableEntry{
				Backends: map[BackendServerKey]BackendServerDetails{
					BackendServerKey{Address: "valid-host.internal", Port: 1111}: {},
				},
			}

			Context("because it contains an invalid port", func() {
				It("retains only valid frontends", func() {
					routingTable.Entries[RoutingKey{Port: 0}] = validRoutingTableEntry
					routingTable.Entries[RoutingKey{Port: 80}] = validRoutingTableEntry

					Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
						80: {
							"": {
								{Address: "valid-host.internal", Port: 1111},
							},
						},
					}))
				})
			})

			Context("because it contains an invalid SNI hostname", func() {
				It("retains only valid frontends", func() {
					routingTable.Entries[RoutingKey{Port: 80, SniHostname: "valid-host.example.com"}] = validRoutingTableEntry
					routingTable.Entries[RoutingKey{Port: 90, SniHostname: "!invalid-host.example.com"}] = validRoutingTableEntry
					routingTable.Entries[RoutingKey{Port: 100, SniHostname: "ünvalid-host.example.com"}] = validRoutingTableEntry

					Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
						80: {
							"valid-host.example.com": {
								{Address: "valid-host.internal", Port: 1111},
							},
						},
					}))
				})
			})

			Context("because it contains no backends", func() {
				It("retains only valid frontends", func() {
					routingTable.Entries[RoutingKey{Port: 80}] = validRoutingTableEntry
					routingTable.Entries[RoutingKey{Port: 90}] = RoutingTableEntry{
						Backends: map[BackendServerKey]BackendServerDetails{},
					}

					Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
						80: {
							"": {
								{Address: "valid-host.internal", Port: 1111},
							},
						},
					}))
				})
			})

			Context("because a backend is invalid", func() {
				Context("because it contains an invalid address", func() {
					It("retains only valid backends", func() {
						routingTable.Entries[RoutingKey{Port: 80}] = RoutingTableEntry{
							Backends: map[BackendServerKey]BackendServerDetails{
								BackendServerKey{Address: "valid-host.internal", Port: 1111}:    {},
								BackendServerKey{Address: "!invalid-host.internal", Port: 2222}: {},
							},
						}

						Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
							80: {
								"": {
									{Address: "valid-host.internal", Port: 1111},
								},
							},
						}))
					})
				})

				Context("because it contains an invalid port", func() {
					It("retains only valid backends", func() {
						routingTable.Entries[RoutingKey{Port: 80}] = RoutingTableEntry{
							Backends: map[BackendServerKey]BackendServerDetails{
								BackendServerKey{Address: "valid-host-1.example.com", Port: 1111}: {},
								BackendServerKey{Address: "valid-host-2.example.com", Port: 0}:    {},
							},
						}

						Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
							80: {
								"": {
									{Address: "valid-host-1.example.com", Port: 1111},
								},
							},
						}))
					})
				})

				Context("because backend TLS port is supplied but instance_id is not", func() {
					It("logs an error", func() {
						routingTable.Entries[RoutingKey{Port: 81}] = RoutingTableEntry{
							Backends: map[BackendServerKey]BackendServerDetails{
								BackendServerKey{Address: "valid-host-1.example.com", Port: 1111, TLSPort: 1443}: {},
							},
						}

						Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{}))
						Eventually(logger).Should(gbytes.Say(`Field: backend_configuration.instance_id. Value: unset.`))
					})
				})
			})
		})

		Context("when TLS port and instance_id are provided", func() {
			It("creates a valid HAProxyConfig", func() {
				instanceId := "foo"
				routingTable.Entries[RoutingKey{Port: 80}] = RoutingTableEntry{
					Backends: map[BackendServerKey]BackendServerDetails{
						BackendServerKey{Address: "valid-host-1.example.com", Port: 1111, TLSPort: 1443, InstanceID: instanceId}: {},
					},
				}

				Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
					80: {
						"": {
							{Address: "valid-host-1.example.com", Port: 1111, TLSPort: 1443, InstanceID: instanceId},
						},
					},
				}))
			})
		})

		Context("when multiple valid frontends exist", func() {
			It("includes all frontends", func() {
				routingTable.Entries[RoutingKey{Port: 80}] = RoutingTableEntry{
					Backends: map[BackendServerKey]BackendServerDetails{
						BackendServerKey{Address: "valid-host-1.internal", Port: 2222}: {},
						BackendServerKey{Address: "valid-host-1.internal", Port: 1111}: {},
					},
				}
				routingTable.Entries[RoutingKey{Port: 90, SniHostname: "valid-host.example.com"}] = RoutingTableEntry{
					Backends: map[BackendServerKey]BackendServerDetails{
						BackendServerKey{Address: "valid-host-4.internal", Port: 8888}: {},
						BackendServerKey{Address: "valid-host-4.internal", Port: 7777}: {},
						BackendServerKey{Address: "valid-host-3.internal", Port: 6666}: {},
						BackendServerKey{Address: "valid-host-3.internal", Port: 5555}: {},
					},
				}
				routingTable.Entries[RoutingKey{Port: 90}] = RoutingTableEntry{
					Backends: map[BackendServerKey]BackendServerDetails{
						BackendServerKey{Address: "valid-host-2.internal", Port: 4444}: {},
						BackendServerKey{Address: "valid-host-2.internal", Port: 3333}: {},
					},
				}

				Expect(NewHAProxyConfig(routingTable, logger)).To(Equal(HAProxyConfig{
					80: {
						"": {
							{Address: "valid-host-1.internal", Port: 1111},
							{Address: "valid-host-1.internal", Port: 2222},
						},
					},
					90: {
						"": {
							{Address: "valid-host-2.internal", Port: 3333},
							{Address: "valid-host-2.internal", Port: 4444},
						},
						"valid-host.example.com": {
							{Address: "valid-host-3.internal", Port: 5555},
							{Address: "valid-host-3.internal", Port: 6666},
							{Address: "valid-host-4.internal", Port: 7777},
							{Address: "valid-host-4.internal", Port: 8888},
						},
					},
				}))
			})
		})
	})
})
