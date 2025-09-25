# PluginConfig

The config of a plugin.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DockerVersion** | **string** | Docker Version used to create the plugin | [optional] [default to undefined]
**Description** | **string** |  | [default to undefined]
**Documentation** | **string** |  | [default to undefined]
**Interface** | [**PluginConfigInterface**](PluginConfigInterface.md) |  | [default to undefined]
**Entrypoint** | **Array&lt;string&gt;** |  | [default to undefined]
**WorkDir** | **string** |  | [default to undefined]
**User** | [**PluginConfigUser**](PluginConfigUser.md) |  | [optional] [default to undefined]
**Network** | [**PluginConfigNetwork**](PluginConfigNetwork.md) |  | [default to undefined]
**Linux** | [**PluginConfigLinux**](PluginConfigLinux.md) |  | [default to undefined]
**PropagatedMount** | **string** |  | [default to undefined]
**IpcHost** | **boolean** |  | [default to undefined]
**PidHost** | **boolean** |  | [default to undefined]
**Mounts** | [**Array&lt;PluginMount&gt;**](PluginMount.md) |  | [default to undefined]
**Env** | [**Array&lt;PluginEnv&gt;**](PluginEnv.md) |  | [default to undefined]
**Args** | [**PluginConfigArgs**](PluginConfigArgs.md) |  | [default to undefined]
**rootfs** | [**PluginConfigRootfs**](PluginConfigRootfs.md) |  | [optional] [default to undefined]

## Example

```typescript
import { PluginConfig } from './api';

const instance: PluginConfig = {
    DockerVersion,
    Description,
    Documentation,
    Interface,
    Entrypoint,
    WorkDir,
    User,
    Network,
    Linux,
    PropagatedMount,
    IpcHost,
    PidHost,
    Mounts,
    Env,
    Args,
    rootfs,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
