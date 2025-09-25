# ConfigCreateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | User-defined name of the config. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Data** | **string** | Data is the data to store as a config, formatted as a Base64-url-safe-encoded ([RFC 4648](https://tools.ietf.org/html/rfc4648#section-5)) string. The maximum allowed size is 1000KB, as defined in [MaxConfigSize](https://pkg.go.dev/github.com/moby/swarmkit/v2@v2.0.0-20250103191802-8c1959736554/manager/controlapi#MaxConfigSize).  | [optional] [default to undefined]
**Templating** | [**Driver**](Driver.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ConfigCreateRequest } from './api';

const instance: ConfigCreateRequest = {
    Name,
    Labels,
    Data,
    Templating,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
