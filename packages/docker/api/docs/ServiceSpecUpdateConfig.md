# ServiceSpecUpdateConfig

Specification for the update strategy of the service.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Parallelism** | **number** | Maximum number of tasks to be updated in one iteration (0 means unlimited parallelism).  | [optional] [default to undefined]
**Delay** | **number** | Amount of time between updates, in nanoseconds. | [optional] [default to undefined]
**FailureAction** | **string** | Action to take if an updated task fails to run, or stops running during the update.  | [optional] [default to undefined]
**Monitor** | **number** | Amount of time to monitor each updated task for failures, in nanoseconds.  | [optional] [default to undefined]
**MaxFailureRatio** | **number** | The fraction of tasks that may fail during an update before the failure action is invoked, specified as a floating point number between 0 and 1.  | [optional] [default to undefined]
**Order** | **string** | The order of operations when rolling out an updated task. Either the old task is shut down before the new task is started, or the new task is started before the old task is shut down.  | [optional] [default to undefined]

## Example

```typescript
import { ServiceSpecUpdateConfig } from './api';

const instance: ServiceSpecUpdateConfig = {
    Parallelism,
    Delay,
    FailureAction,
    Monitor,
    MaxFailureRatio,
    Order,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
