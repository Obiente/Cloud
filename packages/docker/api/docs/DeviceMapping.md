# DeviceMapping

A device mapping between the host and container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PathOnHost** | **string** |  | [optional] [default to undefined]
**PathInContainer** | **string** |  | [optional] [default to undefined]
**CgroupPermissions** | **string** |  | [optional] [default to undefined]

## Example

```typescript
import { DeviceMapping } from './api';

const instance: DeviceMapping = {
    PathOnHost,
    PathInContainer,
    CgroupPermissions,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
