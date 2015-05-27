# TCP Router API overview

## Create backend mappings

To map a list of backend IPs and ports to a external IP and port on the TCP Router

```
POST /v0/external_ports
```

#### Request Sample

```
[
  {"backend_ip": "10.244.0.16", "backend_port":5222}
]
```

The request must contain at least one element in the array. Each elements includes the following properties:

#####`backend_ip` [required]

The `backend_ip` property refers to the host ip, where the application is hosted.

#####`backend_port` [required]

The `backend_port` property refers to the host port, where the application is listening for incoming TCP requests.

#### Response sample

```
{"router_ip": "10.244.0.34", "router_port":5222}
```

#####`router_ip`

The `router_ip` property refers to the external IP of the TCP router.

#####`router_port`

The `router_port` property refers to the external port of the TCP router, to which the backend IP and port are mapped.
