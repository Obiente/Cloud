# Network


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the network.  | [optional] [default to undefined]
**Id** | **string** | ID that uniquely identifies a network on a single machine.  | [optional] [default to undefined]
**Created** | **string** | Date and time at which the network was created in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**Scope** | **string** | The level at which the network exists (e.g. &#x60;swarm&#x60; for cluster-wide or &#x60;local&#x60; for machine level)  | [optional] [default to undefined]
**Driver** | **string** | The name of the driver used to create the network (e.g. &#x60;bridge&#x60;, &#x60;overlay&#x60;).  | [optional] [default to undefined]
**EnableIPv4** | **boolean** | Whether the network was created with IPv4 enabled.  | [optional] [default to undefined]
**EnableIPv6** | **boolean** | Whether the network was created with IPv6 enabled.  | [optional] [default to undefined]
**IPAM** | [**IPAM**](IPAM.md) |  | [optional] [default to undefined]
**Internal** | **boolean** | Whether the network is created to only allow internal networking connectivity.  | [optional] [default to false]
**Attachable** | **boolean** | Whether a global / swarm scope network is manually attachable by regular containers from workers in swarm mode.  | [optional] [default to false]
**Ingress** | **boolean** | Whether the network is providing the routing-mesh for the swarm cluster.  | [optional] [default to false]
**ConfigFrom** | [**ConfigReference**](ConfigReference.md) |  | [optional] [default to undefined]
**ConfigOnly** | **boolean** | Whether the network is a config-only network. Config-only networks are placeholder networks for network configurations to be used by other networks. Config-only networks cannot be used directly to run containers or services.  | [optional] [default to false]
**Containers** | [**{ [key: string]: NetworkContainer; }**](NetworkContainer.md) | Contains endpoints attached to the network.  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | Network-specific options uses when creating the network.  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Peers** | [**Array&lt;PeerInfo&gt;**](PeerInfo.md) | List of peer nodes for an overlay network. This field is only present for overlay networks, and omitted for other network types.  | [optional] [default to undefined]

## Example

```typescript
import { Network } from './api';

const instance: Network = {
    Name,
    Id,
    Created,
    Scope,
    Driver,
    EnableIPv4,
    EnableIPv6,
    IPAM,
    Internal,
    Attachable,
    Ingress,
    ConfigFrom,
    ConfigOnly,
    Containers,
    Options,
    Labels,
    Peers,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
