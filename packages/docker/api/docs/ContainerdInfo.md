# ContainerdInfo

Information for connecting to the containerd instance that is used by the daemon. This is included for debugging purposes only. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Address** | **string** | The address of the containerd socket. | [optional] [default to undefined]
**Namespaces** | [**ContainerdInfoNamespaces**](ContainerdInfoNamespaces.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerdInfo } from './api';

const instance: ContainerdInfo = {
    Address,
    Namespaces,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
