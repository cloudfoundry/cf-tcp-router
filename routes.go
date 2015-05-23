package cf_tcp_router

import "github.com/tedsuo/rata"

const (
	// External Port mapping
	MapExternalPortRoute = "MapExternalPort"
)

var Routes = rata.Routes{
	// External Port mapping
	{Path: "/v0/external_ports", Method: "POST", Name: MapExternalPortRoute},
}
