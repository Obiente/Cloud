# NodeStatus

NodeStatus represents the status of a node.  It provides the current status of the node, as seen by the manager. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**State** | [**NodeState**](NodeState.md) |  | [optional] [default to undefined]
**Message** | **string** |  | [optional] [default to undefined]
**Addr** | **string** | IP address of the node. | [optional] [default to undefined]

## Example

```typescript
import { NodeStatus } from './api';

const instance: NodeStatus = {
    State,
    Message,
    Addr,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
