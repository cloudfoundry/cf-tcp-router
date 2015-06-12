package cf_tcp_router_test

import (
	cf_tcp_router "github.com/GESoftware-CF/cf-tcp-router"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate", func() {

	Describe("BackendHostInfo", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12)
				Expect(backendHostInfo.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					backendHostInfo := cf_tcp_router.NewBackendHostInfo("", 12)
					err := backendHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					backendHostInfo := cf_tcp_router.NewBackendHostInfo("1.2.3.4", 0)
					err := backendHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("port"))
				})
			})
		})
	})

	Describe("BackendHostInfos", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				backendHostInfos := cf_tcp_router.BackendHostInfos{
					cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
					cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
				}
				Expect(backendHostInfos.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					backendHostInfos := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("", 12),
						cf_tcp_router.NewBackendHostInfo("", 13),
					}
					err := backendHostInfos.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					backendHostInfos := cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("1.2.3.4", 0),
						cf_tcp_router.NewBackendHostInfo("1.2.3.5", 12),
					}
					err := backendHostInfos.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("port"))
				})
			})

			Context("when empty", func() {
				It("raises an error", func() {
					backendHostInfos := cf_tcp_router.BackendHostInfos{}
					err := backendHostInfos.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("empty"))
				})
			})
		})
	})

	Describe("MappingRequest", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				mappingRequest := cf_tcp_router.NewMappingRequest(
					1234,
					cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
						cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
					})
				Expect(mappingRequest.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					mappingRequest := cf_tcp_router.NewMappingRequest(
						1234,
						cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("", 12),
							cf_tcp_router.NewBackendHostInfo("", 13),
						})
					err := mappingRequest.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})

			Context("when external port is zero", func() {
				It("raises an error", func() {
					mappingRequest := cf_tcp_router.NewMappingRequest(
						0,
						cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
							cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
						})
					err := mappingRequest.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("external_port"))
				})
			})
		})
	})

	Describe("MappingRequests", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				mappingRequests := cf_tcp_router.MappingRequests{
					cf_tcp_router.NewMappingRequest(1234, cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
						cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
					}),
				}
				Expect(mappingRequests.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					mappingRequests := cf_tcp_router.MappingRequests{
						cf_tcp_router.NewMappingRequest(1234, cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("", 12),
							cf_tcp_router.NewBackendHostInfo("", 13),
						}),
					}
					err := mappingRequests.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})
		})

		Context("when empty", func() {
			It("raises an error", func() {
				mappingRequests := cf_tcp_router.MappingRequests{}
				err := mappingRequests.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("empty"))
			})
		})
	})

	Describe("RouterHostInfo", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				routerHostInfo := cf_tcp_router.NewRouterHostInfo("1.2.3.4", 12)
				Expect(routerHostInfo.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					routerHostInfo := cf_tcp_router.NewRouterHostInfo("", 12)
					err := routerHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("external_ip"))
				})
			})

			Context("when port is invalid", func() {
				It("raises an error", func() {
					routerHostInfo := cf_tcp_router.NewRouterHostInfo("1.2.3.4", 0)
					err := routerHostInfo.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("external_port"))
				})
			})
		})
	})

	Describe("MappingResponse", func() {
		var (
			backends        cf_tcp_router.BackendHostInfos
			routerHostInfo  cf_tcp_router.RouterHostInfo
			mappingResponse cf_tcp_router.MappingResponse
		)

		BeforeEach(func() {
			backends = cf_tcp_router.BackendHostInfos{
				cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
				cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
			}
			routerHostInfo = cf_tcp_router.NewRouterHostInfo("1.2.3.4", 12)
		})

		JustBeforeEach(func() {
			mappingResponse = cf_tcp_router.NewMappingResponse(backends, routerHostInfo)
		})

		Context("when values are valid", func() {
			It("does not raise an error", func() {
				Expect(mappingResponse.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when router host info is invalid", func() {
				BeforeEach(func() {
					routerHostInfo = cf_tcp_router.NewRouterHostInfo("1.2.3.4", 0)
				})

				It("raises an error", func() {
					err := mappingResponse.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("external_port"))
				})
			})

			Context("when backend info is invalid", func() {
				BeforeEach(func() {
					backends = cf_tcp_router.BackendHostInfos{
						cf_tcp_router.NewBackendHostInfo("", 12),
						cf_tcp_router.NewBackendHostInfo("", 13),
					}
				})

				It("raises an error", func() {
					err := mappingResponse.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})
		})
	})

	Describe("MappingResponses", func() {
		Context("when values are valid", func() {
			It("does not raise an error", func() {
				mappingResponses := cf_tcp_router.MappingResponses{
					cf_tcp_router.NewMappingResponse(
						cf_tcp_router.BackendHostInfos{
							cf_tcp_router.NewBackendHostInfo("1.2.3.4", 12),
							cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
						},
						cf_tcp_router.NewRouterHostInfo("1.2.3.4", 12)),
				}
				Expect(mappingResponses.Validate()).ShouldNot(HaveOccurred())
			})
		})

		Context("when values are invalid", func() {
			Context("when address is empty", func() {
				It("raises an error", func() {
					mappingResponses := cf_tcp_router.MappingResponses{
						cf_tcp_router.NewMappingResponse(
							cf_tcp_router.BackendHostInfos{
								cf_tcp_router.NewBackendHostInfo("", 12),
								cf_tcp_router.NewBackendHostInfo("1.2.3.5", 13),
							},
							cf_tcp_router.NewRouterHostInfo("1.2.3.4", 12)),
					}
					err := mappingResponses.Validate()
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("ip"))
				})
			})
		})

		Context("when empty", func() {
			It("raises an error", func() {
				mappingResponses := cf_tcp_router.MappingResponses{}
				err := mappingResponses.Validate()
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("empty"))
			})
		})
	})
})
