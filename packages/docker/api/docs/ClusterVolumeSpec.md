# ClusterVolumeSpec

Cluster-specific options used to create the volume. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Group** | **string** | Group defines the volume group of this volume. Volumes belonging to the same group can be referred to by group name when creating Services.  Referring to a volume by group instructs Swarm to treat volumes in that group interchangeably for the purpose of scheduling. Volumes with an empty string for a group technically all belong to the same, emptystring group.  | [optional] [default to undefined]
**AccessMode** | [**ClusterVolumeSpecAccessMode**](ClusterVolumeSpecAccessMode.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolumeSpec } from './api';

const instance: ClusterVolumeSpec = {
    Group,
    AccessMode,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
