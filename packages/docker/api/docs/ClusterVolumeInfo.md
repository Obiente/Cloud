# ClusterVolumeInfo

Information about the global status of the volume. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CapacityBytes** | **number** | The capacity of the volume in bytes. A value of 0 indicates that the capacity is unknown.  | [optional] [default to undefined]
**VolumeContext** | **{ [key: string]: string; }** | A map of strings to strings returned from the storage plugin when the volume is created.  | [optional] [default to undefined]
**VolumeID** | **string** | The ID of the volume as returned by the CSI storage plugin. This is distinct from the volume\&#39;s ID as provided by Docker. This ID is never used by the user when communicating with Docker to refer to this volume. If the ID is blank, then the Volume has not been successfully created in the plugin yet.  | [optional] [default to undefined]
**AccessibleTopology** | **Array&lt;{ [key: string]: string; }&gt;** | The topology this volume is actually accessible from.  | [optional] [default to undefined]

## Example

```typescript
import { ClusterVolumeInfo } from './api';

const instance: ClusterVolumeInfo = {
    CapacityBytes,
    VolumeContext,
    VolumeID,
    AccessibleTopology,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
