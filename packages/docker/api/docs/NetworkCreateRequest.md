# NetworkCreateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | The network\&#39;s name. | [default to undefined]
**Driver** | **string** | Name of the network driver plugin to use. | [optional] [default to 'bridge']
**Scope** | **string** | The level at which the network exists (e.g. &#x60;swarm&#x60; for cluster-wide or &#x60;local&#x60; for machine level).  | [optional] [default to undefined]
**Internal** | **boolean** | Restrict external access to the network. | [optional] [default to undefined]
**Attachable** | **boolean** | Globally scoped network is manually attachable by regular containers from workers in swarm mode.  | [optional] [default to undefined]
**Ingress** | **boolean** | Ingress network is the network which provides the routing-mesh in swarm mode.  | [optional] [default to undefined]
**ConfigOnly** | **boolean** | Creates a config-only network. Config-only networks are placeholder networks for network configurations to be used by other networks. Config-only networks cannot be used directly to run containers or services.  | [optional] [default to false]
**ConfigFrom** | [**ConfigReference**](ConfigReference.md) |  | [optional] [default to undefined]
**IPAM** | [**IPAM**](IPAM.md) |  | [optional] [default to undefined]
**EnableIPv4** | **boolean** | Enable IPv4 on the network. | [optional] [default to undefined]
**EnableIPv6** | **boolean** | Enable IPv6 on the network. | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | Network specific options to be used by the drivers. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]

## Example

```typescript
import { NetworkCreateRequest } from './api';

const instance: NetworkCreateRequest = {
    Name,
    Driver,
    Scope,
    Internal,
    Attachable,
    Ingress,
    ConfigOnly,
    ConfigFrom,
    IPAM,
    EnableIPv4,
    EnableIPv6,
    Options,
    Labels,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
