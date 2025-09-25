# SwarmSpecTaskDefaultsLogDriver

The log driver to use for tasks created in the orchestrator if unspecified by a service.  Updating this value only affects new tasks. Existing tasks continue to use their previously configured log driver until recreated. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | The log driver to use as a default for new tasks.  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | Driver-specific options for the selected log driver, specified as key/value pairs.  | [optional] [default to undefined]

## Example

```typescript
import { SwarmSpecTaskDefaultsLogDriver } from './api';

const instance: SwarmSpecTaskDefaultsLogDriver = {
    Name,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
