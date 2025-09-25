# EventActor

Actor describes something that generates events, like a container, network, or a volume. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | The ID of the object emitting the event | [optional] [default to undefined]
**Attributes** | **{ [key: string]: string; }** | Various key/value attributes of the object, depending on its type.  | [optional] [default to undefined]

## Example

```typescript
import { EventActor } from './api';

const instance: EventActor = {
    ID,
    Attributes,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
