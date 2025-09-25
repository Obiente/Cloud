# RegistryServiceConfig

RegistryServiceConfig stores daemon registry services configuration. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**InsecureRegistryCIDRs** | **Array&lt;string&gt;** | List of IP ranges of insecure registries, using the CIDR syntax ([RFC 4632](https://tools.ietf.org/html/4632)). Insecure registries accept un-encrypted (HTTP) and/or untrusted (HTTPS with certificates from unknown CAs) communication.  By default, local registries (&#x60;::1/128&#x60; and &#x60;127.0.0.0/8&#x60;) are configured as insecure. All other registries are secure. Communicating with an insecure registry is not possible if the daemon assumes that registry is secure.  This configuration override this behavior, insecure communication with registries whose resolved IP address is within the subnet described by the CIDR syntax.  Registries can also be marked insecure by hostname. Those registries are listed under &#x60;IndexConfigs&#x60; and have their &#x60;Secure&#x60; field set to &#x60;false&#x60;.  &gt; **Warning**: Using this option can be useful when running a local &gt; registry, but introduces security vulnerabilities. This option &gt; should therefore ONLY be used for testing purposes. For increased &gt; security, users should add their CA to their system\&#39;s list of trusted &gt; CAs instead of enabling this option.  | [optional] [default to undefined]
**IndexConfigs** | [**{ [key: string]: IndexInfo; }**](IndexInfo.md) |  | [optional] [default to undefined]
**Mirrors** | **Array&lt;string&gt;** | List of registry URLs that act as a mirror for the official (&#x60;docker.io&#x60;) registry.  | [optional] [default to undefined]

## Example

```typescript
import { RegistryServiceConfig } from './api';

const instance: RegistryServiceConfig = {
    InsecureRegistryCIDRs,
    IndexConfigs,
    Mirrors,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
