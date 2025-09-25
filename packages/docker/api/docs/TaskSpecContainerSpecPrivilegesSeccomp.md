# TaskSpecContainerSpecPrivilegesSeccomp

Options for configuring seccomp on the container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mode** | **string** |  | [optional] [default to undefined]
**Profile** | **string** | The custom seccomp profile as a json object | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecPrivilegesSeccomp } from './api';

const instance: TaskSpecContainerSpecPrivilegesSeccomp = {
    Mode,
    Profile,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
