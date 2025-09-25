# DeviceInfo

DeviceInfo represents a device that can be used by a container. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Source** | **string** | The origin device driver.  | [optional] [default to undefined]
**ID** | **string** | The unique identifier for the device within its source driver. For CDI devices, this would be an FQDN like \&quot;vendor.com/gpu&#x3D;0\&quot;.  | [optional] [default to undefined]

## Example

```typescript
import { DeviceInfo } from './api';

const instance: DeviceInfo = {
    Source,
    ID,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
