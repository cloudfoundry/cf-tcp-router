package models_test

import (
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"
	routing_api_models "github.com/cloudfoundry-incubator/routing-api/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RoutingTable", func() {
	var (
		backendServerKey models.BackendServerKey
		routingTable     models.RoutingTable
		modificationTag  routing_api_models.ModificationTag
	)

	BeforeEach(func() {
		routingTable = models.NewRoutingTable()
		modificationTag = routing_api_models.ModificationTag{Guid: "abc", Index: 0}
	})

	Describe("Set", func() {
		var (
			routingKey           models.RoutingKey
			routingTableEntry    models.RoutingTableEntry
			backendServerDetails models.BackendServerDetails
		)

		BeforeEach(func() {
			routingKey = models.RoutingKey{Port: 12}
			backendServerDetails = models.BackendServerDetails{ModificationTag: modificationTag}
			backends := map[models.BackendServerKey]models.BackendServerDetails{
				backendServerKey: backendServerDetails,
			}
			routingTableEntry = models.RoutingTableEntry{Backends: backends}
		})

		Context("when a new entry is added", func() {
			It("adds the entry", func() {
				ok := routingTable.Set(routingKey, routingTableEntry)
				Expect(ok).To(BeTrue())
				Expect(routingTable.Get(routingKey)).To(Equal(routingTableEntry))
				Expect(routingTable.Size()).To(Equal(1))
			})
		})

		Context("when setting pre-existing routing key", func() {
			var (
				existingRoutingTableEntry models.RoutingTableEntry
				newBackendServerKey       models.BackendServerKey
			)

			BeforeEach(func() {
				newBackendServerKey = models.BackendServerKey{
					Address: "some-ip-2",
					Port:    1234,
				}
				existingRoutingTableEntry = models.RoutingTableEntry{
					Backends: map[models.BackendServerKey]models.BackendServerDetails{
						backendServerKey:    backendServerDetails,
						newBackendServerKey: models.BackendServerDetails{ModificationTag: modificationTag},
					},
				}
				ok := routingTable.Set(routingKey, existingRoutingTableEntry)
				Expect(ok).To(BeTrue())
				Expect(routingTable.Size()).To(Equal(1))
			})

			Context("with different value", func() {
				verifyChangedValue := func(routingTableEntry models.RoutingTableEntry) {
					ok := routingTable.Set(routingKey, routingTableEntry)
					Expect(ok).To(BeTrue())
					Expect(routingTable.Get(routingKey)).Should(Equal(routingTableEntry))
				}

				Context("when number of backends are different", func() {
					It("overwrites the value", func() {
						routingTableEntry := models.RoutingTableEntry{
							Backends: map[models.BackendServerKey]models.BackendServerDetails{
								models.BackendServerKey{
									Address: "some-ip-1",
									Port:    1234,
								}: models.BackendServerDetails{ModificationTag: modificationTag},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})

				Context("when at least one backend server info is different", func() {
					It("overwrites the value", func() {
						routingTableEntry := models.RoutingTableEntry{
							Backends: map[models.BackendServerKey]models.BackendServerDetails{
								models.BackendServerKey{Address: "some-ip-1", Port: 1234}: models.BackendServerDetails{ModificationTag: modificationTag},
								models.BackendServerKey{Address: "some-ip-2", Port: 2345}: models.BackendServerDetails{ModificationTag: modificationTag},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})

				Context("when all backend servers info are different", func() {
					It("overwrites the value", func() {
						routingTableEntry := models.RoutingTableEntry{
							Backends: map[models.BackendServerKey]models.BackendServerDetails{
								models.BackendServerKey{Address: "some-ip-1", Port: 3456}: models.BackendServerDetails{ModificationTag: modificationTag},
								models.BackendServerKey{Address: "some-ip-2", Port: 2345}: models.BackendServerDetails{ModificationTag: modificationTag},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})
			})

			Context("with same value", func() {
				It("returns false", func() {
					routingTableEntry := models.RoutingTableEntry{
						Backends: map[models.BackendServerKey]models.BackendServerDetails{
							backendServerKey:    models.BackendServerDetails{ModificationTag: modificationTag},
							newBackendServerKey: models.BackendServerDetails{ModificationTag: modificationTag},
						},
					}
					ok := routingTable.Set(routingKey, routingTableEntry)
					Expect(ok).To(BeFalse())
					Expect(routingTable.Get(routingKey)).Should(Equal(existingRoutingTableEntry))
				})
			})
		})
	})

	Describe("UpsertBackendServerKey", func() {
		var (
			routingKey models.RoutingKey
		)

		BeforeEach(func() {
			routingKey = models.RoutingKey{Port: 12}
			routingTable = models.NewRoutingTable()
			modificationTag = routing_api_models.ModificationTag{Guid: "abc", Index: 0}
		})

		Context("when the routing key does not exist", func() {
			var (
				routingTableEntry models.RoutingTableEntry
				backendServerInfo models.BackendServerInfo
			)

			BeforeEach(func() {
				backendServerInfo = createBackendServerInfo("some-ip", 1234, modificationTag)
				routingTableEntry = models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo})
			})

			It("inserts the routing key with its backends", func() {
				updated := routingTable.UpsertBackendServerKey(routingKey, backendServerInfo)
				Expect(updated).To(BeTrue())
				Expect(routingTable.Size()).To(Equal(1))
				Expect(routingTable.Get(routingKey)).Should(Equal(routingTableEntry))
			})
		})

		Context("when the routing key does exist", func() {
			var backendServerInfo models.BackendServerInfo

			BeforeEach(func() {
				backendServerInfo = createBackendServerInfo("some-ip", 1234, modificationTag)
				existingRoutingTableEntry := models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo})
				updated := routingTable.Set(routingKey, existingRoutingTableEntry)
				Expect(updated).To(BeTrue())
			})

			Context("and no change in the backends are provided", func() {
				It("does not update the routing entry", func() {
					sameBackendServerInfo := createBackendServerInfo("some-ip", 1234, modificationTag)
					updated := routingTable.UpsertBackendServerKey(routingKey, sameBackendServerInfo)
					Expect(updated).To(BeFalse())
				})
			})

			Context("and a new backend is provided", func() {
				It("updates the routing entry's backends", func() {
					anotherModificationTag := routing_api_models.ModificationTag{Guid: "def", Index: 0}
					differentBackendServerInfo := createBackendServerInfo("some-other-ip", 1234, anotherModificationTag)
					expectedRoutingTableEntry := models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo, differentBackendServerInfo})
					updated := routingTable.UpsertBackendServerKey(routingKey, differentBackendServerInfo)
					Expect(updated).To(BeTrue())
					Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
				})
			})

			Context("when current entry is succeeded by new entry", func() {
				BeforeEach(func() {
					modificationTag.Increment()
				})

				It("should update routing table with new entry", func() {
					newBackendServerInfo := createBackendServerInfo("some-ip", 1234, modificationTag)
					updated := routingTable.UpsertBackendServerKey(routingKey, newBackendServerInfo)
					expectedRoutingTableEntry := models.NewRoutingTableEntry([]models.BackendServerInfo{newBackendServerInfo})
					Expect(updated).To(BeTrue())
					Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
				})
			})
		})
	})

	Describe("DeleteBackendServerKey", func() {
		var (
			routingKey                models.RoutingKey
			existingRoutingTableEntry models.RoutingTableEntry
			backendServerInfo1        models.BackendServerInfo
			backendServerInfo2        models.BackendServerInfo
		)
		BeforeEach(func() {
			routingKey = models.RoutingKey{Port: 12}
			backendServerInfo1 = createBackendServerInfo("some-ip", 1234, modificationTag)
		})

		Context("when the routing key does not exist", func() {
			It("it does not causes any changes or errors", func() {
				updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
				Expect(updated).To(BeFalse())
			})

		})

		Context("when the routing key does exist", func() {
			BeforeEach(func() {
				backendServerInfo2 = createBackendServerInfo("some-other-ip", 1235, modificationTag)
				existingRoutingTableEntry = models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo1, backendServerInfo2})
				updated := routingTable.Set(routingKey, existingRoutingTableEntry)
				Expect(updated).To(BeTrue())
			})

			Context("and the backend does not exist ", func() {
				It("does not causes any changes or errors", func() {
					backendServerInfo1 = createBackendServerInfo("some-missing-ip", 1236, modificationTag)
					ok := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
					Expect(ok).To(BeFalse())
					Expect(routingTable.Get(routingKey)).Should(Equal(existingRoutingTableEntry))
				})
			})

			Context("and the backend does exist", func() {
				It("deletes the backend", func() {
					updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
					Expect(updated).To(BeTrue())
					expectedRoutingTableEntry := models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo2})
					Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
				})

				Context("when a modification tag has the same guid but index is greater then current index", func() {
					BeforeEach(func() {
						modificationTag.Increment()
					})

					It("deletes the backend", func() {
						updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
						Expect(updated).To(BeTrue())
						expectedRoutingTableEntry := models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo2})
						Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
					})
				})

				Context("when a modification tag is not current or newer", func() {
					var expectedRoutingTableEntry models.RoutingTableEntry

					BeforeEach(func() {
						backendServerInfo1.ModificationTag.Increment()
						expectedRoutingTableEntry = models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo1, backendServerInfo2})
						updated := routingTable.UpsertBackendServerKey(routingKey, backendServerInfo1)
						Expect(updated).To(BeTrue())
					})

					It("doesn't delete the backend", func() {
						backendServerInfo1.ModificationTag = routing_api_models.ModificationTag{Guid: "abc"}
						updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
						Expect(updated).To(BeFalse())
						Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
					})
				})

				Context("when a modification tag has different guid", func() {
					var expectedRoutingTableEntry models.RoutingTableEntry

					BeforeEach(func() {
						expectedRoutingTableEntry = models.NewRoutingTableEntry([]models.BackendServerInfo{backendServerInfo1, backendServerInfo2})
						backendServerInfo1.ModificationTag = routing_api_models.ModificationTag{Guid: "def"}
					})

					It("doesn't delete the backend", func() {
						updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
						Expect(updated).To(BeFalse())
						Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
					})
				})

				Context("when there are no more backends left", func() {
					BeforeEach(func() {
						updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo1)
						Expect(updated).To(BeTrue())
					})

					It("deletes the entry", func() {
						updated := routingTable.DeleteBackendServerKey(routingKey, backendServerInfo2)
						Expect(updated).To(BeTrue())
						Expect(routingTable.Size()).Should(Equal(0))
					})
				})
			})
		})
	})
})

func createBackendServerInfo(address string, port uint16, tag routing_api_models.ModificationTag) models.BackendServerInfo {
	return models.BackendServerInfo{Address: address, Port: port, ModificationTag: tag}

}
