# SecretCreateRequest


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | User-defined name of the secret. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**Data** | **string** | Data is the data to store as a secret, formatted as a Base64-url-safe-encoded ([RFC 4648](https://tools.ietf.org/html/rfc4648#section-5)) string. It must be empty if the Driver field is set, in which case the data is loaded from an external secret store. The maximum allowed size is 500KB, as defined in [MaxSecretSize](https://pkg.go.dev/github.com/moby/swarmkit/v2@v2.0.0-20250103191802-8c1959736554/api/validation#MaxSecretSize).  This field is only used to _create_ a secret, and is not returned by other endpoints.  | [optional] [default to undefined]
**Driver** | [**Driver**](Driver.md) |  | [optional] [default to undefined]
**Templating** | [**Driver**](Driver.md) |  | [optional] [default to undefined]

## Example

```typescript
import { SecretCreateRequest } from './api';

const instance: SecretCreateRequest = {
    Name,
    Labels,
    Data,
    Driver,
    Templating,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
