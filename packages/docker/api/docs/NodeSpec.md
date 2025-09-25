# NodeSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name for the node. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Role** | **string** | Role of the node. | [optional] [default to undefined]
**Availability** | **string** | Availability of the node. | [optional] [default to undefined]

## Example

```typescript
import { NodeSpec } from './api';

const instance: NodeSpec = {
    Name,
    Labels,
    Role,
    Availability,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
