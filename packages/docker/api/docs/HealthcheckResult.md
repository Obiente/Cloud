# HealthcheckResult

HealthcheckResult stores information about a single run of a healthcheck probe 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Start** | **string** | Date and time at which this check started in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**End** | **string** | Date and time at which this check ended in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**ExitCode** | **number** | ExitCode meanings:  - &#x60;0&#x60; healthy - &#x60;1&#x60; unhealthy - &#x60;2&#x60; reserved (considered unhealthy) - other values: error running probe  | [optional] [default to undefined]
**Output** | **string** | Output from last check | [optional] [default to undefined]

## Example

```typescript
import { HealthcheckResult } from './api';

const instance: HealthcheckResult = {
    Start,
    End,
    ExitCode,
    Output,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
