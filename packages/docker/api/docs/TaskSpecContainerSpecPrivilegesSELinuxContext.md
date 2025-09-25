# TaskSpecContainerSpecPrivilegesSELinuxContext

SELinux labels of the container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Disable** | **boolean** | Disable SELinux | [optional] [default to undefined]
**User** | **string** | SELinux user label | [optional] [default to undefined]
**Role** | **string** | SELinux role label | [optional] [default to undefined]
**Type** | **string** | SELinux type label | [optional] [default to undefined]
**Level** | **string** | SELinux level label | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecPrivilegesSELinuxContext } from './api';

const instance: TaskSpecContainerSpecPrivilegesSELinuxContext = {
    Disable,
    User,
    Role,
    Type,
    Level,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
