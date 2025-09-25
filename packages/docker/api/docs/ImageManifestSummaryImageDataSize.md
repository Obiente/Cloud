# ImageManifestSummaryImageDataSize


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Unpacked** | **number** | Unpacked is the size (in bytes) of the locally unpacked (uncompressed) image content that\&#39;s directly usable by the containers running this image. It\&#39;s independent of the distributable content - e.g. the image might still have an unpacked data that\&#39;s still used by some container even when the distributable/compressed content is already gone.  | [default to undefined]

## Example

```typescript
import { ImageManifestSummaryImageDataSize } from './api';

const instance: ImageManifestSummaryImageDataSize = {
    Unpacked,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
