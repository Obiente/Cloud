# ServiceEndpoint


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Spec** | [**EndpointSpec**](EndpointSpec.md) |  | [optional] [default to undefined]
**Ports** | [**Array&lt;EndpointPortConfig&gt;**](EndpointPortConfig.md) |  | [optional] [default to undefined]
**VirtualIPs** | [**Array&lt;ServiceEndpointVirtualIPsInner&gt;**](ServiceEndpointVirtualIPsInner.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ServiceEndpoint } from './api';

const instance: ServiceEndpoint = {
    Spec,
    Ports,
    VirtualIPs,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
