package routing_table_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer/fakes"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Updater", func() {
	const (
		externalPort1   = uint16(2222)
		externalPort2   = uint16(2223)
		routerGroupGuid = "rtrgrp001"
	)
	var (
		routingTable               models.RoutingTable
		existingRoutingKey1        models.RoutingKey
		existingRoutingTableEntry1 models.RoutingTableEntry
		existingRoutingKey2        models.RoutingKey
		existingRoutingTableEntry2 models.RoutingTableEntry
		updater                    routing_table.Updater
		fakeConfigurer             *fakes.FakeRouterConfigurer
	)

	verifyRoutingTableEntry := func(key models.RoutingKey, entry models.RoutingTableEntry) {
		existingEntry := routingTable.Get(key)
		Expect(existingEntry).NotTo(BeZero())
		Expect(existingEntry).Should(Equal(entry))
	}

	BeforeEach(func() {
		routingTable = models.NewRoutingTable()

		fakeConfigurer = new(fakes.FakeRouterConfigurer)
		existingRoutingKey1 = models.RoutingKey{externalPort1}
		existingRoutingTableEntry1 = models.NewRoutingTableEntry(
			models.BackendServerInfos{
				models.BackendServerInfo{"some-ip-1", 1234},
				models.BackendServerInfo{"some-ip-2", 1234},
			},
		)
		Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())

		existingRoutingKey2 = models.RoutingKey{externalPort2}
		existingRoutingTableEntry2 = models.NewRoutingTableEntry(
			models.BackendServerInfos{
				models.BackendServerInfo{"some-ip-3", 2345},
				models.BackendServerInfo{"some-ip-4", 2345},
			},
		)
		Expect(routingTable.Set(existingRoutingKey2, existingRoutingTableEntry2)).To(BeTrue())

		updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer)
	})

	Describe("HandleEvent", func() {
		const (
			externalPort4 = uint16(2224)
			externalPort5 = uint16(2225)
			externalPort6 = uint16(2226)
		)
		var (
			tcpEvent routing_api.TcpEvent
		)
		Context("when Upsert event is received", func() {
			Context("when entry does not exist", func() {
				BeforeEach(func() {
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: db.TcpRouteMapping{
							TcpRoute: db.TcpRoute{
								RouterGroupGuid: routerGroupGuid,
								ExternalPort:    externalPort4,
							},
							HostPort: 2346,
							HostIP:   "some-ip-4",
						},
						Action: "Upsert",
					}
				})

				It("inserts handle the event and inserts the new entry", func() {
					err := updater.HandleEvent(tcpEvent)
					Expect(err).NotTo(HaveOccurred())
					expectedRoutingTableEntry := models.NewRoutingTableEntry(
						models.BackendServerInfos{
							models.BackendServerInfo{"some-ip-4", 2346},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{externalPort4}, expectedRoutingTableEntry)
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
				})
			})

			Context("when entry does exist", func() {
				Context("an existing backend is provided", func() {
					BeforeEach(func() {
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: db.TcpRouteMapping{
								TcpRoute: db.TcpRoute{
									RouterGroupGuid: routerGroupGuid,
									ExternalPort:    externalPort1,
								},
								HostPort: 1234,
								HostIP:   "some-ip-1",
							},
							Action: "Upsert",
						}
					})

					It("does not call configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						verifyRoutingTableEntry(existingRoutingKey1, existingRoutingTableEntry1)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
					})
				})

				Context("and a new backend is provided", func() {
					BeforeEach(func() {
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: db.TcpRouteMapping{
								TcpRoute: db.TcpRoute{
									RouterGroupGuid: routerGroupGuid,
									ExternalPort:    externalPort1,
								},
								HostPort: 1234,
								HostIP:   "some-ip-5",
							},
							Action: "Upsert",
						}
					})

					It("adds backend to existing entry and calls configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1234},
								models.BackendServerInfo{"some-ip-5", 1234},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey1, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
					})

					Context("when Configurer return an error", func() {
						BeforeEach(func() {
							fakeConfigurer.ConfigureReturns(errors.New("kaboom"))
						})

						It("returns error", func() {
							err := updater.HandleEvent(tcpEvent)
							Expect(err).To(HaveOccurred())
						})
					})
				})
			})
		})

		Context("when Delete event is received", func() {
			Context("when entry does not exist", func() {
				BeforeEach(func() {
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: db.TcpRouteMapping{
							TcpRoute: db.TcpRoute{
								RouterGroupGuid: routerGroupGuid,
								ExternalPort:    externalPort4,
							},
							HostPort: 2346,
							HostIP:   "some-ip-4",
						},
						Action: "Delete",
					}
				})

				It("does not call configurer", func() {
					err := updater.HandleEvent(tcpEvent)
					Expect(err).NotTo(HaveOccurred())
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
				})
			})

			Context("when entry does exist", func() {

				Context("an existing backend is provided", func() {
					var (
						existingRoutingKey5        models.RoutingKey
						existingRoutingTableEntry5 models.RoutingTableEntry
					)
					BeforeEach(func() {
						existingRoutingKey5 = models.RoutingKey{externalPort5}
						existingRoutingTableEntry5 = models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1234},
							},
						)
						Expect(routingTable.Set(existingRoutingKey5, existingRoutingTableEntry5)).To(BeTrue())
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: db.TcpRouteMapping{
								TcpRoute: db.TcpRoute{
									RouterGroupGuid: routerGroupGuid,
									ExternalPort:    externalPort5,
								},
								HostPort: 1234,
								HostIP:   "some-ip-1",
							},
							Action: "Delete",
						}
					})

					It("deletes backend from entry and calls configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-2", 1234},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey5, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
					})

					Context("when Configurer return an error", func() {
						BeforeEach(func() {
							fakeConfigurer.ConfigureReturns(errors.New("kaboom"))
						})

						It("returns error", func() {
							err := updater.HandleEvent(tcpEvent)
							Expect(err).To(HaveOccurred())
						})
					})
				})

				Context("and a new backend is provided", func() {
					var (
						existingRoutingKey6        models.RoutingKey
						existingRoutingTableEntry6 models.RoutingTableEntry
					)
					BeforeEach(func() {
						existingRoutingKey6 = models.RoutingKey{externalPort5}
						existingRoutingTableEntry6 = models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1234},
							},
						)
						Expect(routingTable.Set(existingRoutingKey6, existingRoutingTableEntry6)).To(BeTrue())

						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: db.TcpRouteMapping{
								TcpRoute: db.TcpRoute{
									RouterGroupGuid: routerGroupGuid,
									ExternalPort:    externalPort5,
								},
								HostPort: 1234,
								HostIP:   "some-ip-5",
							},
							Action: "Delete",
						}
					})

					It("does not call configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							models.BackendServerInfos{
								models.BackendServerInfo{"some-ip-1", 1234},
								models.BackendServerInfo{"some-ip-2", 1234},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey6, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
					})
				})
			})
		})
	})
})
