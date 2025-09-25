# PluginSettings

Settings that can be modified by users.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mounts** | [**Array&lt;PluginMount&gt;**](PluginMount.md) |  | [default to undefined]
**Env** | **Array&lt;string&gt;** |  | [default to undefined]
**Args** | **Array&lt;string&gt;** |  | [default to undefined]
**Devices** | [**Array&lt;PluginDevice&gt;**](PluginDevice.md) |  | [default to undefined]

## Example

```typescript
import { PluginSettings } from './api';

const instance: PluginSettings = {
    Mounts,
    Env,
    Args,
    Devices,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
