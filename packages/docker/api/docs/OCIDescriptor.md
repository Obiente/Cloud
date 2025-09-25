# OCIDescriptor

A descriptor struct containing digest, media type, and size, as defined in the [OCI Content Descriptors Specification](https://github.com/opencontainers/image-spec/blob/v1.0.1/descriptor.md). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mediaType** | **string** | The media type of the object this schema refers to.  | [optional] [default to undefined]
**digest** | **string** | The digest of the targeted content.  | [optional] [default to undefined]
**size** | **number** | The size in bytes of the blob.  | [optional] [default to undefined]
**urls** | **Array&lt;string&gt;** | List of URLs from which this object MAY be downloaded. | [optional] [default to undefined]
**annotations** | **{ [key: string]: string; }** | Arbitrary metadata relating to the targeted content. | [optional] [default to undefined]
**data** | **string** | Data is an embedding of the targeted content. This is encoded as a base64 string when marshalled to JSON (automatically, by encoding/json). If present, Data can be used directly to avoid fetching the targeted content. | [optional] [default to undefined]
**platform** | [**OCIPlatform**](OCIPlatform.md) |  | [optional] [default to undefined]
**artifactType** | **string** | ArtifactType is the IANA media type of this artifact. | [optional] [default to undefined]

## Example

```typescript
import { OCIDescriptor } from './api';

const instance: OCIDescriptor = {
    mediaType,
    digest,
    size,
    urls,
    annotations,
    data,
    platform,
    artifactType,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
