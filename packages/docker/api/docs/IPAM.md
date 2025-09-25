# IPAM


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | **string** | Name of the IPAM driver to use. | [optional] [default to 'default']
**Config** | [**Array&lt;IPAMConfig&gt;**](IPAMConfig.md) | List of IPAM configuration options, specified as a map:  &#x60;&#x60;&#x60; {\&quot;Subnet\&quot;: &lt;CIDR&gt;, \&quot;IPRange\&quot;: &lt;CIDR&gt;, \&quot;Gateway\&quot;: &lt;IP address&gt;, \&quot;AuxAddress\&quot;: &lt;device_name:IP address&gt;} &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | Driver-specific options, specified as a map. | [optional] [default to undefined]

## Example

```typescript
import { IPAM } from './api';

const instance: IPAM = {
    Driver,
    Config,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
