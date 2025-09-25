# NetworkSettings

NetworkSettings exposes the network settings in the API

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Bridge** | **string** | Name of the default bridge interface when dockerd\&#39;s --bridge flag is set.  | [optional] [default to undefined]
**SandboxID** | **string** | SandboxID uniquely represents a container\&#39;s network stack. | [optional] [default to undefined]
**HairpinMode** | **boolean** | Indicates if hairpin NAT should be enabled on the virtual interface.  Deprecated: This field is never set and will be removed in a future release.  | [optional] [default to undefined]
**LinkLocalIPv6Address** | **string** | IPv6 unicast address using the link-local prefix.  Deprecated: This field is never set and will be removed in a future release.  | [optional] [default to undefined]
**LinkLocalIPv6PrefixLen** | **number** | Prefix length of the IPv6 unicast address.  Deprecated: This field is never set and will be removed in a future release.  | [optional] [default to undefined]
**Ports** | **{ [key: string]: Array&lt;PortBinding&gt; | null; }** | PortMap describes the mapping of container ports to host ports, using the container\&#39;s port-number and protocol as key in the format &#x60;&lt;port&gt;/&lt;protocol&gt;&#x60;, for example, &#x60;80/udp&#x60;.  If a container\&#39;s port is mapped for multiple protocols, separate entries are added to the mapping table.  | [optional] [default to undefined]
**SandboxKey** | **string** | SandboxKey is the full path of the netns handle | [optional] [default to undefined]
**SecondaryIPAddresses** | [**Array&lt;Address&gt;**](Address.md) | Deprecated: This field is never set and will be removed in a future release. | [optional] [default to undefined]
**SecondaryIPv6Addresses** | [**Array&lt;Address&gt;**](Address.md) | Deprecated: This field is never set and will be removed in a future release. | [optional] [default to undefined]
**EndpointID** | **string** | EndpointID uniquely represents a service endpoint in a Sandbox.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**Gateway** | **string** | Gateway address for the default \&quot;bridge\&quot; network.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**GlobalIPv6Address** | **string** | Global IPv6 address for the default \&quot;bridge\&quot; network.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**GlobalIPv6PrefixLen** | **number** | Mask length of the global IPv6 address.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**IPAddress** | **string** | IPv4 address for the default \&quot;bridge\&quot; network.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**IPPrefixLen** | **number** | Mask length of the IPv4 address.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**IPv6Gateway** | **string** | IPv6 gateway address for this network.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**MacAddress** | **string** | MAC address for the container on the default \&quot;bridge\&quot; network.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Deprecated**: This field is only propagated when attached to the &gt; default \&quot;bridge\&quot; network. Use the information from the \&quot;bridge\&quot; &gt; network inside the &#x60;Networks&#x60; map instead, which contains the same &gt; information. This field was deprecated in Docker 1.9 and is scheduled &gt; to be removed in Docker 17.12.0  | [optional] [default to undefined]
**Networks** | [**{ [key: string]: EndpointSettings; }**](EndpointSettings.md) | Information about all networks that the container is connected to.  | [optional] [default to undefined]

## Example

```typescript
import { NetworkSettings } from './api';

const instance: NetworkSettings = {
    Bridge,
    SandboxID,
    HairpinMode,
    LinkLocalIPv6Address,
    LinkLocalIPv6PrefixLen,
    Ports,
    SandboxKey,
    SecondaryIPAddresses,
    SecondaryIPv6Addresses,
    EndpointID,
    Gateway,
    GlobalIPv6Address,
    GlobalIPv6PrefixLen,
    IPAddress,
    IPPrefixLen,
    IPv6Gateway,
    MacAddress,
    Networks,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
