# ContainerSummaryNetworkSettings

Summary of the container\'s network settings

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Networks** | [**{ [key: string]: EndpointSettings; }**](EndpointSettings.md) | Summary of network-settings for each network the container is attached to. | [optional] [default to undefined]

## Example

```typescript
import { ContainerSummaryNetworkSettings } from './api';

const instance: ContainerSummaryNetworkSettings = {
    Networks,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
