# SwarmSpecOrchestration

Orchestration configuration.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**TaskHistoryRetentionLimit** | **number** | The number of historic tasks to keep per instance or node. If negative, never remove completed or failed tasks.  | [optional] [default to undefined]

## Example

```typescript
import { SwarmSpecOrchestration } from './api';

const instance: SwarmSpecOrchestration = {
    TaskHistoryRetentionLimit,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
