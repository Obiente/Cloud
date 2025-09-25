# ContainerState

ContainerState stores container\'s running state. It\'s part of ContainerJSONBase and will be returned by the \"inspect\" command. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** | String representation of the container state. Can be one of \&quot;created\&quot;, \&quot;running\&quot;, \&quot;paused\&quot;, \&quot;restarting\&quot;, \&quot;removing\&quot;, \&quot;exited\&quot;, or \&quot;dead\&quot;.  | [optional] [default to undefined]
**Running** | **boolean** | Whether this container is running.  Note that a running container can be _paused_. The &#x60;Running&#x60; and &#x60;Paused&#x60; booleans are not mutually exclusive:  When pausing a container (on Linux), the freezer cgroup is used to suspend all processes in the container. Freezing the process requires the process to be running. As a result, paused containers are both &#x60;Running&#x60; _and_ &#x60;Paused&#x60;.  Use the &#x60;Status&#x60; field instead to determine if a container\&#39;s state is \&quot;running\&quot;.  | [optional] [default to undefined]
**Paused** | **boolean** | Whether this container is paused. | [optional] [default to undefined]
**Restarting** | **boolean** | Whether this container is restarting. | [optional] [default to undefined]
**OOMKilled** | **boolean** | Whether a process within this container has been killed because it ran out of memory since the container was last started.  | [optional] [default to undefined]
**Dead** | **boolean** |  | [optional] [default to undefined]
**Pid** | **number** | The process ID of this container | [optional] [default to undefined]
**ExitCode** | **number** | The last exit code of this container | [optional] [default to undefined]
**Error** | **string** |  | [optional] [default to undefined]
**StartedAt** | **string** | The time when this container was last started. | [optional] [default to undefined]
**FinishedAt** | **string** | The time when this container last exited. | [optional] [default to undefined]
**Health** | [**Health**](Health.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerState } from './api';

const instance: ContainerState = {
    Status,
    Running,
    Paused,
    Restarting,
    OOMKilled,
    Dead,
    Pid,
    ExitCode,
    Error,
    StartedAt,
    FinishedAt,
    Health,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
