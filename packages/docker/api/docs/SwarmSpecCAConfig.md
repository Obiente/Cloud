# SwarmSpecCAConfig

CA configuration.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeCertExpiry** | **number** | The duration node certificates are issued for. | [optional] [default to undefined]
**ExternalCAs** | [**Array&lt;SwarmSpecCAConfigExternalCAsInner&gt;**](SwarmSpecCAConfigExternalCAsInner.md) | Configuration for forwarding signing requests to an external certificate authority.  | [optional] [default to undefined]
**SigningCACert** | **string** | The desired signing CA certificate for all swarm node TLS leaf certificates, in PEM format.  | [optional] [default to undefined]
**SigningCAKey** | **string** | The desired signing CA key for all swarm node TLS leaf certificates, in PEM format.  | [optional] [default to undefined]
**ForceRotate** | **number** | An integer whose purpose is to force swarm to generate a new signing CA certificate and key, if none have been specified in &#x60;SigningCACert&#x60; and &#x60;SigningCAKey&#x60;  | [optional] [default to undefined]

## Example

```typescript
import { SwarmSpecCAConfig } from './api';

const instance: SwarmSpecCAConfig = {
    NodeCertExpiry,
    ExternalCAs,
    SigningCACert,
    SigningCAKey,
    ForceRotate,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
