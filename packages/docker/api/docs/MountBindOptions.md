# MountBindOptions

Optional configuration for the `bind` type.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Propagation** | **string** | A propagation mode with the value &#x60;[r]private&#x60;, &#x60;[r]shared&#x60;, or &#x60;[r]slave&#x60;. | [optional] [default to undefined]
**NonRecursive** | **boolean** | Disable recursive bind mount. | [optional] [default to false]
**CreateMountpoint** | **boolean** | Create mount point on host if missing | [optional] [default to false]
**ReadOnlyNonRecursive** | **boolean** | Make the mount non-recursively read-only, but still leave the mount recursive (unless NonRecursive is set to &#x60;true&#x60; in conjunction).  Added in v1.44, before that version all read-only mounts were non-recursive by default. To match the previous behaviour this will default to &#x60;true&#x60; for clients on versions prior to v1.44.  | [optional] [default to false]
**ReadOnlyForceRecursive** | **boolean** | Raise an error if the mount cannot be made recursively read-only. | [optional] [default to false]

## Example

```typescript
import { MountBindOptions } from './api';

const instance: MountBindOptions = {
    Propagation,
    NonRecursive,
    CreateMountpoint,
    ReadOnlyNonRecursive,
    ReadOnlyForceRecursive,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
