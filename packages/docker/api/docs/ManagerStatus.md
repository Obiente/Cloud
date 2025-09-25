# ManagerStatus

ManagerStatus represents the status of a manager.  It provides the current status of a node\'s manager component, if the node is a manager. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Leader** | **boolean** |  | [optional] [default to false]
**Reachability** | [**Reachability**](Reachability.md) |  | [optional] [default to undefined]
**Addr** | **string** | The IP address and port at which the manager is reachable.  | [optional] [default to undefined]

## Example

```typescript
import { ManagerStatus } from './api';

const instance: ManagerStatus = {
    Leader,
    Reachability,
    Addr,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
