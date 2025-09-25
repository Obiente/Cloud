# VolumeCreateOptions

Volume configuration

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | The new volume\&#39;s name. If not specified, Docker generates a name.  | [optional] [default to undefined]
**Driver** | **string** | Name of the volume driver to use. | [optional] [default to 'local']
**DriverOpts** | **{ [key: string]: string; }** | A mapping of driver options and values. These options are passed directly to the driver and are driver specific.  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**ClusterVolumeSpec** | [**ClusterVolumeSpec**](ClusterVolumeSpec.md) |  | [optional] [default to undefined]

## Example

```typescript
import { VolumeCreateOptions } from './api';

const instance: VolumeCreateOptions = {
    Name,
    Driver,
    DriverOpts,
    Labels,
    ClusterVolumeSpec,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
