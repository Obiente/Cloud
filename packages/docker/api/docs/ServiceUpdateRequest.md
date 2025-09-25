# ServiceUpdateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the service. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**TaskTemplate** | [**TaskSpec**](TaskSpec.md) |  | [optional] [default to undefined]
**Mode** | [**ServiceSpecMode**](ServiceSpecMode.md) |  | [optional] [default to undefined]
**UpdateConfig** | [**ServiceSpecUpdateConfig**](ServiceSpecUpdateConfig.md) |  | [optional] [default to undefined]
**RollbackConfig** | [**ServiceSpecRollbackConfig**](ServiceSpecRollbackConfig.md) |  | [optional] [default to undefined]
**Networks** | [**Array&lt;NetworkAttachmentConfig&gt;**](NetworkAttachmentConfig.md) | Specifies which networks the service should attach to.  Deprecated: This field is deprecated since v1.44. The Networks field in TaskSpec should be used instead.  | [optional] [default to undefined]
**EndpointSpec** | [**EndpointSpec**](EndpointSpec.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ServiceUpdateRequest } from './api';

const instance: ServiceUpdateRequest = {
    Name,
    Labels,
    TaskTemplate,
    Mode,
    UpdateConfig,
    RollbackConfig,
    Networks,
    EndpointSpec,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
