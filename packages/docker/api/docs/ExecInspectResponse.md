# ExecInspectResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CanRemove** | **boolean** |  | [optional] [default to undefined]
**DetachKeys** | **string** |  | [optional] [default to undefined]
**ID** | **string** |  | [optional] [default to undefined]
**Running** | **boolean** |  | [optional] [default to undefined]
**ExitCode** | **number** |  | [optional] [default to undefined]
**ProcessConfig** | [**ProcessConfig**](ProcessConfig.md) |  | [optional] [default to undefined]
**OpenStdin** | **boolean** |  | [optional] [default to undefined]
**OpenStderr** | **boolean** |  | [optional] [default to undefined]
**OpenStdout** | **boolean** |  | [optional] [default to undefined]
**ContainerID** | **string** |  | [optional] [default to undefined]
**Pid** | **number** | The system process ID for the exec process. | [optional] [default to undefined]

## Example

```typescript
import { ExecInspectResponse } from './api';

const instance: ExecInspectResponse = {
    CanRemove,
    DetachKeys,
    ID,
    Running,
    ExitCode,
    ProcessConfig,
    OpenStdin,
    OpenStderr,
    OpenStdout,
    ContainerID,
    Pid,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
