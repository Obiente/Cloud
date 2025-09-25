# ContainerdInfoNamespaces

The namespaces that the daemon uses for running containers and plugins in containerd. These namespaces can be configured in the daemon configuration, and are considered to be used exclusively by the daemon, Tampering with the containerd instance may cause unexpected behavior.  As these namespaces are considered to be exclusively accessed by the daemon, it is not recommended to change these values, or to change them to a value that is used by other systems, such as cri-containerd. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Containers** | **string** | The default containerd namespace used for containers managed by the daemon.  The default namespace for containers is \&quot;moby\&quot;, but will be suffixed with the &#x60;&lt;uid&gt;.&lt;gid&gt;&#x60; of the remapped &#x60;root&#x60; if user-namespaces are enabled and the containerd image-store is used.  | [optional] [default to 'moby']
**Plugins** | **string** | The default containerd namespace used for plugins managed by the daemon.  The default namespace for plugins is \&quot;plugins.moby\&quot;, but will be suffixed with the &#x60;&lt;uid&gt;.&lt;gid&gt;&#x60; of the remapped &#x60;root&#x60; if user-namespaces are enabled and the containerd image-store is used.  | [optional] [default to 'plugins.moby']

## Example

```typescript
import { ContainerdInfoNamespaces } from './api';

const instance: ContainerdInfoNamespaces = {
    Containers,
    Plugins,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
