# EndpointIPAMConfig

EndpointIPAMConfig represents an endpoint\'s IPAM configuration. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IPv4Address** | **string** |  | [optional] [default to undefined]
**IPv6Address** | **string** |  | [optional] [default to undefined]
**LinkLocalIPs** | **Array&lt;string&gt;** |  | [optional] [default to undefined]

## Example

```typescript
import { EndpointIPAMConfig } from './api';

const instance: EndpointIPAMConfig = {
    IPv4Address,
    IPv6Address,
    LinkLocalIPs,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
