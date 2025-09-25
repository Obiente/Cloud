# ContainerInspectResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **string** | The ID of this container as a 128-bit (64-character) hexadecimal string (32 bytes). | [optional] [default to undefined]
**Created** | **string** | Date and time at which the container was created, formatted in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds. | [optional] [default to undefined]
**Path** | **string** | The path to the command being run | [optional] [default to undefined]
**Args** | **Array&lt;string&gt;** | The arguments to the command being run | [optional] [default to undefined]
**State** | [**ContainerState**](ContainerState.md) |  | [optional] [default to undefined]
**Image** | **string** | The ID (digest) of the image that this container was created from. | [optional] [default to undefined]
**ResolvConfPath** | **string** | Location of the &#x60;/etc/resolv.conf&#x60; generated for the container on the host.  This file is managed through the docker daemon, and should not be accessed or modified by other tools. | [optional] [default to undefined]
**HostnamePath** | **string** | Location of the &#x60;/etc/hostname&#x60; generated for the container on the host.  This file is managed through the docker daemon, and should not be accessed or modified by other tools. | [optional] [default to undefined]
**HostsPath** | **string** | Location of the &#x60;/etc/hosts&#x60; generated for the container on the host.  This file is managed through the docker daemon, and should not be accessed or modified by other tools. | [optional] [default to undefined]
**LogPath** | **string** | Location of the file used to buffer the container\&#39;s logs. Depending on the logging-driver used for the container, this field may be omitted.  This file is managed through the docker daemon, and should not be accessed or modified by other tools. | [optional] [default to undefined]
**Name** | **string** | The name associated with this container.  For historic reasons, the name may be prefixed with a forward-slash (&#x60;/&#x60;). | [optional] [default to undefined]
**RestartCount** | **number** | Number of times the container was restarted since it was created, or since daemon was started. | [optional] [default to undefined]
**Driver** | **string** | The storage-driver used for the container\&#39;s filesystem (graph-driver or snapshotter). | [optional] [default to undefined]
**Platform** | **string** | The platform (operating system) for which the container was created.  This field was introduced for the experimental \&quot;LCOW\&quot; (Linux Containers On Windows) features, which has been removed. In most cases, this field is equal to the host\&#39;s operating system (&#x60;linux&#x60; or &#x60;windows&#x60;). | [optional] [default to undefined]
**ImageManifestDescriptor** | [**OCIDescriptor**](OCIDescriptor.md) |  | [optional] [default to undefined]
**MountLabel** | **string** | SELinux mount label set for the container. | [optional] [default to undefined]
**ProcessLabel** | **string** | SELinux process label set for the container. | [optional] [default to undefined]
**AppArmorProfile** | **string** | The AppArmor profile set for the container. | [optional] [default to undefined]
**ExecIDs** | **Array&lt;string&gt;** | IDs of exec instances that are running in the container. | [optional] [default to undefined]
**HostConfig** | [**HostConfig**](HostConfig.md) |  | [optional] [default to undefined]
**GraphDriver** | [**DriverData**](DriverData.md) |  | [optional] [default to undefined]
**SizeRw** | **number** | The size of files that have been created or changed by this container.  This field is omitted by default, and only set when size is requested in the API request. | [optional] [default to undefined]
**SizeRootFs** | **number** | The total size of all files in the read-only layers from the image that the container uses. These layers can be shared between containers.  This field is omitted by default, and only set when size is requested in the API request. | [optional] [default to undefined]
**Mounts** | [**Array&lt;MountPoint&gt;**](MountPoint.md) | List of mounts used by the container. | [optional] [default to undefined]
**Config** | [**ContainerConfig**](ContainerConfig.md) |  | [optional] [default to undefined]
**NetworkSettings** | [**NetworkSettings**](NetworkSettings.md) |  | [optional] [default to undefined]

## Example

```typescript
import { ContainerInspectResponse } from './api';

const instance: ContainerInspectResponse = {
    Id,
    Created,
    Path,
    Args,
    State,
    Image,
    ResolvConfPath,
    HostnamePath,
    HostsPath,
    LogPath,
    Name,
    RestartCount,
    Driver,
    Platform,
    ImageManifestDescriptor,
    MountLabel,
    ProcessLabel,
    AppArmorProfile,
    ExecIDs,
    HostConfig,
    GraphDriver,
    SizeRw,
    SizeRootFs,
    Mounts,
    Config,
    NetworkSettings,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
