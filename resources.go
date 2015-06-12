package cf_tcp_router

const (
	ErrInvalidMapingRequest     = "Invalid mapping request"
	ErrRouterConfigFileNotFound = "Configuration file not found"
	ErrRouterEmptyConfigFile    = "Configuration file not specified"
	ErrInvalidStartFrontendPort = "Invalid start frontend port"

	LowerBoundStartFrontendPort = 1024
)

type ErrInvalidField struct {
	Field string
}

func (err ErrInvalidField) Error() string {
	return "Invalid field: " + err.Field
}

type BackendHostInfo struct {
	Address string `json:"ip"`
	Port    uint16 `json:"port"`
}

type BackendHostInfos []BackendHostInfo

func NewBackendHostInfo(address string, port uint16) BackendHostInfo {
	return BackendHostInfo{
		Address: address,
		Port:    port,
	}
}

func (h BackendHostInfo) Validate() error {
	if h.Address == "" {
		return ErrInvalidField{"ip"}
	}
	if h.Port == 0 {
		return ErrInvalidField{"port"}
	}
	return nil
}

func (h BackendHostInfos) Validate() error {
	var err error
	if len(h) == 0 {
		return ErrInvalidField{"empty"}
	}
	for _, hostInfo := range h {
		err = hostInfo.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type MappingRequest struct {
	ExternalPort uint16           `json:"external_port"`
	Backends     BackendHostInfos `json:"backends"`
}

type MappingRequests []MappingRequest

func NewMappingRequest(externalPort uint16, backends BackendHostInfos) MappingRequest {
	return MappingRequest{
		ExternalPort: externalPort,
		Backends:     backends,
	}
}

func (m MappingRequest) Validate() error {
	if m.ExternalPort == 0 {
		return ErrInvalidField{"external_port"}
	}
	return m.Backends.Validate()
}

func (m MappingRequests) Validate() error {
	var err error
	if len(m) == 0 {
		return ErrInvalidField{"empty"}
	}
	for _, req := range m {
		err = req.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type RouterHostInfo struct {
	Address string `json:"external_ip"`
	Port    uint16 `json:"external_port"`
}

func NewRouterHostInfo(address string, port uint16) RouterHostInfo {
	return RouterHostInfo{
		Address: address,
		Port:    port,
	}
}

func (h RouterHostInfo) Validate() error {
	if h.Address == "" {
		return ErrInvalidField{"external_ip"}
	}
	if h.Port == 0 {
		return ErrInvalidField{"external_port"}
	}
	return nil
}

type MappingResponse struct {
	RouterHostInfo
	Backends BackendHostInfos `json:"backends"`
}

func NewMappingResponse(backends BackendHostInfos, routerInfo RouterHostInfo) MappingResponse {
	return MappingResponse{
		Backends:       backends,
		RouterHostInfo: routerInfo,
	}
}

func (m MappingResponse) Validate() error {
	if err := m.RouterHostInfo.Validate(); err != nil {
		return err
	}
	if err := m.Backends.Validate(); err != nil {
		return err
	}
	return nil
}

type MappingResponses []MappingResponse

func (m MappingResponses) Validate() error {
	var err error
	if len(m) == 0 {
		return ErrInvalidField{"empty"}
	}
	for _, req := range m {
		err = req.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}
