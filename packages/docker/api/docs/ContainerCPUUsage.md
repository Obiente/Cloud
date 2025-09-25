# ContainerCPUUsage

All CPU stats aggregated since container inception. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**total_usage** | **number** | Total CPU time consumed in nanoseconds (Linux) or 100\&#39;s of nanoseconds (Windows).  | [optional] [default to undefined]
**percpu_usage** | **Array&lt;number&gt;** | Total CPU time (in nanoseconds) consumed per core (Linux).  This field is Linux-specific when using cgroups v1. It is omitted when using cgroups v2 and Windows containers.  | [optional] [default to undefined]
**usage_in_kernelmode** | **number** | Time (in nanoseconds) spent by tasks of the cgroup in kernel mode (Linux), or time spent (in 100\&#39;s of nanoseconds) by all container processes in kernel mode (Windows).  Not populated for Windows containers using Hyper-V isolation.  | [optional] [default to undefined]
**usage_in_usermode** | **number** | Time (in nanoseconds) spent by tasks of the cgroup in user mode (Linux), or time spent (in 100\&#39;s of nanoseconds) by all container processes in kernel mode (Windows).  Not populated for Windows containers using Hyper-V isolation.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerCPUUsage } from './api';

const instance: ContainerCPUUsage = {
    total_usage,
    percpu_usage,
    usage_in_kernelmode,
    usage_in_usermode,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
