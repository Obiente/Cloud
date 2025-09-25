# SwarmInfo

Represents generic information about swarm. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeID** | **string** | Unique identifier of for this node in the swarm. | [optional] [default to '']
**NodeAddr** | **string** | IP address at which this node can be reached by other nodes in the swarm.  | [optional] [default to '']
**LocalNodeState** | [**LocalNodeState**](LocalNodeState.md) |  | [optional] [default to undefined]
**ControlAvailable** | **boolean** |  | [optional] [default to false]
**Error** | **string** |  | [optional] [default to '']
**RemoteManagers** | [**Array&lt;PeerNode&gt;**](PeerNode.md) | List of ID\&#39;s and addresses of other managers in the swarm.  | [optional] [default to undefined]
**Nodes** | **number** | Total number of nodes in the swarm. | [optional] [default to undefined]
**Managers** | **number** | Total number of managers in the swarm. | [optional] [default to undefined]
**Cluster** | [**ClusterInfo**](ClusterInfo.md) |  | [optional] [default to undefined]

## Example

```typescript
import { SwarmInfo } from './api';

const instance: SwarmInfo = {
    NodeID,
    NodeAddr,
    LocalNodeState,
    ControlAvailable,
    Error,
    RemoteManagers,
    Nodes,
    Managers,
    Cluster,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
