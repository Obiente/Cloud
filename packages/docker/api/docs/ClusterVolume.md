# ClusterVolume

Options and information specific to, and only present on, Swarm CSI cluster volumes. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | The Swarm ID of this volume. Because cluster volumes are Swarm objects, they have an ID, unlike non-cluster volumes. This ID can be used to refer to the Volume instead of the name.  | [optional] [default to undefined]
**Version** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**CreatedAt** | **string** |  | [optional] [default to undefined]
**UpdatedAt** | **string** |  | [optional] [default to undefined]
**Spec** | [**ClusterVolumeSpec**](ClusterVolumeSpec.md) |  | [optional] [default to undefined]
**Info** | [**ClusterVolumeInfo**](ClusterVolumeInfo.md) |  | [optional] [default to undefined]
**PublishStatus** | [**Array&lt;ClusterVolumePublishStatusInner&gt;**](ClusterVolumePublishStatusInner.md) | The status of the volume as it pertains to its publishing and use on specific nodes  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolume } from './api';

const instance: ClusterVolume = {
    ID,
    Version,
    CreatedAt,
    UpdatedAt,
    Spec,
    Info,
    PublishStatus,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
