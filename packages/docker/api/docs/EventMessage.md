# EventMessage

EventMessage represents the information an event contains. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type** | **string** | The type of object emitting the event | [optional] [default to undefined]
**Action** | **string** | The type of event | [optional] [default to undefined]
**Actor** | [**EventActor**](EventActor.md) |  | [optional] [default to undefined]
**scope** | **string** | Scope of the event. Engine events are &#x60;local&#x60; scope. Cluster (Swarm) events are &#x60;swarm&#x60; scope.  | [optional] [default to undefined]
**time** | **number** | Timestamp of event | [optional] [default to undefined]
**timeNano** | **number** | Timestamp of event, with nanosecond accuracy | [optional] [default to undefined]

## Example

```typescript
import { EventMessage } from './api';

const instance: EventMessage = {
    Type,
    Action,
    Actor,
    scope,
    time,
    timeNano,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
