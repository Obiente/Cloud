# TaskStatus

represents the status of a task.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Timestamp** | **string** |  | [optional] [default to undefined]
**State** | [**TaskState**](TaskState.md) |  | [optional] [default to undefined]
**Message** | **string** |  | [optional] [default to undefined]
**Err** | **string** |  | [optional] [default to undefined]
**ContainerStatus** | [**ContainerStatus**](ContainerStatus.md) |  | [optional] [default to undefined]
**PortStatus** | [**PortStatus**](PortStatus.md) |  | [optional] [default to undefined]

## Example

```typescript
import { TaskStatus } from './api';

const instance: TaskStatus = {
    Timestamp,
    State,
    Message,
    Err,
    ContainerStatus,
    PortStatus,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
