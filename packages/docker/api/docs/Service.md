# Service


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** |  | [optional] [default to undefined]
**Version** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**CreatedAt** | **string** |  | [optional] [default to undefined]
**UpdatedAt** | **string** |  | [optional] [default to undefined]
**Spec** | [**ServiceSpec**](ServiceSpec.md) |  | [optional] [default to undefined]
**Endpoint** | [**ServiceEndpoint**](ServiceEndpoint.md) |  | [optional] [default to undefined]
**UpdateStatus** | [**ServiceUpdateStatus**](ServiceUpdateStatus.md) |  | [optional] [default to undefined]
**ServiceStatus** | [**ServiceServiceStatus**](ServiceServiceStatus.md) |  | [optional] [default to undefined]
**JobStatus** | [**ServiceJobStatus**](ServiceJobStatus.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Service } from './api';

const instance: Service = {
    ID,
    Version,
    CreatedAt,
    UpdatedAt,
    Spec,
    Endpoint,
    UpdateStatus,
    ServiceStatus,
    JobStatus,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
