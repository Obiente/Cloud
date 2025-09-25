# SwarmSpecCAConfigExternalCAsInner


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Protocol** | **string** | Protocol for communication with the external CA (currently only &#x60;cfssl&#x60; is supported).  | [optional] [default to ProtocolEnum_Cfssl]
**URL** | **string** | URL where certificate signing requests should be sent.  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | An object with key/value pairs that are interpreted as protocol-specific options for the external CA driver.  | [optional] [default to undefined]
**CACert** | **string** | The root CA certificate (in PEM format) this external CA uses to issue TLS certificates (assumed to be to the current swarm root CA certificate if not provided).  | [optional] [default to undefined]

## Example

```typescript
import { SwarmSpecCAConfigExternalCAsInner } from './api';

const instance: SwarmSpecCAConfigExternalCAsInner = {
    Protocol,
    URL,
    Options,
    CACert,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
