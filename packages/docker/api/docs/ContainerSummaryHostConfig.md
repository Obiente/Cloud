# ContainerSummaryHostConfig

Summary of host-specific runtime information of the container. This is a reduced set of information in the container\'s \"HostConfig\" as available in the container \"inspect\" response.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NetworkMode** | **string** | Networking mode (&#x60;host&#x60;, &#x60;none&#x60;, &#x60;container:&lt;id&gt;&#x60;) or name of the primary network the container is using.  This field is primarily for backward compatibility. The container can be connected to multiple networks for which information can be found in the &#x60;NetworkSettings.Networks&#x60; field, which enumerates settings per network. | [optional] [default to undefined]
**Annotations** | **{ [key: string]: string; }** | Arbitrary key-value metadata attached to the container. | [optional] [default to undefined]

## Example

```typescript
import { ContainerSummaryHostConfig } from './api';

const instance: ContainerSummaryHostConfig = {
    NetworkMode,
    Annotations,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
