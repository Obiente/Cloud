# ContainerPidsStats

PidsStats contains Linux-specific stats of a container\'s process-IDs (PIDs).  This type is Linux-specific and omitted for Windows containers. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**current** | **number** | Current is the number of PIDs in the cgroup.  | [optional] [default to undefined]
**limit** | **number** | Limit is the hard limit on the number of pids in the cgroup. A \&quot;Limit\&quot; of 0 means that there is no limit.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerPidsStats } from './api';

const instance: ContainerPidsStats = {
    current,
    limit,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
