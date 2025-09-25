# Node


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** |  | [optional] [default to undefined]
**Version** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**CreatedAt** | **string** | Date and time at which the node was added to the swarm in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**UpdatedAt** | **string** | Date and time at which the node was last updated in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**Spec** | [**NodeSpec**](NodeSpec.md) |  | [optional] [default to undefined]
**Description** | [**NodeDescription**](NodeDescription.md) |  | [optional] [default to undefined]
**Status** | [**NodeStatus**](NodeStatus.md) |  | [optional] [default to undefined]
**ManagerStatus** | [**ManagerStatus**](ManagerStatus.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Node } from './api';

const instance: Node = {
    ID,
    Version,
    CreatedAt,
    UpdatedAt,
    Spec,
    Description,
    Status,
    ManagerStatus,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
