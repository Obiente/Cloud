# NetworkAttachmentConfig

Specifies how a service should be attached to a particular network. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Target** | **string** | The target network for attachment. Must be a network name or ID.  | [optional] [default to undefined]
**Aliases** | **Array&lt;string&gt;** | Discoverable alternate names for the service on this network.  | [optional] [default to undefined]
**DriverOpts** | **{ [key: string]: string; }** | Driver attachment options for the network target.  | [optional] [default to undefined]

## Example

```typescript
import { NetworkAttachmentConfig } from './api';

const instance: NetworkAttachmentConfig = {
    Target,
    Aliases,
    DriverOpts,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
