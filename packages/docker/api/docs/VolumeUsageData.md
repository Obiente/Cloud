# VolumeUsageData

Usage details about the volume. This information is used by the `GET /system/df` endpoint, and omitted in other endpoints. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Size** | **number** | Amount of disk space used by the volume (in bytes). This information is only available for volumes created with the &#x60;\&quot;local\&quot;&#x60; volume driver. For volumes created with other volume drivers, this field is set to &#x60;-1&#x60; (\&quot;not available\&quot;)  | [default to -1]
**RefCount** | **number** | The number of containers referencing this volume. This field is set to &#x60;-1&#x60; if the reference-count is not available.  | [default to -1]

## Example

```typescript
import { VolumeUsageData } from './api';

const instance: VolumeUsageData = {
    Size,
    RefCount,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
