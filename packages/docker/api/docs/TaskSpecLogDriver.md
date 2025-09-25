# TaskSpecLogDriver

Specifies the log driver to use for tasks created from this spec. If not present, the default one for the swarm will be used, finally falling back to the engine default if not specified. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** |  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecLogDriver } from './api';

const instance: TaskSpecLogDriver = {
    Name,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
