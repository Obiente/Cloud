# EndpointSpec

Properties that can be configured to access and load balance a service.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mode** | **string** | The mode of resolution to use for internal load balancing between tasks.  | [optional] [default to ModeEnum_Vip]
**Ports** | [**Array&lt;EndpointPortConfig&gt;**](EndpointPortConfig.md) | List of exposed ports that this service is accessible on from the outside. Ports can only be provided if &#x60;vip&#x60; resolution mode is used.  | [optional] [default to undefined]

## Example

```typescript
import { EndpointSpec } from './api';

const instance: EndpointSpec = {
    Mode,
    Ports,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
