# CreateImageInfo


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **string** |  | [optional] [default to undefined]
**error** | **string** | errors encountered during the operation.   &gt; **Deprecated**: This field is deprecated since API v1.4, and will be omitted in a future API version. Use the information in errorDetail instead. | [optional] [default to undefined]
**errorDetail** | [**ErrorDetail**](ErrorDetail.md) |  | [optional] [default to undefined]
**status** | **string** |  | [optional] [default to undefined]
**progress** | **string** | Progress is a pre-formatted presentation of progressDetail.   &gt; **Deprecated**: This field is deprecated since API v1.8, and will be omitted in a future API version. Use the information in progressDetail instead. | [optional] [default to undefined]
**progressDetail** | [**ProgressDetail**](ProgressDetail.md) |  | [optional] [default to undefined]

## Example

```typescript
import { CreateImageInfo } from './api';

const instance: CreateImageInfo = {
    id,
    error,
    errorDetail,
    status,
    progress,
    progressDetail,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
