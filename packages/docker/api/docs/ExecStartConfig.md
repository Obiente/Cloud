# ExecStartConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Detach** | **boolean** | Detach from the command. | [optional] [default to undefined]
**Tty** | **boolean** | Allocate a pseudo-TTY. | [optional] [default to undefined]
**ConsoleSize** | **Array&lt;number&gt;** | Initial console size, as an &#x60;[height, width]&#x60; array. | [optional] [default to undefined]

## Example

```typescript
import { ExecStartConfig } from './api';

const instance: ExecStartConfig = {
    Detach,
    Tty,
    ConsoleSize,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
