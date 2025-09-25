# RestartPolicy

The behavior to apply when the container exits. The default is not to restart.  An ever increasing delay (double the previous delay, starting at 100ms) is added before each restart to prevent flooding the server. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | - Empty string means not to restart - &#x60;no&#x60; Do not automatically restart - &#x60;always&#x60; Always restart - &#x60;unless-stopped&#x60; Restart always except when the user has manually stopped the container - &#x60;on-failure&#x60; Restart only when the container exit code is non-zero  | [optional] [default to undefined]
**MaximumRetryCount** | **number** | If &#x60;on-failure&#x60; is used, the number of times to retry before giving up.  | [optional] [default to undefined]

## Example

```typescript
import { RestartPolicy } from './api';

const instance: RestartPolicy = {
    Name,
    MaximumRetryCount,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
