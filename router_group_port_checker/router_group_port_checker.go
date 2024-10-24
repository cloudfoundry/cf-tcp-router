package router_group_port_checker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/models"
	"code.cloudfoundry.org/routing-api/uaaclient"
	"golang.org/x/oauth2"
)

type PortChecker struct {
	routingAPIClient routing_api.Client
	uaaTokenFetcher  uaaclient.TokenFetcher
}

func NewPortChecker(routingAPIClient routing_api.Client, uaaTokenFetcher uaaclient.TokenFetcher) PortChecker {
	return PortChecker{
		routingAPIClient: routingAPIClient,
		uaaTokenFetcher:  uaaTokenFetcher,
	}
}

func (pc *PortChecker) Check(systemComponentPorts []uint16) (bool, error) {
	routerGroups, err := pc.getRouterGroups()
	if err != nil {
		return false, err
	}
	shouldExit, portErrors := validateRouterGroups(routerGroups, systemComponentPorts)

	if len(portErrors) == 0 {
		return false, nil
	}

	return shouldExit, errors.New(strings.Join(portErrors, "\n"))
}

func (pc *PortChecker) getRouterGroups() ([]models.RouterGroup, error) {
	var err error
	numRetries := 3

	for i := 0; i < numRetries; i++ {
		var token *oauth2.Token
		token, err = pc.uaaTokenFetcher.FetchToken(context.Background(), false)
		if err != nil {
			continue
		}
		pc.routingAPIClient.SetToken(token.AccessToken)
		break
	}

	if err != nil {
		return nil, fmt.Errorf("error-fetching-uaa-token: \"%s\"", err.Error())
	}

	for i := 0; i < numRetries; i++ {
		var routerGroups []models.RouterGroup
		routerGroups, err = pc.routingAPIClient.RouterGroups()
		if err == nil {
			return routerGroups, nil
		}
	}
	return nil, fmt.Errorf("error-fetching-routing-groups: \"%s\"", err.Error())
}

func validateRouterGroups(routerGroups []models.RouterGroup, systemComponentPorts []uint16) (bool, []string) {
	var errors []string
	shouldExit := false

	for _, group := range routerGroups {
		reservablePorts := group.ReservablePorts
		ranges, _ := reservablePorts.Parse()
		for _, r := range ranges {
			start, end := r.Endpoints()

			overlappingPorts := []string{}
			for _, port := range systemComponentPorts {
				if port >= start && port <= end {
					overlappingPorts = append(overlappingPorts, fmt.Sprintf("%d", port))
				}
			}

			if len(overlappingPorts) > 0 {
				shouldExit = true
				formattedPorts := strings.Join(overlappingPorts, ", ")
				errors = append(errors, fmt.Sprintf("The reserved ports for router group '%v' contains the following reserved system component port(s): '%v'. Please update your router group accordingly.", group.Name, formattedPorts))
			}
		}
	}
	return shouldExit, errors
}
