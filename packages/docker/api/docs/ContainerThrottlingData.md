# ContainerThrottlingData

CPU throttling stats of the container.  This type is Linux-specific and omitted for Windows containers. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**periods** | **number** | Number of periods with throttling active.  | [optional] [default to undefined]
**throttled_periods** | **number** | Number of periods when the container hit its throttling limit.  | [optional] [default to undefined]
**throttled_time** | **number** | Aggregated time (in nanoseconds) the container was throttled for.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerThrottlingData } from './api';

const instance: ContainerThrottlingData = {
    periods,
    throttled_periods,
    throttled_time,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
