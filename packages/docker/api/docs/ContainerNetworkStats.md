# ContainerNetworkStats

Aggregates the network stats of one container 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**rx_bytes** | **number** | Bytes received. Windows and Linux.  | [optional] [default to undefined]
**rx_packets** | **number** | Packets received. Windows and Linux.  | [optional] [default to undefined]
**rx_errors** | **number** | Received errors. Not used on Windows.  This field is Linux-specific and always zero for Windows containers.  | [optional] [default to undefined]
**rx_dropped** | **number** | Incoming packets dropped. Windows and Linux.  | [optional] [default to undefined]
**tx_bytes** | **number** | Bytes sent. Windows and Linux.  | [optional] [default to undefined]
**tx_packets** | **number** | Packets sent. Windows and Linux.  | [optional] [default to undefined]
**tx_errors** | **number** | Sent errors. Not used on Windows.  This field is Linux-specific and always zero for Windows containers.  | [optional] [default to undefined]
**tx_dropped** | **number** | Outgoing packets dropped. Windows and Linux.  | [optional] [default to undefined]
**endpoint_id** | **string** | Endpoint ID. Not used on Linux.  This field is Windows-specific and omitted for Linux containers.  | [optional] [default to undefined]
**instance_id** | **string** | Instance ID. Not used on Linux.  This field is Windows-specific and omitted for Linux containers.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerNetworkStats } from './api';

const instance: ContainerNetworkStats = {
    rx_bytes,
    rx_packets,
    rx_errors,
    rx_dropped,
    tx_bytes,
    tx_packets,
    tx_errors,
    tx_dropped,
    endpoint_id,
    instance_id,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
