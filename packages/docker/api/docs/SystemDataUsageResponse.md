# SystemDataUsageResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LayersSize** | **number** |  | [optional] [default to undefined]
**Images** | [**Array&lt;ImageSummary&gt;**](ImageSummary.md) |  | [optional] [default to undefined]
**Containers** | [**Array&lt;ContainerSummary&gt;**](ContainerSummary.md) |  | [optional] [default to undefined]
**Volumes** | [**Array&lt;Volume&gt;**](Volume.md) |  | [optional] [default to undefined]
**BuildCache** | [**Array&lt;BuildCache&gt;**](BuildCache.md) |  | [optional] [default to undefined]

## Example

```typescript
import { SystemDataUsageResponse } from './api';

const instance: SystemDataUsageResponse = {
    LayersSize,
    Images,
    Containers,
    Volumes,
    BuildCache,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
