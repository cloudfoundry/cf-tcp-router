package routing_table_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer/fakes"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	apimodels "github.com/cloudfoundry-incubator/routing-api/models"
	testUaaClient "github.com/cloudfoundry-incubator/uaa-go-client/fakes"
	"github.com/cloudfoundry-incubator/uaa-go-client/schema"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Updater", func() {
	const (
		externalPort1   = uint16(2222)
		externalPort2   = uint16(2223)
		externalPort4   = uint16(2224)
		externalPort5   = uint16(2225)
		externalPort6   = uint16(2226)
		routerGroupGuid = "rtrgrp001"
	)
	var (
		routingTable               *models.RoutingTable
		existingRoutingKey1        models.RoutingKey
		existingRoutingTableEntry1 models.RoutingTableEntry
		existingRoutingKey2        models.RoutingKey
		existingRoutingTableEntry2 models.RoutingTableEntry
		updater                    routing_table.Updater
		fakeConfigurer             *fakes.FakeRouterConfigurer
		fakeRoutingApiClient       *fake_routing_api.FakeClient
		fakeUaaClient              *testUaaClient.FakeClient
		tcpEvent                   routing_api.TcpEvent
	)

	verifyRoutingTableEntry := func(key models.RoutingKey, entry models.RoutingTableEntry) {
		existingEntry := routingTable.Get(key)
		Expect(existingEntry).NotTo(BeZero())
		Expect(existingEntry).Should(Equal(entry))
	}

	BeforeEach(func() {
		fakeConfigurer = new(fakes.FakeRouterConfigurer)
		fakeRoutingApiClient = new(fake_routing_api.FakeClient)
		fakeUaaClient = &testUaaClient.FakeClient{}
		token := &schema.Token{
			AccessToken: "access_token",
			ExpiresIn:   5,
		}
		fakeUaaClient.FetchTokenReturns(token, nil)
		tmpRoutingTable := models.NewRoutingTable(logger)
		routingTable = &tmpRoutingTable
		updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeUaaClient)
	})

	Describe("HandleEvent", func() {
		BeforeEach(func() {
			existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
			existingRoutingTableEntry1 = models.NewRoutingTableEntry(
				[]models.BackendServerInfo{
					models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
					models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
				},
			)
			Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())

			existingRoutingKey2 = models.RoutingKey{Port: externalPort2}
			existingRoutingTableEntry2 = models.NewRoutingTableEntry(
				[]models.BackendServerInfo{
					models.BackendServerInfo{Address: "some-ip-3", Port: 2345},
					models.BackendServerInfo{Address: "some-ip-4", Port: 2345},
				},
			)
			Expect(routingTable.Set(existingRoutingKey2, existingRoutingTableEntry2)).To(BeTrue())

			updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeUaaClient)
		})

		Context("when Upsert event is received", func() {
			Context("when entry does not exist", func() {
				BeforeEach(func() {
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: apimodels.TcpRouteMapping{
							TcpRoute: apimodels.TcpRoute{
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
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-4", Port: 2346},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort4}, expectedRoutingTableEntry)
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
				})
			})

			Context("when entry does exist", func() {
				Context("an existing backend is provided", func() {
					BeforeEach(func() {
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: apimodels.TcpRouteMapping{
								TcpRoute: apimodels.TcpRoute{
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
							TcpRouteMapping: apimodels.TcpRouteMapping{
								TcpRoute: apimodels.TcpRoute{
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
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
								models.BackendServerInfo{Address: "some-ip-5", Port: 1234},
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
						TcpRouteMapping: apimodels.TcpRouteMapping{
							TcpRoute: apimodels.TcpRoute{
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
						existingRoutingKey5 = models.RoutingKey{Port: externalPort5}
						existingRoutingTableEntry5 = models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
							},
						)
						Expect(routingTable.Set(existingRoutingKey5, existingRoutingTableEntry5)).To(BeTrue())
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: apimodels.TcpRouteMapping{
								TcpRoute: apimodels.TcpRoute{
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
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
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
						existingRoutingKey6 = models.RoutingKey{Port: externalPort5}
						existingRoutingTableEntry6 = models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
							},
						)
						Expect(routingTable.Set(existingRoutingKey6, existingRoutingTableEntry6)).To(BeTrue())

						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: apimodels.TcpRouteMapping{
								TcpRoute: apimodels.TcpRoute{
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
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey6, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
					})
				})
			})
		})
	})

	Describe("Sync", func() {

		var (
			doneChannel chan struct{}
			tcpMappings []apimodels.TcpRouteMapping
		)

		invokeSync := func(doneChannel chan struct{}) {
			defer GinkgoRecover()
			updater.Sync()
			close(doneChannel)
		}

		BeforeEach(func() {
			doneChannel = make(chan struct{})
			tcpMappings = []apimodels.TcpRouteMapping{
				apimodels.TcpRouteMapping{
					TcpRoute: apimodels.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    externalPort1,
					},
					HostPort: 61000,
					HostIP:   "some-ip-1",
				},
				apimodels.TcpRouteMapping{
					TcpRoute: apimodels.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    externalPort1,
					},
					HostPort: 61001,
					HostIP:   "some-ip-2",
				},
				apimodels.TcpRouteMapping{
					TcpRoute: apimodels.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    externalPort2,
					},
					HostPort: 60000,
					HostIP:   "some-ip-3",
				},
				apimodels.TcpRouteMapping{
					TcpRoute: apimodels.TcpRoute{
						RouterGroupGuid: routerGroupGuid,
						ExternalPort:    externalPort2,
					},
					HostPort: 60000,
					HostIP:   "some-ip-4",
				},
			}
		})

		Context("when routing api returns tcp route mappings", func() {
			BeforeEach(func() {
				fakeRoutingApiClient.TcpRouteMappingsReturns(tcpMappings, nil)
			})

			It("updates the routing table with that data", func() {
				go invokeSync(doneChannel)
				Eventually(doneChannel).Should(BeClosed())

				Expect(fakeUaaClient.FetchTokenCallCount()).To(Equal(1))
				Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
				Expect(routingTable.Size()).To(Equal(2))
				expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-1", Port: 61000},
						models.BackendServerInfo{Address: "some-ip-2", Port: 61001},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-3", Port: 60000},
						models.BackendServerInfo{Address: "some-ip-4", Port: 60000},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
			})

			Context("when events are received", func() {
				var (
					syncChannel chan struct{}
				)

				BeforeEach(func() {
					syncChannel = make(chan struct{})
					tmpSyncChannel := syncChannel
					fakeRoutingApiClient.TcpRouteMappingsStub = func() ([]apimodels.TcpRouteMapping, error) {
						select {
						case <-tmpSyncChannel:
							return tcpMappings, nil
						}
					}
				})

				It("caches events and then applies the events after it completes syncing", func() {
					go invokeSync(doneChannel)
					Eventually(updater.Syncing).Should(BeTrue())
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: apimodels.TcpRouteMapping{
							TcpRoute: apimodels.TcpRoute{
								RouterGroupGuid: routerGroupGuid,
								ExternalPort:    externalPort1,
							},
							HostPort: 61001,
							HostIP:   "some-ip-2",
						},
						Action: "Delete",
					}
					updater.HandleEvent(tcpEvent)
					Eventually(logger).Should(gbytes.Say("caching-event"))

					close(syncChannel)
					Eventually(updater.Syncing).Should(BeFalse())
					Eventually(doneChannel).Should(BeClosed())
					Eventually(logger).Should(gbytes.Say("applied-cached-events"))

					Expect(fakeUaaClient.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))

					Expect(routingTable.Size()).To(Equal(2))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 61000},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
					expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-3", Port: 60000},
							models.BackendServerInfo{Address: "some-ip-4", Port: 60000},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
				})
			})
		})

		Context("when routing api returns error", func() {
			Context("other than unauthorized", func() {
				BeforeEach(func() {
					fakeRoutingApiClient.TcpRouteMappingsReturns(nil, errors.New("bamboozled"))
					existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
					existingRoutingTableEntry1 = models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
							models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
						},
					)
					Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
				})

				It("uses the cached token and doesn't update its routing table", func() {
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeUaaClient.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))

					Expect(routingTable.Size()).To(Equal(1))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
							models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				})
			})

			Context("unauthorized", func() {
				BeforeEach(func() {
					fakeRoutingApiClient.TcpRouteMappingsReturns(nil, errors.New("unauthorized"))
					existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
					existingRoutingTableEntry1 = models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
							models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
						},
					)
					Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
				})

				It("refresh the token, retries and doesn't update its routing table", func() {
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeUaaClient.FetchTokenCallCount()).To(Equal(2))
					Expect(fakeUaaClient.FetchTokenArgsForCall(0)).To(BeFalse())
					Expect(fakeUaaClient.FetchTokenArgsForCall(1)).To(BeTrue())
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(2))

					Expect(routingTable.Size()).To(Equal(1))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
							models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				})
			})
		})

		Context("when token fetcher returns error", func() {
			BeforeEach(func() {
				fakeUaaClient.FetchTokenReturns(nil, errors.New("no token for you"))
				existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
				existingRoutingTableEntry1 = models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
						models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
					},
				)
				Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
			})

			It("doesn't update its routing table", func() {
				go invokeSync(doneChannel)
				Eventually(doneChannel).Should(BeClosed())

				Expect(fakeUaaClient.FetchTokenCallCount()).To(Equal(1))
				Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(0))

				Expect(routingTable.Size()).To(Equal(1))
				expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-1", Port: 1234},
						models.BackendServerInfo{Address: "some-ip-2", Port: 1234},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
			})
		})

	})
})
