# ImageInspect

Information about an image in the local image cache. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** | ID is the content-addressable ID of an image.  This identifier is a content-addressable digest calculated from the image\&#39;s configuration (which includes the digests of layers used by the image).  Note that this digest differs from the &#x60;RepoDigests&#x60; below, which holds digests of image manifests that reference the image.  | [optional] [default to undefined]
**Descriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [optional] [default to undefined]
**Manifests** | [**Array&lt;ImageManifestSummary&gt;**](ImageManifestSummary.md) | Manifests is a list of image manifests available in this image. It provides a more detailed view of the platform-specific image manifests or other image-attached data like build attestations.  Only available if the daemon provides a multi-platform image store and the &#x60;manifests&#x60; option is set in the inspect request.  WARNING: This is experimental and may change at any time without any backward compatibility.  | [optional] [default to undefined]
**RepoTags** | **Array&lt;string&gt;** | List of image names/tags in the local image cache that reference this image.  Multiple image tags can refer to the same image, and this list may be empty if no tags reference the image, in which case the image is \&quot;untagged\&quot;, in which case it can still be referenced by its ID.  | [optional] [default to undefined]
**RepoDigests** | **Array&lt;string&gt;** | List of content-addressable digests of locally available image manifests that the image is referenced from. Multiple manifests can refer to the same image.  These digests are usually only available if the image was either pulled from a registry, or if the image was pushed to a registry, which is when the manifest is generated and its digest calculated.  | [optional] [default to undefined]
**Parent** | **string** | ID of the parent image.  Depending on how the image was created, this field may be empty and is only set for images that were built/created locally. This field is empty if the image was pulled from an image registry.  | [optional] [default to undefined]
**Comment** | **string** | Optional message that was set when committing or importing the image.  | [optional] [default to undefined]
**Created** | **string** | Date and time at which the image was created, formatted in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  This information is only available if present in the image, and omitted otherwise.  | [optional] [default to undefined]
**DockerVersion** | **string** | The version of Docker that was used to build the image.  Depending on how the image was created, this field may be empty.  | [optional] [default to undefined]
**Author** | **string** | Name of the author that was specified when committing the image, or as specified through MAINTAINER (deprecated) in the Dockerfile.  | [optional] [default to undefined]
**Config** | [**ImageConfig**](ImageConfig.md) |  | [optional] [default to undefined]
**Architecture** | **string** | Hardware CPU architecture that the image runs on.  | [optional] [default to undefined]
**Variant** | **string** | CPU architecture variant (presently ARM-only).  | [optional] [default to undefined]
**Os** | **string** | Operating System the image is built to run on.  | [optional] [default to undefined]
**OsVersion** | **string** | Operating System version the image is built to run on (especially for Windows).  | [optional] [default to undefined]
**Size** | **number** | Total size of the image including all layers it is composed of.  | [optional] [default to undefined]
**VirtualSize** | **number** | Total size of the image including all layers it is composed of.  Deprecated: this field is omitted in API v1.44, but kept for backward compatibility. Use Size instead.  | [optional] [default to undefined]
**GraphDriver** | [**DriverData**](DriverData.md) |  | [optional] [default to undefined]
**RootFS** | [**ImageInspectRootFS**](ImageInspectRootFS.md) |  | [optional] [default to undefined]
**Metadata** | [**ImageInspectMetadata**](ImageInspectMetadata.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ImageInspect } from './api';

const instance: ImageInspect = {
    Id,
    Descriptor,
    Manifests,
    RepoTags,
    RepoDigests,
    Parent,
    Comment,
    Created,
    DockerVersion,
    Author,
    Config,
    Architecture,
    Variant,
    Os,
    OsVersion,
    Size,
    VirtualSize,
    GraphDriver,
    RootFS,
    Metadata,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
