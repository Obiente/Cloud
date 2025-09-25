# EndpointPortConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | [optional] [default to undefined]
**Protocol** | **string** |  | [optional] [default to undefined]
**TargetPort** | **number** | The port inside the container. | [optional] [default to undefined]
**PublishedPort** | **number** | The port on the swarm hosts. | [optional] [default to undefined]
**PublishMode** | **string** | The mode in which port is published.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  - \&quot;ingress\&quot; makes the target port accessible on every node,   regardless of whether there is a task for the service running on   that node or not. - \&quot;host\&quot; bypasses the routing mesh and publish the port directly on   the swarm node where that service is running.  | [optional] [default to PublishModeEnum_Ingress]

## Example

```typescript
import { EndpointPortConfig } from './api';

const instance: EndpointPortConfig = {
    Name,
    Protocol,
    TargetPort,
    PublishedPort,
    PublishMode,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
