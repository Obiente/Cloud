# Driver

Driver represents a driver (network, logging, secrets).

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name of the driver. | [default to undefined]
**Options** | **{ [key: string]: string; }** | Key/value map of driver-specific options. | [optional] [default to undefined]

## Example

```typescript
import { Driver } from './api';

const instance: Driver = {
    Name,
    Options,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
