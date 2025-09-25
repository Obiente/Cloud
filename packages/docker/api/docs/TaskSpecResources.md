# TaskSpecResources

Resource requirements which apply to each individual container created as part of the service. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Limits** | [**Limit**](Limit.md) |  | [optional] [default to undefined]
**Reservations** | [**ResourceObject**](ResourceObject.md) |  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecResources } from './api';

const instance: TaskSpecResources = {
    Limits,
    Reservations,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
