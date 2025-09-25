# EngineDescription

EngineDescription provides information about an engine.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EngineVersion** | **string** |  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** |  | [optional] [default to undefined]
**Plugins** | [**Array&lt;EngineDescriptionPluginsInner&gt;**](EngineDescriptionPluginsInner.md) |  | [optional] [default to undefined]

## Example

```typescript
import { EngineDescription } from './api';

const instance: EngineDescription = {
    EngineVersion,
    Labels,
    Plugins,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
