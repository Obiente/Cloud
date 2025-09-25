# ServiceSpecModeReplicatedJob

The mode used for services with a finite number of tasks that run to a completed state. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MaxConcurrent** | **number** | The maximum number of replicas to run simultaneously.  | [optional] [default to 1]
**TotalCompletions** | **number** | The total number of replicas desired to reach the Completed state. If unset, will default to the value of &#x60;MaxConcurrent&#x60;  | [optional] [default to undefined]

## Example

```typescript
import { ServiceSpecModeReplicatedJob } from './api';

const instance: ServiceSpecModeReplicatedJob = {
    MaxConcurrent,
    TotalCompletions,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
