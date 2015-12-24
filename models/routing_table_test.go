package models_test

import (
	"github.com/cloudfoundry-incubator/cf-tcp-router/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RoutingTable", func() {
	var routingTable models.RoutingTable

	BeforeEach(func() {
		routingTable = models.NewRoutingTable()
	})

	Describe("Set", func() {
		Context("when a new entry is added", func() {
			It("adds the entry", func() {
				routingKey := models.RoutingKey{Port: 12}
				routingTableEntry := models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						models.BackendServerInfo{Address: "some-ip", Port: 1234}: struct{}{},
					},
				}
				ok := routingTable.Set(routingKey, routingTableEntry)
				Expect(ok).To(BeTrue())
				Expect(routingTable.Get(routingKey)).To(Equal(routingTableEntry))
				Expect(routingTable.Size()).To(Equal(1))
			})
		})

		Context("when setting pre-existing routing key", func() {
			var (
				routingKey                models.RoutingKey
				existingRoutingTableEntry models.RoutingTableEntry
			)
			BeforeEach(func() {
				routingKey = models.RoutingKey{Port: 12}
				existingRoutingTableEntry = models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						models.BackendServerInfo{Address: "some-ip-1", Port: 1234}: struct{}{},
						models.BackendServerInfo{Address: "some-ip-2", Port: 1234}: struct{}{},
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
							Backends: map[models.BackendServerInfo]struct{}{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234}: struct{}{},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})

				Context("when at least one backend server info is different", func() {
					It("overwrites the value", func() {
						routingTableEntry := models.RoutingTableEntry{
							Backends: map[models.BackendServerInfo]struct{}{
								models.BackendServerInfo{Address: "some-ip-1", Port: 1234}: struct{}{},
								models.BackendServerInfo{Address: "some-ip-2", Port: 2345}: struct{}{},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})

				Context("when all backend servers info are different", func() {
					It("overwrites the value", func() {
						routingTableEntry := models.RoutingTableEntry{
							Backends: map[models.BackendServerInfo]struct{}{
								models.BackendServerInfo{Address: "some-ip-1", Port: 3456}: struct{}{},
								models.BackendServerInfo{Address: "some-ip-2", Port: 2345}: struct{}{},
							},
						}
						verifyChangedValue(routingTableEntry)
					})
				})
			})

			Context("with same value", func() {
				It("returns false", func() {
					routingTableEntry := models.RoutingTableEntry{
						Backends: map[models.BackendServerInfo]struct{}{
							models.BackendServerInfo{Address: "some-ip-1", Port: 1234}: struct{}{},
							models.BackendServerInfo{Address: "some-ip-2", Port: 1234}: struct{}{},
						},
					}
					ok := routingTable.Set(routingKey, routingTableEntry)
					Expect(ok).To(BeFalse())
					Expect(routingTable.Get(routingKey)).Should(Equal(existingRoutingTableEntry))
				})
			})
		})
	})

	Describe("UpsertBackendServerInfo", func() {
		var (
			routingKey        models.RoutingKey
			backendServerInfo models.BackendServerInfo
		)
		BeforeEach(func() {
			routingKey = models.RoutingKey{Port: 12}
			backendServerInfo = models.BackendServerInfo{Address: "some-ip", Port: 1234}
		})
		Context("when the routing key does not exist", func() {
			It("inserts the routing key with its backends", func() {
				routingTableEntry := models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						backendServerInfo: struct{}{},
					},
				}
				ok := routingTable.UpsertBackendServerInfo(routingKey, backendServerInfo)
				Expect(ok).To(BeTrue())
				Expect(routingTable.Get(routingKey)).Should(Equal(routingTableEntry))
				Expect(routingTable.Size()).To(Equal(1))
			})
		})
		Context("when the routing key does exist", func() {
			BeforeEach(func() {
				existingRoutingTableEntry := models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						backendServerInfo: struct{}{},
					},
				}
				ok := routingTable.Set(routingKey, existingRoutingTableEntry)
				Expect(ok).To(BeTrue())
			})

			Context("and no change in the backends are provided", func() {
				It("it does not update the routing entry", func() {
					sameBackendServerInfo := models.BackendServerInfo{Address: "some-ip", Port: 1234}
					ok := routingTable.UpsertBackendServerInfo(routingKey, sameBackendServerInfo)
					Expect(ok).To(BeFalse())
				})
			})

			Context("and a new backend is provided", func() {
				It("it updates the routing entry's backends", func() {
					routingTableEntry := models.RoutingTableEntry{
						Backends: map[models.BackendServerInfo]struct{}{
							models.BackendServerInfo{Address: "some-ip", Port: 1234}:       struct{}{},
							models.BackendServerInfo{Address: "some-other-ip", Port: 1234}: struct{}{},
						},
					}
					differentBackendServerInfo := models.BackendServerInfo{Address: "some-other-ip", Port: 1234}
					ok := routingTable.UpsertBackendServerInfo(routingKey, differentBackendServerInfo)
					Expect(ok).To(BeTrue())
					Expect(routingTable.Get(routingKey)).Should(Equal(routingTableEntry))
				})
			})
		})
	})

	Describe("DeleteBackendServerInfo", func() {
		var (
			routingKey                models.RoutingKey
			backendServerInfo         models.BackendServerInfo
			existingRoutingTableEntry models.RoutingTableEntry
		)
		BeforeEach(func() {
			routingKey = models.RoutingKey{Port: 12}
			backendServerInfo = models.BackendServerInfo{Address: "some-ip", Port: 1234}
		})

		Context("when the routing key does not exist", func() {
			It("it does not causes any changes or errors", func() {
				ok := routingTable.DeleteBackendServerInfo(routingKey, backendServerInfo)
				Expect(ok).To(BeFalse())
			})
		})

		Context("when the routing key does exist", func() {
			BeforeEach(func() {
				existingRoutingTableEntry = models.RoutingTableEntry{
					Backends: map[models.BackendServerInfo]struct{}{
						backendServerInfo:                                              struct{}{},
						models.BackendServerInfo{Address: "some-other-ip", Port: 1235}: struct{}{},
					},
				}
				ok := routingTable.Set(routingKey, existingRoutingTableEntry)
				Expect(ok).To(BeTrue())
			})

			Context("and the backend does not exist ", func() {
				It("it does not causes any changes or errors", func() {
					someBackendServerInfo := models.BackendServerInfo{Address: "some-missing-ip", Port: 1236}
					ok := routingTable.DeleteBackendServerInfo(routingKey, someBackendServerInfo)
					Expect(ok).To(BeFalse())
					Expect(routingTable.Get(routingKey)).Should(Equal(existingRoutingTableEntry))
				})
			})

			Context("and the backend does exist", func() {
				It("it deletes the backend", func() {
					ok := routingTable.DeleteBackendServerInfo(routingKey, backendServerInfo)
					Expect(ok).To(BeTrue())
					expectedRoutingTableEntry := models.RoutingTableEntry{
						Backends: map[models.BackendServerInfo]struct{}{
							models.BackendServerInfo{Address: "some-other-ip", Port: 1235}: struct{}{},
						},
					}
					Expect(routingTable.Get(routingKey)).Should(Equal(expectedRoutingTableEntry))
				})

				Context("when there are no more backends left", func() {
					It("deletes the entry", func() {
						ok := routingTable.DeleteBackendServerInfo(routingKey, backendServerInfo)
						Expect(ok).To(BeTrue())

						ok = routingTable.DeleteBackendServerInfo(routingKey, models.BackendServerInfo{Address: "some-other-ip", Port: 1235})
						Expect(ok).To(BeTrue())

						Expect(routingTable.Size()).Should(Equal(0))
					})
				})
			})
		})
	})
})
