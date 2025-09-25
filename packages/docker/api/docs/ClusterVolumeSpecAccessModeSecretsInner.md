# ClusterVolumeSpecAccessModeSecretsInner

One cluster volume secret entry. Defines a key-value pair that is passed to the plugin. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Key** | **string** | Key is the name of the key of the key-value pair passed to the plugin.  | [optional] [default to undefined]
**Secret** | **string** | Secret is the swarm Secret object from which to read data. This can be a Secret name or ID. The Secret data is retrieved by swarm and used as the value of the key-value pair passed to the plugin.  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolumeSpecAccessModeSecretsInner } from './api';

const instance: ClusterVolumeSpecAccessModeSecretsInner = {
    Key,
    Secret,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
