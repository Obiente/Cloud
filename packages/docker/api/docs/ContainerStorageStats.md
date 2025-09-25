# ContainerStorageStats

StorageStats is the disk I/O stats for read/write on Windows.  This type is Windows-specific and omitted for Linux containers. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**read_count_normalized** | **number** |  | [optional] [default to undefined]
**read_size_bytes** | **number** |  | [optional] [default to undefined]
**write_count_normalized** | **number** |  | [optional] [default to undefined]
**write_size_bytes** | **number** |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerStorageStats } from './api';

const instance: ContainerStorageStats = {
    read_count_normalized,
    read_size_bytes,
    write_count_normalized,
    write_size_bytes,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
