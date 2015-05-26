package cf_tcp_router

type ErrInvalidField struct {
	Field string
}

func (err ErrInvalidField) Error() string {
	return "Invalid field: " + err.Field
}

type BackendHostInfo struct {
	Address string `json:"backend_ip"`
	Port    uint16 `json:"backend_port"`
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
		return ErrInvalidField{"backend_ip"}
	}
	if h.Port == 0 {
		return ErrInvalidField{"backend_port"}
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

type RouterHostInfo struct {
	Address string `json:"router_ip"`
	Port    uint16 `json:"router_port"`
}

func NewRouterHostInfo(address string, port uint16) RouterHostInfo {
	return RouterHostInfo{
		Address: address,
		Port:    port,
	}
}

func (h RouterHostInfo) Validate() error {
	if h.Address == "" {
		return ErrInvalidField{"router_ip"}
	}
	if h.Port == 0 {
		return ErrInvalidField{"router_port"}
	}
	return nil
}
