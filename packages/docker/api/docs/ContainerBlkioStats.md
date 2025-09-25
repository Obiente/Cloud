# ContainerBlkioStats

BlkioStats stores all IO service stats for data read and write.  This type is Linux-specific and holds many fields that are specific to cgroups v1. On a cgroup v2 host, all fields other than `io_service_bytes_recursive` are omitted or `null`.  This type is only populated on Linux and omitted for Windows containers. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**io_service_bytes_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) |  | [optional] [default to undefined]
**io_serviced_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**io_queue_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**io_service_time_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**io_wait_time_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**io_merged_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**io_time_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]
**sectors_recursive** | [**Array&lt;ContainerBlkioStatEntry&gt;**](ContainerBlkioStatEntry.md) | This field is only available when using Linux containers with cgroups v1. It is omitted or &#x60;null&#x60; when using cgroups v2.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerBlkioStats } from './api';

const instance: ContainerBlkioStats = {
    io_service_bytes_recursive,
    io_serviced_recursive,
    io_queue_recursive,
    io_service_time_recursive,
    io_wait_time_recursive,
    io_merged_recursive,
    io_time_recursive,
    sectors_recursive,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
