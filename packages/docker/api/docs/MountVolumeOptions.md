# MountVolumeOptions

Optional configuration for the `volume` type.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NoCopy** | **boolean** | Populate volume with data from the target. | [optional] [default to false]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**DriverConfig** | [**MountVolumeOptionsDriverConfig**](MountVolumeOptionsDriverConfig.md) |  | [optional] [default to undefined]
**Subpath** | **string** | Source path inside the volume. Must be relative without any back traversals. | [optional] [default to undefined]

## Example

```typescript
import { MountVolumeOptions } from './api';

const instance: MountVolumeOptions = {
    NoCopy,
    Labels,
    DriverConfig,
    Subpath,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
