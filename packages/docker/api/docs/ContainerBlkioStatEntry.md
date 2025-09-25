# ContainerBlkioStatEntry

Blkio stats entry.  This type is Linux-specific and omitted for Windows containers. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**major** | **number** |  | [optional] [default to undefined]
**minor** | **number** |  | [optional] [default to undefined]
**op** | **string** |  | [optional] [default to undefined]
**value** | **number** |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerBlkioStatEntry } from './api';

const instance: ContainerBlkioStatEntry = {
    major,
    minor,
    op,
    value,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
