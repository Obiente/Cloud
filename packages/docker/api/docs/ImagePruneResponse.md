# ImagePruneResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ImagesDeleted** | [**Array&lt;ImageDeleteResponseItem&gt;**](ImageDeleteResponseItem.md) | Images that were deleted | [optional] [default to undefined]
**SpaceReclaimed** | **number** | Disk space reclaimed in bytes | [optional] [default to undefined]

## Example

```typescript
import { ImagePruneResponse } from './api';

const instance: ImagePruneResponse = {
    ImagesDeleted,
    SpaceReclaimed,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
