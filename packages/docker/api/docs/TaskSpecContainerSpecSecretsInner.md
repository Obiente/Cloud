# TaskSpecContainerSpecSecretsInner


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**File** | [**TaskSpecContainerSpecSecretsInnerFile**](TaskSpecContainerSpecSecretsInnerFile.md) |  | [optional] [default to undefined]
**SecretID** | **string** | SecretID represents the ID of the specific secret that we\&#39;re referencing.  | [optional] [default to undefined]
**SecretName** | **string** | SecretName is the name of the secret that this references, but this is just provided for lookup/display purposes. The secret in the reference will be identified by its ID.  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecSecretsInner } from './api';

const instance: TaskSpecContainerSpecSecretsInner = {
    File,
    SecretID,
    SecretName,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
