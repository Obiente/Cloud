# NetworkingConfig

NetworkingConfig represents the container\'s networking configuration for each of its interfaces. It is used for the networking configs specified in the `docker create` and `docker network connect` commands. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EndpointsConfig** | [**{ [key: string]: EndpointSettings; }**](EndpointSettings.md) | A mapping of network name to endpoint configuration for that network. The endpoint configuration can be left empty to connect to that network with no particular endpoint configuration.  | [optional] [default to undefined]

## Example

```typescript
import { NetworkingConfig } from './api';

const instance: NetworkingConfig = {
    EndpointsConfig,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
