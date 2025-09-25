# ServiceSpecMode

Scheduling mode for the service.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Replicated** | [**ServiceSpecModeReplicated**](ServiceSpecModeReplicated.md) |  | [optional] [default to undefined]
**Global** | **object** |  | [optional] [default to undefined]
**ReplicatedJob** | [**ServiceSpecModeReplicatedJob**](ServiceSpecModeReplicatedJob.md) |  | [optional] [default to undefined]
**GlobalJob** | **object** | The mode used for services which run a task to the completed state on each valid node.  | [optional] [default to undefined]

## Example

```typescript
import { ServiceSpecMode } from './api';

const instance: ServiceSpecMode = {
    Replicated,
    Global,
    ReplicatedJob,
    GlobalJob,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
