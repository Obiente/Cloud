# IndexInfo

IndexInfo contains information about a registry.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the registry, such as \&quot;docker.io\&quot;.  | [optional] [default to undefined]
**Mirrors** | **Array&lt;string&gt;** | List of mirrors, expressed as URIs.  | [optional] [default to undefined]
**Secure** | **boolean** | Indicates if the registry is part of the list of insecure registries.  If &#x60;false&#x60;, the registry is insecure. Insecure registries accept un-encrypted (HTTP) and/or untrusted (HTTPS with certificates from unknown CAs) communication.  &gt; **Warning**: Insecure registries can be useful when running a local &gt; registry. However, because its use creates security vulnerabilities &gt; it should ONLY be enabled for testing purposes. For increased &gt; security, users should add their CA to their system\&#39;s list of &gt; trusted CAs instead of enabling this option.  | [optional] [default to undefined]
**Official** | **boolean** | Indicates whether this is an official registry (i.e., Docker Hub / docker.io)  | [optional] [default to undefined]

## Example

```typescript
import { IndexInfo } from './api';

const instance: IndexInfo = {
    Name,
    Mirrors,
    Secure,
    Official,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
