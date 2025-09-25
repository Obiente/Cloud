# Mount


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Target** | **string** | Container path. | [optional] [default to undefined]
**Source** | **string** | Mount source (e.g. a volume name, a host path). | [optional] [default to undefined]
**Type** | **string** | The mount type. Available types:  - &#x60;bind&#x60; Mounts a file or directory from the host into the container. Must exist prior to creating the container. - &#x60;volume&#x60; Creates a volume with the given name and options (or uses a pre-existing volume with the same name and options). These are **not** removed when the container is removed. - &#x60;image&#x60; Mounts an image. - &#x60;tmpfs&#x60; Create a tmpfs with the given options. The mount source cannot be specified for tmpfs. - &#x60;npipe&#x60; Mounts a named pipe from the host into the container. Must exist prior to creating the container. - &#x60;cluster&#x60; a Swarm cluster volume  | [optional] [default to undefined]
**ReadOnly** | **boolean** | Whether the mount should be read-only. | [optional] [default to undefined]
**Consistency** | **string** | The consistency requirement for the mount: &#x60;default&#x60;, &#x60;consistent&#x60;, &#x60;cached&#x60;, or &#x60;delegated&#x60;. | [optional] [default to undefined]
**BindOptions** | [**MountBindOptions**](MountBindOptions.md) |  | [optional] [default to undefined]
**VolumeOptions** | [**MountVolumeOptions**](MountVolumeOptions.md) |  | [optional] [default to undefined]
**ImageOptions** | [**MountImageOptions**](MountImageOptions.md) |  | [optional] [default to undefined]
**TmpfsOptions** | [**MountTmpfsOptions**](MountTmpfsOptions.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Mount } from './api';

const instance: Mount = {
    Target,
    Source,
    Type,
    ReadOnly,
    Consistency,
    BindOptions,
    VolumeOptions,
    ImageOptions,
    TmpfsOptions,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
