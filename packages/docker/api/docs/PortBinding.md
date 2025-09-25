# PortBinding

PortBinding represents a binding between a host IP address and a host port. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**HostIp** | **string** | Host IP address that the container\&#39;s port is mapped to. | [optional] [default to undefined]
**HostPort** | **string** | Host port number that the container\&#39;s port is mapped to. | [optional] [default to undefined]

## Example

```typescript
import { PortBinding } from './api';

const instance: PortBinding = {
    HostIp,
    HostPort,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
