# ContainerCPUStats

CPU related info of the container 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cpu_usage** | [**ContainerCPUUsage**](ContainerCPUUsage.md) |  | [optional] [default to undefined]
**system_cpu_usage** | **number** | System Usage.  This field is Linux-specific and omitted for Windows containers.  | [optional] [default to undefined]
**online_cpus** | **number** | Number of online CPUs.  This field is Linux-specific and omitted for Windows containers.  | [optional] [default to undefined]
**throttling_data** | [**ContainerThrottlingData**](ContainerThrottlingData.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerCPUStats } from './api';

const instance: ContainerCPUStats = {
    cpu_usage,
    system_cpu_usage,
    online_cpus,
    throttling_data,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
