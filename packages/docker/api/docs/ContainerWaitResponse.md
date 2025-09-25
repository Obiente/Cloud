# ContainerWaitResponse

OK response to ContainerWait operation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**StatusCode** | **number** | Exit code of the container | [default to undefined]
**Error** | [**ContainerWaitExitError**](ContainerWaitExitError.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerWaitResponse } from './api';

const instance: ContainerWaitResponse = {
    StatusCode,
    Error,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
