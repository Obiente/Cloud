# DeviceRequest

A request for devices to be sent to device drivers

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | **string** |  | [optional] [default to undefined]
**Count** | **number** |  | [optional] [default to undefined]
**DeviceIDs** | **Array&lt;string&gt;** |  | [optional] [default to undefined]
**Capabilities** | **Array&lt;Array&lt;string&gt;&gt;** | A list of capabilities; an OR list of AND lists of capabilities.  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | Driver-specific options, specified as a key/value pairs. These options are passed directly to the driver.  | [optional] [default to undefined]

## Example

```typescript
import { DeviceRequest } from './api';

const instance: DeviceRequest = {
    Driver,
    Count,
    DeviceIDs,
    Capabilities,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
