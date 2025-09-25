# HostConfigAllOfLogConfig

The logging configuration for this container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type** | **string** | Name of the logging driver used for the container or \&quot;none\&quot; if logging is disabled. | [optional] [default to undefined]
**Config** | **{ [key: string]: string; }** | Driver-specific configuration options for the logging driver. | [optional] [default to undefined]

## Example

```typescript
import { HostConfigAllOfLogConfig } from './api';

const instance: HostConfigAllOfLogConfig = {
    Type,
    Config,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
