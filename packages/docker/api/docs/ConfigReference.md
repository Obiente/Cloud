# ConfigReference

The config-only network source to provide the configuration for this network. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Network** | **string** | The name of the config-only network that provides the network\&#39;s configuration. The specified network must be an existing config-only network. Only network names are allowed, not network IDs.  | [optional] [default to undefined]

## Example

```typescript
import { ConfigReference } from './api';

const instance: ConfigReference = {
    Network,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
