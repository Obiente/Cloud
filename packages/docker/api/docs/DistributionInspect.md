# DistributionInspect

Describes the result obtained from contacting the registry to retrieve image metadata. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Descriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [default to undefined]
**Platforms** | [**Array&lt;OCIPlatform&gt;**](OCIPlatform.md) | An array containing all platforms supported by the image.  | [default to undefined]

## Example

```typescript
import { DistributionInspect } from './api';

const instance: DistributionInspect = {
    Descriptor,
    Platforms,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
