# TaskSpecPluginSpec

Plugin spec for the service.  *(Experimental release only.)*  <p><br /></p>  > **Note**: ContainerSpec, NetworkAttachmentSpec, and PluginSpec are > mutually exclusive. PluginSpec is only used when the Runtime field > is set to `plugin`. NetworkAttachmentSpec is used when the Runtime > field is set to `attachment`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | The name or \&#39;alias\&#39; to use for the plugin. | [optional] [default to undefined]
**Remote** | **string** | The plugin image reference to use. | [optional] [default to undefined]
**Disabled** | **boolean** | Disable the plugin once scheduled. | [optional] [default to undefined]
**PluginPrivilege** | [**Array&lt;PluginPrivilege&gt;**](PluginPrivilege.md) |  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecPluginSpec } from './api';

const instance: TaskSpecPluginSpec = {
    Name,
    Remote,
    Disabled,
    PluginPrivilege,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
