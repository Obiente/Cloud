# Volume


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the volume. | [default to undefined]
**Driver** | **string** | Name of the volume driver used by the volume. | [default to undefined]
**Mountpoint** | **string** | Mount path of the volume on the host. | [default to undefined]
**CreatedAt** | **string** | Date/Time the volume was created. | [optional] [default to undefined]
**Status** | **{ [key: string]: object; }** | Low-level details about the volume, provided by the volume driver. Details are returned as a map with key/value pairs: &#x60;{\&quot;key\&quot;:\&quot;value\&quot;,\&quot;key2\&quot;:\&quot;value2\&quot;}&#x60;.  The &#x60;Status&#x60; field is optional, and is omitted if the volume driver does not support this feature.  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [default to undefined]
**Scope** | **string** | The level at which the volume exists. Either &#x60;global&#x60; for cluster-wide, or &#x60;local&#x60; for machine level.  | [default to ScopeEnum_Local]
**ClusterVolume** | [**ClusterVolume**](ClusterVolume.md) |  | [optional] [default to undefined]
**Options** | **{ [key: string]: string; }** | The driver specific options used when creating the volume.  | [default to undefined]
**UsageData** | [**VolumeUsageData**](VolumeUsageData.md) |  | [optional] [default to undefined]

## Example

```typescript
import { Volume } from './api';

const instance: Volume = {
    Name,
    Driver,
    Mountpoint,
    CreatedAt,
    Status,
    Labels,
    Scope,
    ClusterVolume,
    Options,
    UsageData,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
