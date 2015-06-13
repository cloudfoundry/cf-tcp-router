# TCP Router API overview

## Create backend mappings

To map a list of backend IPs and ports to a external IP and port on the TCP Router

```
POST /v0/external_ports
```

#### Request Sample

```
[
  {
   "external_port": 2222, 
   "backends" : [
      {"ip": "10.244.0.16", "port":5222}
   ]
  }
]
```

The request must contain at least one element in the array. Each elements includes the following properties:

#####`external_port` [required]

The `external_port` property refers to the external port that needs to be mapped to the given backends. 

#####`backends` [required]

The `backends` property refers to list of backend ips and ports to be mapped to the given external port. 

Each element of the backend collections has the following properties:

######`ip` [required]

The `ip` property refers to the host ip, where the application is hosted.

######`port` [required]

The `port` property refers to the host port, where the application is listening for incoming TCP requests.

#### Response

No response body is returned, and the 200 HTTP status code is returned for successful calls.
