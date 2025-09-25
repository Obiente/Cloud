# SwarmInitRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ListenAddr** | **string** | Listen address used for inter-manager communication, as well as determining the networking interface used for the VXLAN Tunnel Endpoint (VTEP). This can either be an address/port combination in the form &#x60;192.168.1.1:4567&#x60;, or an interface followed by a port number, like &#x60;eth0:4567&#x60;. If the port number is omitted, the default swarm listening port is used.  | [optional] [default to undefined]
**AdvertiseAddr** | **string** | Externally reachable address advertised to other nodes. This can either be an address/port combination in the form &#x60;192.168.1.1:4567&#x60;, or an interface followed by a port number, like &#x60;eth0:4567&#x60;. If the port number is omitted, the port number from the listen address is used. If &#x60;AdvertiseAddr&#x60; is not specified, it will be automatically detected when possible.  | [optional] [default to undefined]
**DataPathAddr** | **string** | Address or interface to use for data path traffic (format: &#x60;&lt;ip|interface&gt;&#x60;), for example,  &#x60;192.168.1.1&#x60;, or an interface, like &#x60;eth0&#x60;. If &#x60;DataPathAddr&#x60; is unspecified, the same address as &#x60;AdvertiseAddr&#x60; is used.  The &#x60;DataPathAddr&#x60; specifies the address that global scope network drivers will publish towards other  nodes in order to reach the containers running on this node. Using this parameter it is possible to separate the container data traffic from the management traffic of the cluster.  | [optional] [default to undefined]
**DataPathPort** | **number** | DataPathPort specifies the data path port number for data traffic. Acceptable port range is 1024 to 49151. if no port is set or is set to 0, default port 4789 will be used.  | [optional] [default to undefined]
**DefaultAddrPool** | **Array&lt;string&gt;** | Default Address Pool specifies default subnet pools for global scope networks.  | [optional] [default to undefined]
**ForceNewCluster** | **boolean** | Force creation of a new swarm. | [optional] [default to undefined]
**SubnetSize** | **number** | SubnetSize specifies the subnet size of the networks created from the default subnet pool.  | [optional] [default to undefined]
**Spec** | [**SwarmSpec**](SwarmSpec.md) |  | [optional] [default to undefined]

## Example

```typescript
import { SwarmInitRequest } from './api';

const instance: SwarmInitRequest = {
    ListenAddr,
    AdvertiseAddr,
    DataPathAddr,
    DataPathPort,
    DefaultAddrPool,
    ForceNewCluster,
    SubnetSize,
    Spec,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
