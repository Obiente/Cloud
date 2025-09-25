# Health

Health stores information about the container\'s healthcheck results. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** | Status is one of &#x60;none&#x60;, &#x60;starting&#x60;, &#x60;healthy&#x60; or &#x60;unhealthy&#x60;  - \&quot;none\&quot;      Indicates there is no healthcheck - \&quot;starting\&quot;  Starting indicates that the container is not yet ready - \&quot;healthy\&quot;   Healthy indicates that the container is running correctly - \&quot;unhealthy\&quot; Unhealthy indicates that the container has a problem  | [optional] [default to undefined]
**FailingStreak** | **number** | FailingStreak is the number of consecutive failures | [optional] [default to undefined]
**Log** | [**Array&lt;HealthcheckResult&gt;**](HealthcheckResult.md) | Log contains the last few results (oldest first)  | [optional] [default to undefined]

## Example

```typescript
import { Health } from './api';

const instance: Health = {
    Status,
    FailingStreak,
    Log,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
