# MountVolumeOptionsDriverConfig

Map of driver specific options

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the driver to use to create the volume. | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | key/value map of driver specific options. | [optional] [default to undefined]

## Example

```typescript
import { MountVolumeOptionsDriverConfig } from './api';

const instance: MountVolumeOptionsDriverConfig = {
    Name,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
