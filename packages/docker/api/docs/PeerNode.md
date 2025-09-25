# PeerNode

Represents a peer-node in the swarm

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeID** | **string** | Unique identifier of for this node in the swarm. | [optional] [default to undefined]
**Addr** | **string** | IP address and ports at which this node can be reached.  | [optional] [default to undefined]

## Example

```typescript
import { PeerNode } from './api';

const instance: PeerNode = {
    NodeID,
    Addr,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
