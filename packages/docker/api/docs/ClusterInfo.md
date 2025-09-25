# ClusterInfo

ClusterInfo represents information about the swarm as is returned by the \"/info\" endpoint. Join-tokens are not included. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | The ID of the swarm. | [optional] [default to undefined]
**Version** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**CreatedAt** | **string** | Date and time at which the swarm was initialised in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**UpdatedAt** | **string** | Date and time at which the swarm was last updated in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**Spec** | [**SwarmSpec**](SwarmSpec.md) |  | [optional] [default to undefined]
**TLSInfo** | [**TLSInfo**](TLSInfo.md) |  | [optional] [default to undefined]
**RootRotationInProgress** | **boolean** | Whether there is currently a root CA rotation in progress for the swarm  | [optional] [default to undefined]
**DataPathPort** | **number** | DataPathPort specifies the data path port number for data traffic. Acceptable port range is 1024 to 49151. If no port is set or is set to 0, the default port (4789) is used.  | [optional] [default to undefined]
**DefaultAddrPool** | **Array&lt;string&gt;** | Default Address Pool specifies default subnet pools for global scope networks.  | [optional] [default to undefined]
**SubnetSize** | **number** | SubnetSize specifies the subnet size of the networks created from the default subnet pool.  | [optional] [default to undefined]

## Example

```typescript
import { ClusterInfo } from './api';

const instance: ClusterInfo = {
    ID,
    Version,
    CreatedAt,
    UpdatedAt,
    Spec,
    TLSInfo,
    RootRotationInProgress,
    DataPathPort,
    DefaultAddrPool,
    SubnetSize,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
