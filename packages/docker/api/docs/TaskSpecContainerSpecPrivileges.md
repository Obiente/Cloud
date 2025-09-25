# TaskSpecContainerSpecPrivileges

Security options for the container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CredentialSpec** | [**TaskSpecContainerSpecPrivilegesCredentialSpec**](TaskSpecContainerSpecPrivilegesCredentialSpec.md) |  | [optional] [default to undefined]
**SELinuxContext** | [**TaskSpecContainerSpecPrivilegesSELinuxContext**](TaskSpecContainerSpecPrivilegesSELinuxContext.md) |  | [optional] [default to undefined]
**Seccomp** | [**TaskSpecContainerSpecPrivilegesSeccomp**](TaskSpecContainerSpecPrivilegesSeccomp.md) |  | [optional] [default to undefined]
**AppArmor** | [**TaskSpecContainerSpecPrivilegesAppArmor**](TaskSpecContainerSpecPrivilegesAppArmor.md) |  | [optional] [default to undefined]
**NoNewPrivileges** | **boolean** | Configuration of the no_new_privs bit in the container | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecPrivileges } from './api';

const instance: TaskSpecContainerSpecPrivileges = {
    CredentialSpec,
    SELinuxContext,
    Seccomp,
    AppArmor,
    NoNewPrivileges,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
