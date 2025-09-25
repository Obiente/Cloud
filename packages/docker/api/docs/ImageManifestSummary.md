# ImageManifestSummary

ImageManifestSummary represents a summary of an image manifest. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | ID is the content-addressable ID of an image and is the same as the digest of the image manifest.  | [default to undefined]
**Descriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [default to undefined]
**Available** | **boolean** | Indicates whether all the child content (image config, layers) is fully available locally. | [default to undefined]
**Size** | [**ImageManifestSummarySize**](ImageManifestSummarySize.md) |  | [default to undefined]
**Kind** | **string** | The kind of the manifest.  kind         | description -------------|----------------------------------------------------------- image        | Image manifest that can be used to start a container. attestation  | Attestation manifest produced by the Buildkit builder for a specific image manifest.  | [default to undefined]
**ImageData** | [**ImageManifestSummaryImageData**](ImageManifestSummaryImageData.md) |  | [optional] [default to undefined]
**AttestationData** | [**ImageManifestSummaryAttestationData**](ImageManifestSummaryAttestationData.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ImageManifestSummary } from './api';

const instance: ImageManifestSummary = {
    ID,
    Descriptor,
    Available,
    Size,
    Kind,
    ImageData,
    AttestationData,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
