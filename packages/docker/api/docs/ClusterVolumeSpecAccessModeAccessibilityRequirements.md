# ClusterVolumeSpecAccessModeAccessibilityRequirements

Requirements for the accessible topology of the volume. These fields are optional. For an in-depth description of what these fields mean, see the CSI specification. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Requisite** | **Array&lt;{ [key: string]: string; }&gt;** | A list of required topologies, at least one of which the volume must be accessible from.  | [optional] [default to undefined]
**Preferred** | **Array&lt;{ [key: string]: string; }&gt;** | A list of topologies that the volume should attempt to be provisioned in.  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolumeSpecAccessModeAccessibilityRequirements } from './api';

const instance: ClusterVolumeSpecAccessModeAccessibilityRequirements = {
    Requisite,
    Preferred,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
