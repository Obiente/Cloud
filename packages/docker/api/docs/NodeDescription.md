# NodeDescription

NodeDescription encapsulates the properties of the Node as reported by the agent. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Hostname** | **string** |  | [optional] [default to undefined]
**Platform** | [**Platform**](Platform.md) |  | [optional] [default to undefined]
**Resources** | [**ResourceObject**](ResourceObject.md) |  | [optional] [default to undefined]
**Engine** | [**EngineDescription**](EngineDescription.md) |  | [optional] [default to undefined]
**TLSInfo** | [**TLSInfo**](TLSInfo.md) |  | [optional] [default to undefined]

## Example

```typescript
import { NodeDescription } from './api';

const instance: NodeDescription = {
    Hostname,
    Platform,
    Resources,
    Engine,
    TLSInfo,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
