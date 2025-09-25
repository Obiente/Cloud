# DriverData

Information about the storage driver used to store the container\'s and image\'s filesystem. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the storage driver. | [default to undefined]
**Data** | **{ [key: string]: string; }** | Low-level storage metadata, provided as key/value pairs.  This information is driver-specific, and depends on the storage-driver in use, and should be used for informational purposes only.  | [default to undefined]

## Example

```typescript
import { DriverData } from './api';

const instance: DriverData = {
    Name,
    Data,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
