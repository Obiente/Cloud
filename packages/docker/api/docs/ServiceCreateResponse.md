# ServiceCreateResponse

contains the information returned to a client on the creation of a new service. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | The ID of the created service. | [optional] [default to undefined]
**Warnings** | **Array&lt;string&gt;** | Optional warning message.  FIXME(thaJeztah): this should have \&quot;omitempty\&quot; in the generated type.  | [optional] [default to undefined]

## Example

```typescript
import { ServiceCreateResponse } from './api';

const instance: ServiceCreateResponse = {
    ID,
    Warnings,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
