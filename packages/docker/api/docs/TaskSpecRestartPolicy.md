# TaskSpecRestartPolicy

Specification for the restart policy which applies to containers created as part of this service. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Condition** | **string** | Condition for restart. | [optional] [default to undefined]
**Delay** | **number** | Delay between restart attempts. | [optional] [default to undefined]
**MaxAttempts** | **number** | Maximum attempts to restart a given container before giving up (default value is 0, which is ignored).  | [optional] [default to 0]
**Window** | **number** | Windows is the time window used to evaluate the restart policy (default value is 0, which is unbounded).  | [optional] [default to 0]

## Example

```typescript
import { TaskSpecRestartPolicy } from './api';

const instance: TaskSpecRestartPolicy = {
    Condition,
    Delay,
    MaxAttempts,
    Window,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
