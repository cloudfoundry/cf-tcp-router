package routing_table_test

import (
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/cf-tcp-router/configurer/fakes"
	"code.cloudfoundry.org/cf-tcp-router/models"
	"code.cloudfoundry.org/cf-tcp-router/routing_table"
	"code.cloudfoundry.org/cf-tcp-router/testutil"
	"code.cloudfoundry.org/clock/fakeclock"
	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/fake_routing_api"
	apimodels "code.cloudfoundry.org/routing-api/models"
	test_uaa_client "code.cloudfoundry.org/routing-api/uaaclient/fakes"
	"golang.org/x/oauth2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Updater", func() {
	const (
		externalPort1   = uint16(2222)
		externalPort2   = uint16(2223)
		externalPort3   = uint16(22233)
		externalPort4   = uint16(2224)
		externalPort5   = uint16(2225)
		externalPort6   = uint16(2226)
		routerGroupGuid = "rtrgrp001"
		defaultTTL      = 60
	)
	var (
		routingTable               *models.RoutingTable
		existingRoutingKey1        models.RoutingKey
		existingRoutingTableEntry1 models.RoutingTableEntry
		existingRoutingKey2        models.RoutingKey
		existingRoutingTableEntry2 models.RoutingTableEntry
		existingRoutingKey3        models.RoutingKey
		existingRoutingTableEntry3 models.RoutingTableEntry
		updater                    routing_table.Updater
		fakeConfigurer             *fakes.FakeRouterConfigurer
		fakeRoutingApiClient       *fake_routing_api.FakeClient
		fakeTokenFetcher           *test_uaa_client.FakeTokenFetcher
		tcpEvent                   routing_api.TcpEvent
		ttl                        int
		modificationTag            apimodels.ModificationTag
		fakeClock                  *fakeclock.FakeClock
		drainWaitDuration          time.Duration
	)

	verifyRoutingTableEntry := func(key models.RoutingKey, entry models.RoutingTableEntry) {
		existingEntry := routingTable.Get(key)
		Expect(existingEntry).NotTo(BeZero())
		testutil.RoutingTableEntryMatches(existingEntry, entry)
	}

	BeforeEach(func() {
		ttl = 60
		modificationTag = apimodels.ModificationTag{Guid: "guid-1", Index: 0}
		fakeConfigurer = new(fakes.FakeRouterConfigurer)
		fakeRoutingApiClient = new(fake_routing_api.FakeClient)
		fakeTokenFetcher = &test_uaa_client.FakeTokenFetcher{}
		token := &oauth2.Token{
			AccessToken: "access_token",
			Expiry:      time.Now().Add(5 * time.Second),
		}
		fakeTokenFetcher.FetchTokenReturns(token, nil)
		tmpRoutingTable := models.NewRoutingTable(logger)
		routingTable = &tmpRoutingTable
		fakeClock = fakeclock.NewFakeClock(time.Now())
	})

	JustBeforeEach(func() {
		updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeTokenFetcher, fakeClock, defaultTTL, drainWaitDuration)
	})

	Describe("HandleEvent", func() {
		BeforeEach(func() {
			existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
			existingRoutingTableEntry1 = models.NewRoutingTableEntry(
				[]models.BackendServerInfo{
					models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
					models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
				},
			)
			Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())

			existingRoutingKey2 = models.RoutingKey{Port: externalPort2}
			existingRoutingTableEntry2 = models.NewRoutingTableEntry(
				[]models.BackendServerInfo{
					models.BackendServerInfo{Address: "some-ip-3", Port: 2345, ModificationTag: modificationTag, TTL: ttl},
					models.BackendServerInfo{Address: "some-ip-4", Port: 2345, ModificationTag: modificationTag, TTL: ttl},
				},
			)
			Expect(routingTable.Set(existingRoutingKey2, existingRoutingTableEntry2)).To(BeTrue())

			existingRoutingKey3 = models.RoutingKey{Port: externalPort3}
			existingRoutingTableEntry3 = models.NewRoutingTableEntry(
				[]models.BackendServerInfo{
					models.BackendServerInfo{Address: "some-ip-5", Port: 2346, ModificationTag: modificationTag, TTL: ttl, TLSPort: 61002, InstanceID: "meow-guid-1"},
					models.BackendServerInfo{Address: "some-ip-6", Port: 2346, ModificationTag: modificationTag, TTL: ttl, TLSPort: 61002, InstanceID: "meow-guid-2"},
				},
			)
			Expect(routingTable.Set(existingRoutingKey3, existingRoutingTableEntry3)).To(BeTrue())
		})

		JustBeforeEach(func() {
			updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeTokenFetcher, fakeClock, defaultTTL, drainWaitDuration)
		})

		Context("when Upsert event is received", func() {
			Context("when entry does not exist", func() {
				BeforeEach(func() {
					mapping := apimodels.NewTcpRouteMapping(
						routerGroupGuid,
						externalPort4,
						"some-ip-4",
						2346,
						61002, // host tls port
						"meow-instance-guid",
						nil,
						ttl,
						modificationTag,
					)
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: mapping,
						Action:          "Upsert",
					}
				})

				It("inserts handle the event and inserts the new entry", func() {
					err := updater.HandleEvent(tcpEvent)
					Expect(err).NotTo(HaveOccurred())
					tlsPort := 61002
					expectedRoutingTableEntry := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-4", Port: 2346, TTL: ttl, ModificationTag: modificationTag, TLSPort: tlsPort, InstanceID: "meow-instance-guid"},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort4}, expectedRoutingTableEntry)
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
				})
			})

			Context("when entry does exist", func() {
				var (
					newModificationTag apimodels.ModificationTag
					newTTL             int
				)
				BeforeEach(func() {
					newModificationTag = apimodels.ModificationTag{Guid: "guid-1", Index: 1}
					newTTL = 100
				})

				Context("an existing backend is provided", func() {
					BeforeEach(func() {
						mapping := apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-1",
							1234,
							0,
							"",
							nil,
							newTTL,
							newModificationTag,
						)
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: mapping,
							Action:          "Upsert",
						}
					})

					It("does not call configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						existingRoutingTableEntry := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: newModificationTag, TTL: newTTL},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey1, existingRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
					})

					Context("when TLSPort and InstanceID are set", func() {
						BeforeEach(func() {
							mapping := apimodels.NewTcpRouteMapping(
								routerGroupGuid,
								externalPort3,
								"some-ip-5",
								2346,
								61002,
								"meow-guid-1",
								nil,
								newTTL,
								newModificationTag,
							)
							tcpEvent = routing_api.TcpEvent{
								TcpRouteMapping: mapping,
								Action:          "Upsert",
							}
						})

						It("does not call configurer", func() {
							err := updater.HandleEvent(tcpEvent)
							Expect(err).NotTo(HaveOccurred())
							existingRoutingTableEntry := models.NewRoutingTableEntry(
								[]models.BackendServerInfo{
									models.BackendServerInfo{Address: "some-ip-5", Port: 2346, ModificationTag: newModificationTag, TTL: newTTL, TLSPort: 61002, InstanceID: "meow-guid-1"},
									models.BackendServerInfo{Address: "some-ip-6", Port: 2346, ModificationTag: modificationTag, TTL: ttl, TLSPort: 61002, InstanceID: "meow-guid-2"},
								},
							)
							verifyRoutingTableEntry(existingRoutingKey3, existingRoutingTableEntry)
							Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
						})

					})
				})

				Context("and a new backend is provided", func() {
					BeforeEach(func() {
						mapping := apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-5",
							1234,
							0,
							"",
							nil,
							ttl,
							newModificationTag,
						)
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: mapping,
							Action:          "Upsert",
						}
					})

					It("adds backend to existing entry and calls configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-5", Port: 1234, ModificationTag: newModificationTag, TTL: ttl},
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
			var (
				newModificationTag apimodels.ModificationTag
			)
			BeforeEach(func() {
				newModificationTag = apimodels.ModificationTag{Guid: "guid-1", Index: 1}
			})

			Context("when entry does not exist", func() {
				BeforeEach(func() {
					mapping := apimodels.NewTcpRouteMapping(
						routerGroupGuid,
						externalPort4,
						"some-ip-4",
						2346,
						0,
						"",
						nil,
						ttl,
						newModificationTag,
					)
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: mapping,
						Action:          "Delete",
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
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							},
						)
						Expect(routingTable.Set(existingRoutingKey5, existingRoutingTableEntry5)).To(BeTrue())
						mapping := apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort5,
							"some-ip-1",
							1234,
							0,
							"",
							nil,
							ttl,
							modificationTag,
						)
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: mapping,
							Action:          "Delete",
						}
					})

					It("deletes backend from entry and calls configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
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
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							},
						)
						Expect(routingTable.Set(existingRoutingKey6, existingRoutingTableEntry6)).To(BeTrue())

						mapping := apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort5,
							"some-ip-5",
							1234,
							0,
							"",
							nil,
							ttl,
							newModificationTag,
						)
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: mapping,
							Action:          "Delete",
						}
					})

					It("does not call configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey6, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
					})
				})

				Context("TLSPort and InstanceID are provided", func() {
					var (
						existingRoutingKey5        models.RoutingKey
						existingRoutingTableEntry5 models.RoutingTableEntry
					)
					BeforeEach(func() {
						existingRoutingKey5 = models.RoutingKey{Port: externalPort5}
						existingRoutingTableEntry5 = models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, TLSPort: 60012, InstanceID: "another-meow-for-the-back", ModificationTag: modificationTag, TTL: ttl},
								models.BackendServerInfo{Address: "some-ip-2", Port: 1234, TLSPort: 60013, InstanceID: "griffin", ModificationTag: modificationTag, TTL: ttl},
							},
						)
						Expect(routingTable.Set(existingRoutingKey5, existingRoutingTableEntry5)).To(BeTrue())
						mapping := apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort5,
							"some-ip-2",
							1234,
							60013,
							"griffin",
							nil,
							ttl,
							modificationTag,
						)
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: mapping,
							Action:          "Delete",
						}
					})

					It("deletes backend from entry and calls configurer", func() {
						err := updater.HandleEvent(tcpEvent)
						Expect(err).NotTo(HaveOccurred())
						expectedRoutingTableEntry := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234, TLSPort: 60012, InstanceID: "another-meow-for-the-back", ModificationTag: modificationTag, TTL: ttl},
							},
						)
						verifyRoutingTableEntry(existingRoutingKey5, expectedRoutingTableEntry)
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
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
				apimodels.NewTcpRouteMapping(
					routerGroupGuid,
					externalPort1,
					"some-ip-1",
					61000,
					0,
					"",
					nil,
					ttl,
					modificationTag,
				),
				apimodels.NewTcpRouteMapping(
					routerGroupGuid,
					externalPort1,
					"some-ip-2",
					61001,
					0,
					"",
					nil,
					ttl,
					modificationTag,
				),
				apimodels.NewTcpRouteMapping(
					routerGroupGuid,
					externalPort2,
					"some-ip-3",
					60000,
					0,
					"",
					nil,
					ttl,
					modificationTag,
				),
				apimodels.NewTcpRouteMapping(
					routerGroupGuid,
					externalPort2,
					"some-ip-4",
					60000,
					0,
					"",
					nil,
					ttl,
					modificationTag,
				),
			}
		})

		Context("when routing api returns tcp route mappings", func() {
			BeforeEach(func() {
				fakeRoutingApiClient.TcpRouteMappingsReturns(tcpMappings, nil)
			})

			It("updates the routing table with that data", func() {
				Expect(routingTable.Size()).To(Equal(0))
				go invokeSync(doneChannel)
				Eventually(doneChannel).Should(BeClosed())

				Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
				Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
				Expect(routingTable.Size()).To(Equal(2))
				expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-1", Port: 61000, ModificationTag: modificationTag, TTL: ttl},
						models.BackendServerInfo{Address: "some-ip-2", Port: 61001, ModificationTag: modificationTag, TTL: ttl},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						models.BackendServerInfo{Address: "some-ip-3", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
						models.BackendServerInfo{Address: "some-ip-4", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
				Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
			})

			Context("when there are no changes to the routing table", func() {
				BeforeEach(func() {
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-1", Port: 61000, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 61001, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					routingTable.Set(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)

					expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-3", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-4", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					routingTable.Set(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
				})

				It("does not call the configurer", func() {
					Expect(routingTable.Size()).To(Equal(2))
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
					Expect(routingTable.Size()).To(Equal(2))
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))
				})
			})

			Context("when things have been deleted from the table", func() {
				BeforeEach(func() {
					tcpMappings = []apimodels.TcpRouteMapping{
						apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-1",
							61000,
							0,
							"",
							nil,
							ttl,
							modificationTag,
						),
						apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-2",
							61001,
							0,
							"",
							nil,
							ttl,
							modificationTag,
						),
					}

					fakeRoutingApiClient.TcpRouteMappingsReturns(tcpMappings, nil)

					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-1", Port: 61000, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 61001, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					routingTable.Set(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)

					expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-3", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-4", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					routingTable.Set(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
				})

				It("calls the configurer", func() {
					Expect(routingTable.Size()).To(Equal(2))
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
					Expect(routingTable.Size()).To(Equal(1))
				})
			})

			Context("when events are received", func() {
				var (
					syncChannel chan struct{}
				)

				BeforeEach(func() {
					syncChannel = make(chan struct{})
					tmpSyncChannel := syncChannel
					fakeRoutingApiClient.TcpRouteMappingsStub = func() ([]apimodels.TcpRouteMapping, error) {
						<-tmpSyncChannel
						return tcpMappings, nil
					}
				})

				Context("but there are no changes in the bulk sync", func() {
					BeforeEach(func() {
						syncChannel = make(chan struct{})
						tmpSyncChannel := syncChannel
						tcpMappings := make([]apimodels.TcpRouteMapping, 0)
						fakeRoutingApiClient.TcpRouteMappingsStub = func() ([]apimodels.TcpRouteMapping, error) {
							<-tmpSyncChannel
							return tcpMappings, nil
						}
					})
					It("still applies the cached event", func() {
						go invokeSync(doneChannel)
						Eventually(updater.Syncing).Should(BeTrue())
						tcpEvent = routing_api.TcpEvent{
							TcpRouteMapping: apimodels.NewTcpRouteMapping(
								routerGroupGuid,
								externalPort1,
								"some-ip-2",
								61001,
								0,
								"",
								nil,
								0,
								modificationTag,
							),
							Action: "Upsert",
						}
						_ = updater.HandleEvent(tcpEvent)
						Eventually(logger).Should(gbytes.Say("caching-event"))

						close(syncChannel)
						Eventually(updater.Syncing).Should(BeFalse())
						Eventually(doneChannel).Should(BeClosed())
						Eventually(logger).Should(gbytes.Say("applied-cached-events"))

						Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
						Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))

						Expect(routingTable.Size()).To(Equal(1))
						expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
							[]models.BackendServerInfo{
								models.BackendServerInfo{Address: "some-ip-2", Port: 61001, ModificationTag: modificationTag, TTL: 0},
							},
						)
						verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
					})
				})

				It("caches events and then applies the events after it completes syncing", func() {
					Expect(routingTable.Size()).To(Equal(0))
					go invokeSync(doneChannel)
					Eventually(updater.Syncing).Should(BeTrue())
					tcpEvent = routing_api.TcpEvent{
						TcpRouteMapping: apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-2",
							61001,
							0,
							"",
							nil,
							0,
							modificationTag,
						),
						Action: "Delete",
					}
					updater.HandleEvent(tcpEvent)
					Eventually(logger).Should(gbytes.Say("caching-event"))

					close(syncChannel)
					Eventually(updater.Syncing).Should(BeFalse())
					Eventually(doneChannel).Should(BeClosed())
					Eventually(logger).Should(gbytes.Say("applied-cached-events"))

					Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))

					Expect(routingTable.Size()).To(Equal(2))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-1", Port: 61000, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
					expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							models.BackendServerInfo{Address: "some-ip-3", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
							models.BackendServerInfo{Address: "some-ip-4", Port: 60000, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
				})
				Context("when cached events come in during sync", func() {
					var originalEntries map[models.RoutingKey]models.RoutingTableEntry
					BeforeEach(func() {
						// Prepopulate routing table
						originalEntries = map[models.RoutingKey]models.RoutingTableEntry{
							{Port: 2222, SniHostname: ""}: {
								Backends: map[models.BackendServerKey]models.BackendServerDetails{
									{Address: "some-ip-1", Port: 61000}: {
										ModificationTag: apimodels.ModificationTag{Guid: "guid-1", Index: 0},
										TTL:             60,
									},
									{Address: "some-ip-2", Port: 61001}: {
										ModificationTag: apimodels.ModificationTag{Guid: "guid-1", Index: 0},
										TTL:             60,
									},
								},
							},
							{Port: 2223, SniHostname: ""}: {
								Backends: map[models.BackendServerKey]models.BackendServerDetails{
									{Address: "some-ip-3", Port: 60000}: {
										ModificationTag: apimodels.ModificationTag{Guid: "guid-1", Index: 0},
										TTL:             60,
									},
									{Address: "some-ip-4", Port: 60000}: {
										ModificationTag: apimodels.ModificationTag{Guid: "guid-1", Index: 0},
										TTL:             60,
									},
								},
							},
						}
						routingTable.Entries = map[models.RoutingKey]models.RoutingTableEntry{}
						for k, v := range originalEntries {
							routingTable.Entries[k] = v
						}
					})
					Context("and the events don't constitute substantive changes", func() {
						It("does not reload haproxy", func() {
							// ensure the change hasn't made it to the table
							targetBackend := routingTable.Entries[models.RoutingKey{Port: externalPort1}].Backends[models.BackendServerKey{Address: "some-ip-22", Port: 61001}]
							Expect(targetBackend.ModificationTag.Index).To(Equal(uint32(0)))

							// start syncing
							go invokeSync(doneChannel)
							Eventually(updater.Syncing).Should(BeTrue())
							// submit an event while syncing
							newModificationTag := modificationTag
							newModificationTag.Increment()
							tcpEvent = routing_api.TcpEvent{
								TcpRouteMapping: apimodels.NewTcpRouteMapping(
									routerGroupGuid,
									externalPort1,
									"some-ip-1",
									61000,
									0,
									"",
									nil,
									22,
									apimodels.ModificationTag{Guid: "guid-1", Index: 1},
								),
								Action: "Upsert",
							}
							updater.HandleEvent(tcpEvent)
							Eventually(logger).Should(gbytes.Say("caching-event"))

							close(syncChannel)
							Eventually(updater.Syncing).Should(BeFalse())
							Eventually(doneChannel).Should(BeClosed())

							targetBackend = routingTable.Entries[models.RoutingKey{Port: externalPort1}].Backends[models.BackendServerKey{Address: "some-ip-1", Port: 61000}]
							Expect(targetBackend.ModificationTag.Index).To(Equal(uint32(1))) // ensure the routing table took the change
							Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(0))         // ensure it didn't reload haproxy
						})
					})
					Context("and the events modify more than just modification tags", func() {
						It("reloads haproxy", func() {
							// ensure the change hasn't made it to the table
							targetBackend := routingTable.Entries[models.RoutingKey{Port: externalPort1}].Backends[models.BackendServerKey{Address: "some-ip-22", Port: 61001}]
							Expect(targetBackend).To(Equal(models.BackendServerDetails{}))

							// start syncing
							newModificationTag := modificationTag
							newModificationTag.Increment()
							go invokeSync(doneChannel)
							Eventually(updater.Syncing).Should(BeTrue())
							// submit an event while syncing
							tcpEvent = routing_api.TcpEvent{
								TcpRouteMapping: apimodels.NewTcpRouteMapping(
									routerGroupGuid,
									externalPort1,
									"some-ip-22",
									61001,
									0,
									"",
									nil,
									22,
									newModificationTag,
								),
								Action: "Upsert",
							}
							updater.HandleEvent(tcpEvent)
							Eventually(logger).Should(gbytes.Say("caching-event"))

							close(syncChannel)
							Eventually(updater.Syncing).Should(BeFalse())
							Eventually(doneChannel).Should(BeClosed())

							targetBackend = routingTable.Entries[models.RoutingKey{Port: externalPort1}].Backends[models.BackendServerKey{Address: "some-ip-22", Port: 61001}]
							Expect(targetBackend.ModificationTag.Index).To(Equal(uint32(1))) // ensure the routing table took the change
							Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))         // ensure it did reload haproxy
						})
					})
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
							{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
				})

				It("uses the cached token and doesn't update its routing table", func() {
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(1))

					Expect(routingTable.Size()).To(Equal(1))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
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
							{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
				})

				It("refresh the token, retries and doesn't update its routing table", func() {
					go invokeSync(doneChannel)
					Eventually(doneChannel).Should(BeClosed())

					Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(2))
					_, forceUpdate := fakeTokenFetcher.FetchTokenArgsForCall(0)
					Expect(forceUpdate).To(BeFalse())
					_, forceUpdate = fakeTokenFetcher.FetchTokenArgsForCall(1)
					Expect(forceUpdate).To(BeTrue())
					Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(2))

					Expect(routingTable.Size()).To(Equal(1))
					expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
						[]models.BackendServerInfo{
							{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
							{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
						},
					)
					verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				})
			})
		})

		Context("when token fetcher returns error", func() {
			BeforeEach(func() {
				fakeTokenFetcher.FetchTokenReturns(nil, errors.New("no token for you"))
				existingRoutingKey1 = models.RoutingKey{Port: externalPort1}
				existingRoutingTableEntry1 = models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
						{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
					},
				)
				Expect(routingTable.Set(existingRoutingKey1, existingRoutingTableEntry1)).To(BeTrue())
			})

			It("doesn't update its routing table", func() {
				go invokeSync(doneChannel)
				Eventually(doneChannel).Should(BeClosed())

				Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(1))
				Expect(fakeRoutingApiClient.TcpRouteMappingsCallCount()).To(Equal(0))

				Expect(routingTable.Size()).To(Equal(1))
				expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
						{Address: "some-ip-2", Port: 1234, ModificationTag: modificationTag, TTL: ttl},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
			})
		})

	})

	Describe("Prune", func() {
		BeforeEach(func() {
			routingKey1 := models.RoutingKey{Port: externalPort1}
			backendServerKey := models.BackendServerKey{Address: "some-ip-1", Port: 1234}
			backendServerDetails := models.BackendServerDetails{ModificationTag: modificationTag, UpdatedTime: time.Now().Add(-50 * time.Second)}
			backendServerKey2 := models.BackendServerKey{Address: "some-ip-2", Port: 1235}
			backendServerDetails2 := models.BackendServerDetails{ModificationTag: modificationTag, UpdatedTime: time.Now().Add(-50 * time.Second)}
			backends := map[models.BackendServerKey]models.BackendServerDetails{
				backendServerKey:  backendServerDetails,
				backendServerKey2: backendServerDetails2,
			}
			routingTableEntry := models.RoutingTableEntry{Backends: backends}
			updated := routingTable.Set(routingKey1, routingTableEntry)
			Expect(updated).To(BeTrue())

			routingKey2 := models.RoutingKey{Port: externalPort2}
			backendServerKey = models.BackendServerKey{Address: "some-ip-3", Port: 1234}
			backendServerDetails = models.BackendServerDetails{ModificationTag: modificationTag, UpdatedTime: time.Now().Add(-10 * time.Second)}
			backendServerKey2 = models.BackendServerKey{Address: "some-ip-4", Port: 1235}
			backendServerDetails2 = models.BackendServerDetails{ModificationTag: modificationTag, UpdatedTime: time.Now()}
			backends = map[models.BackendServerKey]models.BackendServerDetails{
				backendServerKey:  backendServerDetails,
				backendServerKey2: backendServerDetails2,
			}
			routingTableEntry = models.RoutingTableEntry{Backends: backends}
			updated = routingTable.Set(routingKey2, routingTableEntry)
			Expect(updated).To(BeTrue())
		})

		JustBeforeEach(func() {
			updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeTokenFetcher, fakeClock, defaultTTL, drainWaitDuration)
		})

		Context("when none of the routes are stale", func() {
			It("doesn't prune any routes", func() {
				updater.PruneStaleRoutes()
				Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(0))
				Expect(routingTable.Size()).To(Equal(2))
				expectedRoutingTableEntry1 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						{Address: "some-ip-1", Port: 1234, ModificationTag: modificationTag},
						{Address: "some-ip-2", Port: 1235, ModificationTag: modificationTag},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort1}, expectedRoutingTableEntry1)
				expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						{Address: "some-ip-3", Port: 1234, ModificationTag: modificationTag},
						{Address: "some-ip-4", Port: 1235, ModificationTag: modificationTag},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
			})
		})

		Context("when some routes are stale", func() {
			BeforeEach(func() {
				fakeClock.IncrementBySeconds(65)
			})

			JustBeforeEach(func() {
				updater = routing_table.NewUpdater(logger, routingTable, fakeConfigurer, fakeRoutingApiClient, fakeTokenFetcher, fakeClock, 40, drainWaitDuration)
			})

			It("prunes those routes", func() {
				updater.PruneStaleRoutes()
				Expect(fakeTokenFetcher.FetchTokenCallCount()).To(Equal(0))
				Expect(routingTable.Size()).To(Equal(1))
				expectedRoutingTableEntry2 := models.NewRoutingTableEntry(
					[]models.BackendServerInfo{
						{Address: "some-ip-3", Port: 1234, ModificationTag: modificationTag},
						{Address: "some-ip-4", Port: 1235, ModificationTag: modificationTag},
					},
				)
				verifyRoutingTableEntry(models.RoutingKey{Port: externalPort2}, expectedRoutingTableEntry2)
			})
		})
	})

	Describe("Drain", func() {
		Context("when there is no sync going on", func() {
			It("calls configure", func() {
				err := updater.Drain()
				Expect(err).NotTo(HaveOccurred())
				Eventually(fakeConfigurer.ConfigureCallCount).Should(Equal(1))
			})
			Context("when drain was already called", func() {
				JustBeforeEach(func() {
					err := updater.Drain()
					Expect(err).NotTo(HaveOccurred())
				})
				It("should only drain once", func() {
					err := updater.Drain()
					Expect(err).NotTo(HaveOccurred())
					Eventually(logger).Should(gbytes.Say("drain-already-in-progress"))
					Consistently(fakeConfigurer.ConfigureCallCount).Should(Equal(1))
				})
			})
		})
		Context("when there is a sync going on", func() {
			var (
				syncChannel chan struct{}
				doneChannel chan struct{}
			)

			invokeSync := func(doneChannel chan struct{}) {
				defer GinkgoRecover()
				updater.Sync()
				close(doneChannel)
			}
			BeforeEach(func() {
				syncChannel = make(chan struct{})
				doneChannel = make(chan struct{})
				tmpSyncChannel := syncChannel
				tcpMappings := make([]apimodels.TcpRouteMapping, 0)
				fakeRoutingApiClient.TcpRouteMappingsStub = func() ([]apimodels.TcpRouteMapping, error) {
					<-tmpSyncChannel
					return tcpMappings, nil
				}
			})

			It("waits for the sync to finish, then calls configure", func() {
				go invokeSync(doneChannel)
				Eventually(updater.Syncing).Should(BeTrue())
				go updater.Drain()
				Eventually(logger).Should(gbytes.Say("waiting-for-sync-to-finish-before-starting-drain"))
				Consistently(fakeConfigurer.ConfigureCallCount()).Should(Equal(0))
				close(syncChannel)
				Eventually(updater.Syncing).Should(BeFalse())
				Eventually(logger).Should(gbytes.Say("drain-started"))
				Eventually(fakeConfigurer.ConfigureCallCount()).Should(Equal(1))
			})
		})

		It("tells the configurer to reconfigure in drain mode", func() {
			err := updater.Drain()
			Expect(err).NotTo(HaveOccurred())
			_, drain := fakeConfigurer.ConfigureArgsForCall(0)
			Expect(drain).To(BeTrue())
		})

		Context("when Configure() returns an error", func() {
			BeforeEach(func() {
				fakeConfigurer.ConfigureReturns(fmt.Errorf("meow"))
			})

			It("propagates the error", func() {
				err := updater.Drain()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("meow"))
			})
		})

		Context("when the drain wait is larger than 0", func() {
			BeforeEach(func() {
				drainWaitDuration = 2 * time.Second
			})

			It("waits for the drain wait before returning", func() {
				go updater.Drain()
				Eventually(logger).Should(gbytes.Say("starting-drain-wait"))
				Consistently(logger, 1*time.Second).ShouldNot(gbytes.Say("finished-drain-wait"))
				Eventually(logger, 2*time.Second).Should(gbytes.Say("finished-drain-wait"))
			})
		})

		Context("when Sync is called after drain", func() {
			BeforeEach(func() {
				tcpMappings := []apimodels.TcpRouteMapping{
					apimodels.NewTcpRouteMapping(
						routerGroupGuid,
						externalPort1,
						"some-ip-1",
						61000,
						61001,
						"instance-id",
						nil,
						ttl,
						modificationTag,
					),
				}
				fakeRoutingApiClient.TcpRouteMappingsReturns(tcpMappings, nil)

			})
			It("it calls Configure with forceHealthCheckToFail == true", func() {
				err := updater.Drain()
				Expect(err).NotTo(HaveOccurred())
				_, drain := fakeConfigurer.ConfigureArgsForCall(0)
				Expect(drain).To(BeTrue())
				updater.Sync()
				_, drain = fakeConfigurer.ConfigureArgsForCall(1)
				Expect(drain).To(BeTrue())
			})
		})

		Context("when handleEvent is called after drain", func() {
			Context("when the action is delete", func() {
				BeforeEach(func() {
					tcpMappings := []apimodels.TcpRouteMapping{
						apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-1",
							61000,
							61001,
							"instance-id",
							nil,
							ttl,
							modificationTag,
						),
					}
					fakeRoutingApiClient.TcpRouteMappingsReturns(tcpMappings, nil)

				})
				It("it calls Configure with forceHealthCheckToFail == true", func() {
					// add item to the routing table
					updater.Sync()
					Eventually(routingTable.Size).Should(Equal(1))
					_, drain := fakeConfigurer.ConfigureArgsForCall(0)
					Expect(drain).To(BeFalse())

					// call drain
					err := updater.Drain()
					Expect(err).NotTo(HaveOccurred())
					_, drain = fakeConfigurer.ConfigureArgsForCall(1)
					Expect(drain).To(BeTrue())

					// delete item from the routing table
					err = updater.HandleEvent(routing_api.TcpEvent{
						Action: "Delete",
						TcpRouteMapping: apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-1",
							61000,
							61001,
							"instance-id",
							nil,
							ttl,
							modificationTag,
						),
					})
					Expect(err).NotTo(HaveOccurred())
					Eventually(routingTable.Size).Should(Equal(0))

					// see that forceHealthCheckToFail is true
					_, drain = fakeConfigurer.ConfigureArgsForCall(2)
					Expect(drain).To(BeTrue())
				})
			})

			Context("when the action is upsert", func() {
				It("it calls Configure with forceHealthCheckToFail == true", func() {
					err := updater.Drain()
					Expect(err).NotTo(HaveOccurred())
					_, drain := fakeConfigurer.ConfigureArgsForCall(0)
					Expect(drain).To(BeTrue())
					err = updater.HandleEvent(routing_api.TcpEvent{
						Action: "Upsert",
						TcpRouteMapping: apimodels.NewTcpRouteMapping(
							routerGroupGuid,
							externalPort1,
							"some-ip-1",
							61000,
							61001,
							"instance-id",
							nil,
							ttl,
							modificationTag,
						),
					})
					Expect(err).NotTo(HaveOccurred())
					_, drain = fakeConfigurer.ConfigureArgsForCall(1)
					Expect(drain).To(BeTrue())
				})
			})
		})
	})
})
