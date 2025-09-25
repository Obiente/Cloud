# TaskSpecContainerSpecPrivilegesCredentialSpec

CredentialSpec for managed service account (Windows only)

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Config** | **string** | Load credential spec from a Swarm Config with the given ID. The specified config must also be present in the Configs field with the Runtime property set.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;   &gt; **Note**: &#x60;CredentialSpec.File&#x60;, &#x60;CredentialSpec.Registry&#x60;, &gt; and &#x60;CredentialSpec.Config&#x60; are mutually exclusive.  | [optional] [default to undefined]
**File** | **string** | Load credential spec from this file. The file is read by the daemon, and must be present in the &#x60;CredentialSpecs&#x60; subdirectory in the docker data directory, which defaults to &#x60;C:\\ProgramData\\Docker\\&#x60; on Windows.  For example, specifying &#x60;spec.json&#x60; loads &#x60;C:\\ProgramData\\Docker\\CredentialSpecs\\spec.json&#x60;.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Note**: &#x60;CredentialSpec.File&#x60;, &#x60;CredentialSpec.Registry&#x60;, &gt; and &#x60;CredentialSpec.Config&#x60; are mutually exclusive.  | [optional] [default to undefined]
**Registry** | **string** | Load credential spec from this value in the Windows registry. The specified registry value must be located in:  &#x60;HKLM\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\Virtualization\\Containers\\CredentialSpecs&#x60;  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;   &gt; **Note**: &#x60;CredentialSpec.File&#x60;, &#x60;CredentialSpec.Registry&#x60;, &gt; and &#x60;CredentialSpec.Config&#x60; are mutually exclusive.  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecPrivilegesCredentialSpec } from './api';

const instance: TaskSpecContainerSpecPrivilegesCredentialSpec = {
    Config,
    File,
    Registry,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
