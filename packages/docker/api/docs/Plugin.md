# Plugin

A plugin for the Engine API

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** |  | [optional] [default to undefined]
**Name** | **string** |  | [default to undefined]
**Enabled** | **boolean** | True if the plugin is running. False if the plugin is not running, only installed. | [default to undefined]
**Settings** | [**PluginSettings**](PluginSettings.md) |  | [default to undefined]
**PluginReference** | **string** | plugin remote reference used to push/pull the plugin | [optional] [default to undefined]
**Config** | [**PluginConfig**](PluginConfig.md) |  | [default to undefined]

## Example

```typescript
import { Plugin } from './api';

const instance: Plugin = {
    Id,
    Name,
    Enabled,
    Settings,
    PluginReference,
    Config,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
