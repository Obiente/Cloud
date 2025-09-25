# MountTmpfsOptions

Optional configuration for the `tmpfs` type.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SizeBytes** | **number** | The size for the tmpfs mount in bytes. | [optional] [default to undefined]
**Mode** | **number** | The permission mode for the tmpfs mount in an integer. | [optional] [default to undefined]
**Options** | **Array&lt;Array&lt;string&gt;&gt;** | The options to be passed to the tmpfs mount. An array of arrays. Flag options should be provided as 1-length arrays. Other types should be provided as as 2-length arrays, where the first item is the key and the second the value.  | [optional] [default to undefined]

## Example

```typescript
import { MountTmpfsOptions } from './api';

const instance: MountTmpfsOptions = {
    SizeBytes,
    Mode,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
