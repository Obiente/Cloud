# ContainerMemoryStats

Aggregates all memory stats since container inception on Linux. Windows returns stats for commit and private working set only. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**usage** | **number** | Current &#x60;res_counter&#x60; usage for memory.  This field is Linux-specific and omitted for Windows containers.  | [optional] [default to undefined]
**max_usage** | **number** | Maximum usage ever recorded.  This field is Linux-specific and only supported on cgroups v1. It is omitted when using cgroups v2 and for Windows containers.  | [optional] [default to undefined]
**stats** | **{ [key: string]: number | null; }** | All the stats exported via memory.stat. when using cgroups v2.  This field is Linux-specific and omitted for Windows containers.  | [optional] [default to undefined]
**failcnt** | **number** | Number of times memory usage hits limits.  This field is Linux-specific and only supported on cgroups v1. It is omitted when using cgroups v2 and for Windows containers.  | [optional] [default to undefined]
**limit** | **number** | This field is Linux-specific and omitted for Windows containers.  | [optional] [default to undefined]
**commitbytes** | **number** | Committed bytes.  This field is Windows-specific and omitted for Linux containers.  | [optional] [default to undefined]
**commitpeakbytes** | **number** | Peak committed bytes.  This field is Windows-specific and omitted for Linux containers.  | [optional] [default to undefined]
**privateworkingset** | **number** | Private working set.  This field is Windows-specific and omitted for Linux containers.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerMemoryStats } from './api';

const instance: ContainerMemoryStats = {
    usage,
    max_usage,
    stats,
    failcnt,
    limit,
    commitbytes,
    commitpeakbytes,
    privateworkingset,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
