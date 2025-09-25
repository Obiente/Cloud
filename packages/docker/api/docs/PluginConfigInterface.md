# PluginConfigInterface

The interface between Docker and the plugin

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Types** | [**Array&lt;PluginInterfaceType&gt;**](PluginInterfaceType.md) |  | [default to undefined]
**Socket** | **string** |  | [default to undefined]
**ProtocolScheme** | **string** | Protocol to use for clients connecting to the plugin. | [optional] [default to undefined]

## Example

```typescript
import { PluginConfigInterface } from './api';

const instance: PluginConfigInterface = {
    Types,
    Socket,
    ProtocolScheme,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
