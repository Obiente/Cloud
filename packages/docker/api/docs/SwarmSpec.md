# SwarmSpec

User modifiable swarm configuration.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the swarm. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Orchestration** | [**SwarmSpecOrchestration**](SwarmSpecOrchestration.md) |  | [optional] [default to undefined]
**Raft** | [**SwarmSpecRaft**](SwarmSpecRaft.md) |  | [optional] [default to undefined]
**Dispatcher** | [**SwarmSpecDispatcher**](SwarmSpecDispatcher.md) |  | [optional] [default to undefined]
**CAConfig** | [**SwarmSpecCAConfig**](SwarmSpecCAConfig.md) |  | [optional] [default to undefined]
**EncryptionConfig** | [**SwarmSpecEncryptionConfig**](SwarmSpecEncryptionConfig.md) |  | [optional] [default to undefined]
**TaskDefaults** | [**SwarmSpecTaskDefaults**](SwarmSpecTaskDefaults.md) |  | [optional] [default to undefined]

## Example

```typescript
import { SwarmSpec } from './api';

const instance: SwarmSpec = {
    Name,
    Labels,
    Orchestration,
    Raft,
    Dispatcher,
    CAConfig,
    EncryptionConfig,
    TaskDefaults,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
