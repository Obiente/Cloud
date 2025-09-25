# TLSInfo

Information about the issuer of leaf TLS certificates and the trusted root CA certificate. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**TrustRoot** | **string** | The root CA certificate(s) that are used to validate leaf TLS certificates.  | [optional] [default to undefined]
**CertIssuerSubject** | **string** | The base64-url-safe-encoded raw subject bytes of the issuer. | [optional] [default to undefined]
**CertIssuerPublicKey** | **string** | The base64-url-safe-encoded raw public key bytes of the issuer.  | [optional] [default to undefined]

## Example

```typescript
import { TLSInfo } from './api';

const instance: TLSInfo = {
    TrustRoot,
    CertIssuerSubject,
    CertIssuerPublicKey,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
