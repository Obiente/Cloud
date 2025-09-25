# TaskSpec

User modifiable task configuration.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**PluginSpec** | [**TaskSpecPluginSpec**](TaskSpecPluginSpec.md) |  | [optional] [default to undefined]
**ContainerSpec** | [**TaskSpecContainerSpec**](TaskSpecContainerSpec.md) |  | [optional] [default to undefined]
**NetworkAttachmentSpec** | [**TaskSpecNetworkAttachmentSpec**](TaskSpecNetworkAttachmentSpec.md) |  | [optional] [default to undefined]
**Resources** | [**TaskSpecResources**](TaskSpecResources.md) |  | [optional] [default to undefined]
**RestartPolicy** | [**TaskSpecRestartPolicy**](TaskSpecRestartPolicy.md) |  | [optional] [default to undefined]
**Placement** | [**TaskSpecPlacement**](TaskSpecPlacement.md) |  | [optional] [default to undefined]
**ForceUpdate** | **number** | A counter that triggers an update even if no relevant parameters have been changed.  | [optional] [default to undefined]
**Runtime** | **string** | Runtime is the type of runtime specified for the task executor.  | [optional] [default to undefined]
**Networks** | [**Array&lt;NetworkAttachmentConfig&gt;**](NetworkAttachmentConfig.md) | Specifies which networks the service should attach to. | [optional] [default to undefined]
**LogDriver** | [**TaskSpecLogDriver**](TaskSpecLogDriver.md) |  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpec } from './api';

const instance: TaskSpec = {
    PluginSpec,
    ContainerSpec,
    NetworkAttachmentSpec,
    Resources,
    RestartPolicy,
    Placement,
    ForceUpdate,
    Runtime,
    Networks,
    LogDriver,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
