package routing_table_test

import (
	"errors"
	"sync"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
	"github.com/cloudfoundry-incubator/cf-tcp-router/configurer/fakes"
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	"github.com/cloudfoundry-incubator/cf-tcp-router/routing_table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Updater", func() {

	Describe("Update", func() {
		const (
			externalPort1 = uint16(2222)
			externalPort2 = uint16(2223)
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

		verifyBackends := func(port uint16, backends cf_tcp_router.BackendHostInfos) {
			key := models.RoutingKey{port}
			value := routingTable.Get(key)
			Expect(value).NotTo(BeZero())
			Expect(value.Backends).To(HaveLen(len(backends)))
			for _, backend := range backends {
				backendServerInfo := models.BackendServerInfo{backend.Address, backend.Port}
				Expect(value.Backends).To(HaveKey(backendServerInfo))
			}
		}

		verifyPreExistingEntry := func(key models.RoutingKey, entry models.RoutingTableEntry) {
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

		Context("when single entry is being updated", func() {
			Context("that entry does not exist in routing table", func() {
				var (
					externalPort uint16
					backends     cf_tcp_router.BackendHostInfos
				)
				BeforeEach(func() {
					backends = cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
					}
					externalPort = 3333
					mappingRequests := cf_tcp_router.MappingRequests{
						cf_tcp_router.NewMappingRequest(externalPort, backends),
					}
					Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
					err := updater.Update(mappingRequests)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("adds a new entry into routing table", func() {
					verifyBackends(externalPort, backends)
				})

				It("preserves pre-existing entries in routing table", func() {
					verifyPreExistingEntry(existingRoutingKey1, existingRoutingTableEntry1)
					verifyPreExistingEntry(existingRoutingKey2, existingRoutingTableEntry2)
				})

				It("calls configurer to configure the proxy", func() {
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
				})
			})

			Context("that entry exists in routing table", func() {
				Context("that entry is same as existing", func() {
					BeforeEach(func() {
						backends := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-2", 1234),
						}
						mappingRequests := cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(externalPort1, backends),
						}
						Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
						err := updater.Update(mappingRequests)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("doesn't call configurer to configure the proxy", func() {
						Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
					})
				})

				Context("that entry is not same as existing", func() {
					var (
						backends cf_tcp_router.BackendHostInfos
					)
					BeforeEach(func() {
						backends = cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-3", 2345),
						}
						mappingRequests := cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(externalPort1, backends),
						}
						Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
						err := updater.Update(mappingRequests)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("overwrites existing entry with new entry in routing table", func() {
						verifyBackends(externalPort1, backends)
					})

					It("preserves pre-existing entries in routing table", func() {
						verifyPreExistingEntry(existingRoutingKey2, existingRoutingTableEntry2)
					})

					It("calls configurer to configure the proxy", func() {
						Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
					})
				})
			})
		})

		Context("when multiple entries are being updated", func() {
			Context("there are some new entries and some existing entries", func() {
				var (
					backends1     cf_tcp_router.BackendHostInfos
					backends3     cf_tcp_router.BackendHostInfos
					externalPort3 uint16
				)
				BeforeEach(func() {
					backends1 = cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-3", 2345),
					}
					backends3 = cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("some-ip-4", 1234),
						cf_tcp_router.NewBackendHostInfo("some-ip-5", 2345),
					}
					externalPort3 = 3333
					mappingRequests := cf_tcp_router.MappingRequests{
						cf_tcp_router.NewMappingRequest(externalPort1, backends1),
						cf_tcp_router.NewMappingRequest(externalPort3, backends3),
					}
					Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
					err := updater.Update(mappingRequests)
					Expect(err).ShouldNot(HaveOccurred())
				})

				It("adds new entries to routing table", func() {
					verifyBackends(externalPort3, backends3)
				})

				It("updates existing entries in routing table", func() {
					verifyBackends(externalPort1, backends1)
				})

				It("preserves pre-existing entries in routing table", func() {
					verifyPreExistingEntry(existingRoutingKey2, existingRoutingTableEntry2)
				})

				It("calls configurer to configure the proxy", func() {
					Expect(fakeConfigurer.ConfigureCallCount()).To(Equal(1))
				})
			})

			Context("all the entries are existing", func() {
				Context("all entries are same as existing", func() {

					BeforeEach(func() {
						backends1 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
							cf_tcp_router.NewBackendHostInfo("some-ip-2", 1234),
						}
						backends2 := cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("some-ip-3", 2345),
							cf_tcp_router.NewBackendHostInfo("some-ip-4", 2345),
						}
						mappingRequests := cf_tcp_router.MappingRequests{
							cf_tcp_router.NewMappingRequest(externalPort1, backends1),
							cf_tcp_router.NewMappingRequest(externalPort2, backends2),
						}
						Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
						err := updater.Update(mappingRequests)
						Expect(err).ShouldNot(HaveOccurred())
					})

					It("preserves pre-existing entries in routing table", func() {
						verifyPreExistingEntry(existingRoutingKey1, existingRoutingTableEntry1)
						verifyPreExistingEntry(existingRoutingKey2, existingRoutingTableEntry2)
					})

					It("does not call configurer to configure the proxy", func() {
						Expect(fakeConfigurer.ConfigureCallCount()).To(BeZero())
					})
				})
			})
		})

		Context("when multiple mappings with same external port are provided as part of one request", func() {
			var (
				backends cf_tcp_router.BackendHostInfos
			)
			BeforeEach(func() {
				backends1 := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
				}
				backends2 := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
				}
				err := updater.Update(cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(2222, backends1),
					cf_tcp_router.NewMappingRequest(2222, backends2),
				})
				backends = cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-1", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-2", 1235),
					cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
				}
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("combines the backends", func() {
				verifyBackends(externalPort1, backends)
			})
		})

		Context("when invalid maping request is passed", func() {
			It("returns error", func() {
				err := updater.Update(cf_tcp_router.MappingRequests{})
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal(cf_tcp_router.ErrInvalidMapingRequest))
			})
		})

		Context("when configurer returns error", func() {
			BeforeEach(func() {
				fakeConfigurer.ConfigureReturns(errors.New("kaboom"))
			})

			It("returns error", func() {
				backends := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-4", 1235),
				}
				mappingRequests := cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(3333, backends),
				}
				err := updater.Update(mappingRequests)
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when multiple concurrent update requests are made", func() {
			var (
				numCalls int
			)

			BeforeEach(func() {
				mappingRequests := make([]cf_tcp_router.MappingRequests, 0)
				backends1 := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-1", 2345),
					cf_tcp_router.NewBackendHostInfo("some-ip-2", 2345),
				}
				backends2 := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-3", 1234),
					cf_tcp_router.NewBackendHostInfo("some-ip-4", 1234),
				}
				// First two request to update the existing mapping for externalPort1
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(externalPort1, backends1),
				})
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(externalPort1, backends2),
				})

				// Next two request to update the existing mapping for externalPort2
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(externalPort2, backends1),
				})
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(externalPort2, backends2),
				})

				// Next two request to add new mapping for same port
				backends1 = cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-5", 9999),
					cf_tcp_router.NewBackendHostInfo("some-ip-6", 8888),
				}
				backends2 = cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("some-ip-7", 1000),
					cf_tcp_router.NewBackendHostInfo("some-ip-8", 2000),
				}
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(4444, backends1),
				})
				mappingRequests = append(mappingRequests, cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(4444, backends2),
				})
				numCalls = len(mappingRequests)
				wg := sync.WaitGroup{}
				for _, mappingRequest := range mappingRequests {
					wg.Add(1)
					go func(mappingRequest cf_tcp_router.MappingRequests, wg *sync.WaitGroup) {
						defer GinkgoRecover()
						defer wg.Done()
						err := updater.Update(mappingRequest)
						Expect(err).ShouldNot(HaveOccurred())
					}(mappingRequest, &wg)
				}
				wg.Wait()
			})

			It("adds and updates in the order which the requests are processed", func() {
				passedRoutingTable := fakeConfigurer.ConfigureArgsForCall(2)
				Expect(passedRoutingTable).Should(Equal(routingTable))
			})
		})
	})
})
