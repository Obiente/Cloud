# ServiceJobStatus

The status of the service when it is in one of ReplicatedJob or GlobalJob modes. Absent on Replicated and Global mode services. The JobIteration is an ObjectVersion, but unlike the Service\'s version, does not need to be sent with an update request. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**JobIteration** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**LastExecution** | **string** | The last time, as observed by the server, that this job was started.  | [optional] [default to undefined]

## Example

```typescript
import { ServiceJobStatus } from './api';

const instance: ServiceJobStatus = {
    JobIteration,
    LastExecution,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
