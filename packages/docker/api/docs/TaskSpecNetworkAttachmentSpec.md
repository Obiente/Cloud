# TaskSpecNetworkAttachmentSpec

Read-only spec type for non-swarm containers attached to swarm overlay networks.  <p><br /></p>  > **Note**: ContainerSpec, NetworkAttachmentSpec, and PluginSpec are > mutually exclusive. PluginSpec is only used when the Runtime field > is set to `plugin`. NetworkAttachmentSpec is used when the Runtime > field is set to `attachment`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ContainerID** | **string** | ID of the container represented by this task | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecNetworkAttachmentSpec } from './api';

const instance: TaskSpecNetworkAttachmentSpec = {
    ContainerID,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
