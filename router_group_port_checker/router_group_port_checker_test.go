package router_group_port_checker_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/routing-api/fake_routing_api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"

	"code.cloudfoundry.org/cf-tcp-router/router_group_port_checker"
	"code.cloudfoundry.org/routing-api/models"
	test_uaa_client "code.cloudfoundry.org/routing-api/uaaclient/fakes"
)

var _ = Describe("RouterGroupPortChecker", func() {
	var (
		fakeRoutingApiClient       *fake_routing_api.FakeClient
		fakeTokenFetcher           *test_uaa_client.FakeTokenFetcher
		token                      *oauth2.Token
		routerGroup1, routerGroup2 models.RouterGroup
	)
	BeforeEach(func() {

		fakeRoutingApiClient = new(fake_routing_api.FakeClient)
		fakeTokenFetcher = &test_uaa_client.FakeTokenFetcher{}
		token = &oauth2.Token{
			AccessToken: "access_token",
			Expiry:      time.Now().Add(5 * time.Second),
		}
		routerGroup1 = models.RouterGroup{
			Name:            "router-group-1",
			Type:            "tcp",
			ReservablePorts: "1024-2000",
		}
		routerGroup2 = models.RouterGroup{
			Name:            "router-group-2",
			Type:            "tcp",
			ReservablePorts: "2001-2048",
		}

	})
	It("doesn't return an error when there is no overlaps and should not exit", func() {
		fakeTokenFetcher.FetchTokenReturns(token, nil)
		fakeRoutingApiClient.RouterGroupsReturns([]models.RouterGroup{routerGroup1}, nil)
		checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
		shouldExit, err := checker.Check([]uint16{2048})

		Expect(fakeRoutingApiClient.SetTokenArgsForCall(0)).To(Equal(token.AccessToken))
		Expect(err).To(BeNil())
		Expect(shouldExit).To(BeFalse())
	})

	It("Returns an error when there is an overlap and should exit", func() {
		fakeTokenFetcher.FetchTokenReturns(token, nil)
		fakeRoutingApiClient.RouterGroupsReturns([]models.RouterGroup{routerGroup1}, nil)
		checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
		shouldExit, err := checker.Check([]uint16{1026})

		Expect(fakeRoutingApiClient.SetTokenArgsForCall(0)).To(Equal(token.AccessToken))

		msg := "The reserved ports for router group 'router-group-1' contains the following reserved system component port(s): '1026'. Please update your router group accordingly."
		Expect(err).To(MatchError(msg))
		Expect(shouldExit).To(BeTrue())
	})

	It("Returns multiple errors when there is multiple overlaps and should exit", func() {
		fakeTokenFetcher.FetchTokenReturns(token, nil)
		fakeRoutingApiClient.RouterGroupsReturns([]models.RouterGroup{routerGroup1, routerGroup2}, nil)
		checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
		shouldExit, err := checker.Check([]uint16{1026, 1027, 2001, 2002})

		Expect(fakeRoutingApiClient.SetTokenArgsForCall(0)).To(Equal(token.AccessToken))

		msg := "The reserved ports for router group 'router-group-1' contains the following reserved system component port(s): '1026, 1027'. Please update your router group accordingly.\n"
		msg = msg + "The reserved ports for router group 'router-group-2' contains the following reserved system component port(s): '2001, 2002'. Please update your router group accordingly."

		Expect(err).To(MatchError(msg))
		Expect(shouldExit).To(BeTrue())
	})

	Context("when routing api requires retries", func() {
		Context("but eventually works", func() {
			BeforeEach(func() {
				fakeTokenFetcher.FetchTokenReturns(token, nil)
				fakeRoutingApiClient.RouterGroupsReturnsOnCall(0, []models.RouterGroup{}, errors.New("oh no!"))
				fakeRoutingApiClient.RouterGroupsReturnsOnCall(1, []models.RouterGroup{}, errors.New("oh no!"))
				fakeRoutingApiClient.RouterGroupsReturnsOnCall(2, []models.RouterGroup{routerGroup1}, nil)
			})

			It("doesn't error when there is no overlap and should not exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{2048})
				Expect(err).To(BeNil())
				Expect(shouldExit).To(BeFalse())
			})

			It("returns an error when there is an overlap and should exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{1026})
				msg := "The reserved ports for router group 'router-group-1' contains the following reserved system component port(s): '1026'. Please update your router group accordingly."
				Expect(err).To(MatchError(msg))
				Expect(shouldExit).To(BeTrue())
			})
		})

		Context("and always fails", func() {
			BeforeEach(func() {
				fakeTokenFetcher.FetchTokenReturns(token, nil)
				fakeRoutingApiClient.RouterGroupsReturns([]models.RouterGroup{}, errors.New("oh no!"))
			})

			It("returns an error and should not exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{})
				Expect(err).To(MatchError("error-fetching-routing-groups: \"oh no!\""))
				Expect(shouldExit).To(BeFalse())
			})
		})
	})

	Context("when fetch token requires retries", func() {
		Context("but eventually works", func() {
			BeforeEach(func() {
				fakeTokenFetcher.FetchTokenReturnsOnCall(0, nil, errors.New("oh no!"))
				fakeTokenFetcher.FetchTokenReturnsOnCall(1, nil, errors.New("oh no!"))
				fakeTokenFetcher.FetchTokenReturnsOnCall(2, token, nil)
				fakeRoutingApiClient.RouterGroupsReturns([]models.RouterGroup{routerGroup1}, nil)
			})

			It("doesn't error when there is no overlap and should not exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{2048})
				Expect(err).To(BeNil())
				Expect(shouldExit).To(BeFalse())
			})

			It("returns an error when there is an overlap and should exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{1026})
				msg := "The reserved ports for router group 'router-group-1' contains the following reserved system component port(s): '1026'. Please update your router group accordingly."
				Expect(err).To(MatchError(msg))
				Expect(shouldExit).To(BeTrue())
			})
		})
		Context("and always fails", func() {
			BeforeEach(func() {
				fakeTokenFetcher.FetchTokenReturns(nil, errors.New("oh no!"))
			})
			It("returns an error and should not exit", func() {
				checker := router_group_port_checker.NewPortChecker(fakeRoutingApiClient, fakeTokenFetcher)
				shouldExit, err := checker.Check([]uint16{})
				Expect(err).To(MatchError("error-fetching-uaa-token: \"oh no!\""))
				Expect(shouldExit).To(BeFalse())
			})
		})
	})
})
