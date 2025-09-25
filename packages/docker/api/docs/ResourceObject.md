# ResourceObject

An object describing the resources which can be advertised by a node and requested by a task. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NanoCPUs** | **number** |  | [optional] [default to undefined]
**MemoryBytes** | **number** |  | [optional] [default to undefined]
**GenericResources** | [**Array&lt;GenericResourcesInner&gt;**](GenericResourcesInner.md) | User-defined resources can be either Integer resources (e.g, &#x60;SSD&#x3D;3&#x60;) or String resources (e.g, &#x60;GPU&#x3D;UUID1&#x60;).  | [optional] [default to undefined]

## Example

```typescript
import { ResourceObject } from './api';

const instance: ResourceObject = {
    NanoCPUs,
    MemoryBytes,
    GenericResources,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
