# ImageSummary


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** | ID is the content-addressable ID of an image.  This identifier is a content-addressable digest calculated from the image\&#39;s configuration (which includes the digests of layers used by the image).  Note that this digest differs from the &#x60;RepoDigests&#x60; below, which holds digests of image manifests that reference the image.  | [default to undefined]
**ParentId** | **string** | ID of the parent image.  Depending on how the image was created, this field may be empty and is only set for images that were built/created locally. This field is empty if the image was pulled from an image registry.  | [default to undefined]
**RepoTags** | **Array&lt;string&gt;** | List of image names/tags in the local image cache that reference this image.  Multiple image tags can refer to the same image, and this list may be empty if no tags reference the image, in which case the image is \&quot;untagged\&quot;, in which case it can still be referenced by its ID.  | [default to undefined]
**RepoDigests** | **Array&lt;string&gt;** | List of content-addressable digests of locally available image manifests that the image is referenced from. Multiple manifests can refer to the same image.  These digests are usually only available if the image was either pulled from a registry, or if the image was pushed to a registry, which is when the manifest is generated and its digest calculated.  | [default to undefined]
**Created** | **number** | Date and time at which the image was created as a Unix timestamp (number of seconds since EPOCH).  | [default to undefined]
**Size** | **number** | Total size of the image including all layers it is composed of.  | [default to undefined]
**SharedSize** | **number** | Total size of image layers that are shared between this image and other images.  This size is not calculated by default. &#x60;-1&#x60; indicates that the value has not been set / calculated.  | [default to undefined]
**VirtualSize** | **number** | Total size of the image including all layers it is composed of.  Deprecated: this field is omitted in API v1.44, but kept for backward compatibility. Use Size instead. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [default to undefined]
**Containers** | **number** | Number of containers using this image. Includes both stopped and running containers.  &#x60;-1&#x60; indicates that the value has not been set / calculated.  | [default to undefined]
**Manifests** | [**Array&lt;ImageManifestSummary&gt;**](ImageManifestSummary.md) | Manifests is a list of manifests available in this image. It provides a more detailed view of the platform-specific image manifests or other image-attached data like build attestations.  WARNING: This is experimental and may change at any time without any backward compatibility.  | [optional] [default to undefined]
**Descriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ImageSummary } from './api';

const instance: ImageSummary = {
    Id,
    ParentId,
    RepoTags,
    RepoDigests,
    Created,
    Size,
    SharedSize,
    VirtualSize,
    Labels,
    Containers,
    Manifests,
    Descriptor,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
