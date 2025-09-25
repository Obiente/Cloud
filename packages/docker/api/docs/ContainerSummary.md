# ContainerSummary


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** | The ID of this container as a 128-bit (64-character) hexadecimal string (32 bytes). | [optional] [default to undefined]
**Names** | **Array&lt;string&gt;** | The names associated with this container. Most containers have a single name, but when using legacy \&quot;links\&quot;, the container can have multiple names.  For historic reasons, names are prefixed with a forward-slash (&#x60;/&#x60;). | [optional] [default to undefined]
**Image** | **string** | The name or ID of the image used to create the container.  This field shows the image reference as was specified when creating the container, which can be in its canonical form (e.g., &#x60;docker.io/library/ubuntu:latest&#x60; or &#x60;docker.io/library/ubuntu@sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782&#x60;), short form (e.g., &#x60;ubuntu:latest&#x60;)), or the ID(-prefix) of the image (e.g., &#x60;72297848456d&#x60;).  The content of this field can be updated at runtime if the image used to create the container is untagged, in which case the field is updated to contain the the image ID (digest) it was resolved to in its canonical, non-truncated form (e.g., &#x60;sha256:72297848456d5d37d1262630108ab308d3e9ec7ed1c3286a32fe09856619a782&#x60;). | [optional] [default to undefined]
**ImageID** | **string** | The ID (digest) of the image that this container was created from. | [optional] [default to undefined]
**ImageManifestDescriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [optional] [default to undefined]
**Command** | **string** | Command to run when starting the container | [optional] [default to undefined]
**Created** | **number** | Date and time at which the container was created as a Unix timestamp (number of seconds since EPOCH). | [optional] [default to undefined]
**Ports** | [**Array&lt;Port&gt;**](Port.md) | Port-mappings for the container. | [optional] [default to undefined]
**SizeRw** | **number** | The size of files that have been created or changed by this container.  This field is omitted by default, and only set when size is requested in the API request. | [optional] [default to undefined]
**SizeRootFs** | **number** | The total size of all files in the read-only layers from the image that the container uses. These layers can be shared between containers.  This field is omitted by default, and only set when size is requested in the API request. | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**State** | **string** | The state of this container.  | [optional] [default to undefined]
**Status** | **string** | Additional human-readable status of this container (e.g. &#x60;Exit 0&#x60;) | [optional] [default to undefined]
**HostConfig** | [**ContainerSummaryHostConfig**](ContainerSummaryHostConfig.md) |  | [optional] [default to undefined]
**NetworkSettings** | [**ContainerSummaryNetworkSettings**](ContainerSummaryNetworkSettings.md) |  | [optional] [default to undefined]
**Mounts** | [**Array&lt;MountPoint&gt;**](MountPoint.md) | List of mounts used by the container. | [optional] [default to undefined]

## Example

```typescript
import { ContainerSummary } from './api';

const instance: ContainerSummary = {
    Id,
    Names,
    Image,
    ImageID,
    ImageManifestDescriptor,
    Command,
    Created,
    Ports,
    SizeRw,
    SizeRootFs,
    Labels,
    State,
    Status,
    HostConfig,
    NetworkSettings,
    Mounts,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
