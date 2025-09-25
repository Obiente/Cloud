# VolumeListResponse

Volume list response

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Volumes** | [**Array&lt;Volume&gt;**](Volume.md) | List of volumes | [optional] [default to undefined]
**Warnings** | **Array&lt;string&gt;** | Warnings that occurred when fetching the list of volumes.  | [optional] [default to undefined]

## Example

```typescript
import { VolumeListResponse } from './api';

const instance: VolumeListResponse = {
    Volumes,
    Warnings,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
