# TaskSpecContainerSpecDNSConfig

Specification for DNS related configurations in resolver configuration file (`resolv.conf`). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Nameservers** | **Array&lt;string&gt;** | The IP addresses of the name servers. | [optional] [default to undefined]
**Search** | **Array&lt;string&gt;** | A search list for host-name lookup. | [optional] [default to undefined]
**Options** | **Array&lt;string&gt;** | A list of internal resolver variables to be modified (e.g., &#x60;debug&#x60;, &#x60;ndots:3&#x60;, etc.).  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecDNSConfig } from './api';

const instance: TaskSpecContainerSpecDNSConfig = {
    Nameservers,
    Search,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
