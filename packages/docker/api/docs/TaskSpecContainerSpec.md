# TaskSpecContainerSpec

Container spec for the service.  <p><br /></p>  > **Note**: ContainerSpec, NetworkAttachmentSpec, and PluginSpec are > mutually exclusive. PluginSpec is only used when the Runtime field > is set to `plugin`. NetworkAttachmentSpec is used when the Runtime > field is set to `attachment`. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Image** | **string** | The image name to use for the container | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value data. | [optional] [default to undefined]
**Command** | **Array&lt;string&gt;** | The command to be run in the image. | [optional] [default to undefined]
**Args** | **Array&lt;string&gt;** | Arguments to the command. | [optional] [default to undefined]
**Hostname** | **string** | The hostname to use for the container, as a valid [RFC 1123](https://tools.ietf.org/html/rfc1123) hostname.  | [optional] [default to undefined]
**Env** | **Array&lt;string&gt;** | A list of environment variables in the form &#x60;VAR&#x3D;value&#x60;.  | [optional] [default to undefined]
**Dir** | **string** | The working directory for commands to run in. | [optional] [default to undefined]
**User** | **string** | The user inside the container. | [optional] [default to undefined]
**Groups** | **Array&lt;string&gt;** | A list of additional groups that the container process will run as.  | [optional] [default to undefined]
**Privileges** | [**TaskSpecContainerSpecPrivileges**](TaskSpecContainerSpecPrivileges.md) |  | [optional] [default to undefined]
**TTY** | **boolean** | Whether a pseudo-TTY should be allocated. | [optional] [default to undefined]
**OpenStdin** | **boolean** | Open &#x60;stdin&#x60; | [optional] [default to undefined]
**ReadOnly** | **boolean** | Mount the container\&#39;s root filesystem as read only. | [optional] [default to undefined]
**Mounts** | [**Array&lt;Mount&gt;**](Mount.md) | Specification for mounts to be added to containers created as part of the service.  | [optional] [default to undefined]
**StopSignal** | **string** | Signal to stop the container. | [optional] [default to undefined]
**StopGracePeriod** | **number** | Amount of time to wait for the container to terminate before forcefully killing it.  | [optional] [default to undefined]
**HealthCheck** | [**HealthConfig**](HealthConfig.md) |  | [optional] [default to undefined]
**Hosts** | **Array&lt;string&gt;** | A list of hostname/IP mappings to add to the container\&#39;s &#x60;hosts&#x60; file. The format of extra hosts is specified in the [hosts(5)](http://man7.org/linux/man-pages/man5/hosts.5.html) man page:      IP_address canonical_hostname [aliases...]  | [optional] [default to undefined]
**DNSConfig** | [**TaskSpecContainerSpecDNSConfig**](TaskSpecContainerSpecDNSConfig.md) |  | [optional] [default to undefined]
**Secrets** | [**Array&lt;TaskSpecContainerSpecSecretsInner&gt;**](TaskSpecContainerSpecSecretsInner.md) | Secrets contains references to zero or more secrets that will be exposed to the service.  | [optional] [default to undefined]
**OomScoreAdj** | **number** | An integer value containing the score given to the container in order to tune OOM killer preferences.  | [optional] [default to undefined]
**Configs** | [**Array&lt;TaskSpecContainerSpecConfigsInner&gt;**](TaskSpecContainerSpecConfigsInner.md) | Configs contains references to zero or more configs that will be exposed to the service.  | [optional] [default to undefined]
**Isolation** | **string** | Isolation technology of the containers running the service. (Windows only)  | [optional] [default to undefined]
**Init** | **boolean** | Run an init inside the container that forwards signals and reaps processes. This field is omitted if empty, and the default (as configured on the daemon) is used.  | [optional] [default to undefined]
**Sysctls** | **{ [key: string]: string; }** | Set kernel namedspaced parameters (sysctls) in the container. The Sysctls option on services accepts the same sysctls as the are supported on containers. Note that while the same sysctls are supported, no guarantees or checks are made about their suitability for a clustered environment, and it\&#39;s up to the user to determine whether a given sysctl will work properly in a Service.  | [optional] [default to undefined]
**CapabilityAdd** | **Array&lt;string&gt;** | A list of kernel capabilities to add to the default set for the container.  | [optional] [default to undefined]
**CapabilityDrop** | **Array&lt;string&gt;** | A list of kernel capabilities to drop from the default set for the container.  | [optional] [default to undefined]
**Ulimits** | [**Array&lt;ResourcesUlimitsInner&gt;**](ResourcesUlimitsInner.md) | A list of resource limits to set in the container. For example: &#x60;{\&quot;Name\&quot;: \&quot;nofile\&quot;, \&quot;Soft\&quot;: 1024, \&quot;Hard\&quot;: 2048}&#x60;\&quot;  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpec } from './api';

const instance: TaskSpecContainerSpec = {
    Image,
    Labels,
    Command,
    Args,
    Hostname,
    Env,
    Dir,
    User,
    Groups,
    Privileges,
    TTY,
    OpenStdin,
    ReadOnly,
    Mounts,
    StopSignal,
    StopGracePeriod,
    HealthCheck,
    Hosts,
    DNSConfig,
    Secrets,
    OomScoreAdj,
    Configs,
    Isolation,
    Init,
    Sysctls,
    CapabilityAdd,
    CapabilityDrop,
    Ulimits,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
