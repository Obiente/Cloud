# FilesystemChange

Change in the container\'s filesystem. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Path** | **string** | Path to file or directory that has changed.  | [default to undefined]
**Kind** | [**ChangeType**](ChangeType.md) |  | [default to undefined]

## Example

```typescript
import { FilesystemChange } from './api';

const instance: FilesystemChange = {
    Path,
    Kind,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
