# ClusterVolumeSpecAccessModeCapacityRange

The desired capacity that the volume should be created with. If empty, the plugin will decide the capacity. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RequiredBytes** | **number** | The volume must be at least this big. The value of 0 indicates an unspecified minimum  | [optional] [default to undefined]
**LimitBytes** | **number** | The volume must not be bigger than this. The value of 0 indicates an unspecified maximum.  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolumeSpecAccessModeCapacityRange } from './api';

const instance: ClusterVolumeSpecAccessModeCapacityRange = {
    RequiredBytes,
    LimitBytes,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
