# ImageManifestSummaryAttestationData

The image data for the attestation manifest. This field is only populated when Kind is \"attestation\". 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**For** | **string** | The digest of the image manifest that this attestation is for.  | [default to undefined]

## Example

```typescript
import { ImageManifestSummaryAttestationData } from './api';

const instance: ImageManifestSummaryAttestationData = {
    For,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
