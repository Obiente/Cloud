# Port

An open port on a container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IP** | **string** | Host IP address that the container\&#39;s port is mapped to | [optional] [default to undefined]
**PrivatePort** | **number** | Port on the container | [default to undefined]
**PublicPort** | **number** | Port exposed on the host | [optional] [default to undefined]
**Type** | **string** |  | [default to undefined]

## Example

```typescript
import { Port } from './api';

const instance: Port = {
    IP,
    PrivatePort,
    PublicPort,
    Type,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
