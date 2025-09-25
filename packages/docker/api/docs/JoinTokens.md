# JoinTokens

JoinTokens contains the tokens workers and managers need to join the swarm. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Worker** | **string** | The token workers can use to join the swarm.  | [optional] [default to undefined]
**Manager** | **string** | The token managers can use to join the swarm.  | [optional] [default to undefined]

## Example

```typescript
import { JoinTokens } from './api';

const instance: JoinTokens = {
    Worker,
    Manager,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
