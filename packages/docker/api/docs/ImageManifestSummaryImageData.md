# ImageManifestSummaryImageData

The image data for the image manifest. This field is only populated when Kind is \"image\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Platform** | [**OCIPlatform**](OCIPlatform.md) |  | [default to undefined]
**Containers** | **Array&lt;string&gt;** | The IDs of the containers that are using this image.  | [default to undefined]
**Size** | [**ImageManifestSummaryImageDataSize**](ImageManifestSummaryImageDataSize.md) |  | [default to undefined]

## Example

```typescript
import { ImageManifestSummaryImageData } from './api';

const instance: ImageManifestSummaryImageData = {
    Platform,
    Containers,
    Size,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
