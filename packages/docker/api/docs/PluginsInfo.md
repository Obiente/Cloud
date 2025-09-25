# PluginsInfo

Available plugins per type.  <p><br /></p>  > **Note**: Only unmanaged (V1) plugins are included in this list. > V1 plugins are \"lazily\" loaded, and are not returned in this list > if there is no resource using the plugin. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Volume** | **Array&lt;string&gt;** | Names of available volume-drivers, and network-driver plugins. | [optional] [default to undefined]
**Network** | **Array&lt;string&gt;** | Names of available network-drivers, and network-driver plugins. | [optional] [default to undefined]
**Authorization** | **Array&lt;string&gt;** | Names of available authorization plugins. | [optional] [default to undefined]
**Log** | **Array&lt;string&gt;** | Names of available logging-drivers, and logging-driver plugins. | [optional] [default to undefined]

## Example

```typescript
import { PluginsInfo } from './api';

const instance: PluginsInfo = {
    Volume,
    Network,
    Authorization,
    Log,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
