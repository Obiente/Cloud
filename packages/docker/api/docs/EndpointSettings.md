# EndpointSettings

Configuration for a network endpoint.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**IPAMConfig** | [**EndpointIPAMConfig**](EndpointIPAMConfig.md) |  | [optional] [default to undefined]
**Links** | **Array&lt;string&gt;** |  | [optional] [default to undefined]
**MacAddress** | **string** | MAC address for the endpoint on this network. The network driver might ignore this parameter.  | [optional] [default to undefined]
**Aliases** | **Array&lt;string&gt;** |  | [optional] [default to undefined]
**DriverOpts** | **{ [key: string]: string; }** | DriverOpts is a mapping of driver options and values. These options are passed directly to the driver and are driver specific.  | [optional] [default to undefined]
**GwPriority** | **number** | This property determines which endpoint will provide the default gateway for a container. The endpoint with the highest priority will be used. If multiple endpoints have the same priority, endpoints are lexicographically sorted based on their network name, and the one that sorts first is picked.  | [optional] [default to undefined]
**NetworkID** | **string** | Unique ID of the network.  | [optional] [default to undefined]
**EndpointID** | **string** | Unique ID for the service endpoint in a Sandbox.  | [optional] [default to undefined]
**Gateway** | **string** | Gateway address for this network.  | [optional] [default to undefined]
**IPAddress** | **string** | IPv4 address.  | [optional] [default to undefined]
**IPPrefixLen** | **number** | Mask length of the IPv4 address.  | [optional] [default to undefined]
**IPv6Gateway** | **string** | IPv6 gateway address.  | [optional] [default to undefined]
**GlobalIPv6Address** | **string** | Global IPv6 address.  | [optional] [default to undefined]
**GlobalIPv6PrefixLen** | **number** | Mask length of the global IPv6 address.  | [optional] [default to undefined]
**DNSNames** | **Array&lt;string&gt;** | List of all DNS names an endpoint has on a specific network. This list is based on the container name, network aliases, container short ID, and hostname.  These DNS names are non-fully qualified but can contain several dots. You can get fully qualified DNS names by appending &#x60;.&lt;network-name&gt;&#x60;. For instance, if container name is &#x60;my.ctr&#x60; and the network is named &#x60;testnet&#x60;, &#x60;DNSNames&#x60; will contain &#x60;my.ctr&#x60; and the FQDN will be &#x60;my.ctr.testnet&#x60;.  | [optional] [default to undefined]

## Example

```typescript
import { EndpointSettings } from './api';

const instance: EndpointSettings = {
    IPAMConfig,
    Links,
    MacAddress,
    Aliases,
    DriverOpts,
    GwPriority,
    NetworkID,
    EndpointID,
    Gateway,
    IPAddress,
    IPPrefixLen,
    IPv6Gateway,
    GlobalIPv6Address,
    GlobalIPv6PrefixLen,
    DNSNames,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
