# Limit

An object describing a limit on resources which can be requested by a task. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NanoCPUs** | **number** |  | [optional] [default to undefined]
**MemoryBytes** | **number** |  | [optional] [default to undefined]
**Pids** | **number** | Limits the maximum number of PIDs in the container. Set &#x60;0&#x60; for unlimited.  | [optional] [default to 0]

## Example

```typescript
import { Limit } from './api';

const instance: Limit = {
    NanoCPUs,
    MemoryBytes,
    Pids,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
