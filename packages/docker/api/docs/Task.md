# Task


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | The ID of the task. | [optional] [default to undefined]
**Version** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]
**CreatedAt** | **string** |  | [optional] [default to undefined]
**UpdatedAt** | **string** |  | [optional] [default to undefined]
**Name** | **string** | Name of the task. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Spec** | [**TaskSpec**](TaskSpec.md) |  | [optional] [default to undefined]
**ServiceID** | **string** | The ID of the service this task is part of. | [optional] [default to undefined]
**Slot** | **number** |  | [optional] [default to undefined]
**NodeID** | **string** | The ID of the node that this task is on. | [optional] [default to undefined]
**AssignedGenericResources** | [**Array&lt;GenericResourcesInner&gt;**](GenericResourcesInner.md) | User-defined resources can be either Integer resources (e.g, &#x60;SSD&#x3D;3&#x60;) or String resources (e.g, &#x60;GPU&#x3D;UUID1&#x60;).  | [optional] [default to undefined]
**Status** | [**TaskStatus**](TaskStatus.md) |  | [optional] [default to undefined]
**DesiredState** | [**TaskState**](TaskState.md) |  | [optional] [default to undefined]
**JobIteration** | [**ObjectVersion**](ObjectVersion.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Task } from './api';

const instance: Task = {
    ID,
    Version,
    CreatedAt,
    UpdatedAt,
    Name,
    Labels,
    Spec,
    ServiceID,
    Slot,
    NodeID,
    AssignedGenericResources,
    Status,
    DesiredState,
    JobIteration,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
