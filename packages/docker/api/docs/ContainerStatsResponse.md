# ContainerStatsResponse

Statistics sample for a container. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **string** | Name of the container | [optional] [default to undefined]
**id** | **string** | ID of the container | [optional] [default to undefined]
**read** | **string** | Date and time at which this sample was collected. The value is formatted as [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) with nano-seconds.  | [optional] [default to undefined]
**preread** | **string** | Date and time at which this first sample was collected. This field is not propagated if the \&quot;one-shot\&quot; option is set. If the \&quot;one-shot\&quot; option is set, this field may be omitted, empty, or set to a default date (&#x60;0001-01-01T00:00:00Z&#x60;).  The value is formatted as [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) with nano-seconds.  | [optional] [default to undefined]
**pids_stats** | [**ContainerPidsStats**](ContainerPidsStats.md) |  | [optional] [default to undefined]
**blkio_stats** | [**ContainerBlkioStats**](ContainerBlkioStats.md) |  | [optional] [default to undefined]
**num_procs** | **number** | The number of processors on the system.  This field is Windows-specific and always zero for Linux containers.  | [optional] [default to undefined]
**storage_stats** | [**ContainerStorageStats**](ContainerStorageStats.md) |  | [optional] [default to undefined]
**cpu_stats** | [**ContainerCPUStats**](ContainerCPUStats.md) |  | [optional] [default to undefined]
**precpu_stats** | [**ContainerCPUStats**](ContainerCPUStats.md) |  | [optional] [default to undefined]
**memory_stats** | [**ContainerMemoryStats**](ContainerMemoryStats.md) |  | [optional] [default to undefined]
**networks** | **object** | Network statistics for the container per interface.  This field is omitted if the container has no networking enabled.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerStatsResponse } from './api';

const instance: ContainerStatsResponse = {
    name,
    id,
    read,
    preread,
    pids_stats,
    blkio_stats,
    num_procs,
    storage_stats,
    cpu_stats,
    precpu_stats,
    memory_stats,
    networks,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
