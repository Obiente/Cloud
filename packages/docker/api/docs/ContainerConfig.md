# ContainerConfig

Configuration for a container that is portable between hosts. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Hostname** | **string** | The hostname to use for the container, as a valid RFC 1123 hostname.  | [optional] [default to undefined]
**Domainname** | **string** | The domain name to use for the container.  | [optional] [default to undefined]
**User** | **string** | Commands run as this user inside the container. If omitted, commands run as the user specified in the image the container was started from.  Can be either user-name or UID, and optional group-name or GID, separated by a colon (&#x60;&lt;user-name|UID&gt;[&lt;:group-name|GID&gt;]&#x60;). | [optional] [default to undefined]
**AttachStdin** | **boolean** | Whether to attach to &#x60;stdin&#x60;. | [optional] [default to false]
**AttachStdout** | **boolean** | Whether to attach to &#x60;stdout&#x60;. | [optional] [default to true]
**AttachStderr** | **boolean** | Whether to attach to &#x60;stderr&#x60;. | [optional] [default to true]
**ExposedPorts** | **{ [key: string]: object; }** | An object mapping ports to an empty object in the form:  &#x60;{\&quot;&lt;port&gt;/&lt;tcp|udp|sctp&gt;\&quot;: {}}&#x60;  | [optional] [default to undefined]
**Tty** | **boolean** | Attach standard streams to a TTY, including &#x60;stdin&#x60; if it is not closed.  | [optional] [default to false]
**OpenStdin** | **boolean** | Open &#x60;stdin&#x60; | [optional] [default to false]
**StdinOnce** | **boolean** | Close &#x60;stdin&#x60; after one attached client disconnects | [optional] [default to false]
**Env** | **Array&lt;string&gt;** | A list of environment variables to set inside the container in the form &#x60;[\&quot;VAR&#x3D;value\&quot;, ...]&#x60;. A variable without &#x60;&#x3D;&#x60; is removed from the environment, rather than to have an empty value.  | [optional] [default to undefined]
**Cmd** | **Array&lt;string&gt;** | Command to run specified as a string or an array of strings.  | [optional] [default to undefined]
**Healthcheck** | [**HealthConfig**](HealthConfig.md) |  | [optional] [default to undefined]
**ArgsEscaped** | **boolean** | Command is already escaped (Windows only) | [optional] [default to false]
**Image** | **string** | The name (or reference) of the image to use when creating the container, or which was used when the container was created.  | [optional] [default to undefined]
**Volumes** | **{ [key: string]: object; }** | An object mapping mount point paths inside the container to empty objects.  | [optional] [default to undefined]
**WorkingDir** | **string** | The working directory for commands to run in. | [optional] [default to undefined]
**Entrypoint** | **Array&lt;string&gt;** | The entry point for the container as a string or an array of strings.  If the array consists of exactly one empty string (&#x60;[\&quot;\&quot;]&#x60;) then the entry point is reset to system default (i.e., the entry point used by docker when there is no &#x60;ENTRYPOINT&#x60; instruction in the &#x60;Dockerfile&#x60;).  | [optional] [default to undefined]
**NetworkDisabled** | **boolean** | Disable networking for the container. | [optional] [default to undefined]
**MacAddress** | **string** | MAC address of the container.  Deprecated: this field is deprecated in API v1.44 and up. Use EndpointSettings.MacAddress instead.  | [optional] [default to undefined]
**OnBuild** | **Array&lt;string&gt;** | &#x60;ONBUILD&#x60; metadata that were defined in the image\&#39;s &#x60;Dockerfile&#x60;.  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**StopSignal** | **string** | Signal to stop a container as a string or unsigned integer.  | [optional] [default to undefined]
**StopTimeout** | **number** | Timeout to stop a container in seconds. | [optional] [default to undefined]
**Shell** | **Array&lt;string&gt;** | Shell for when &#x60;RUN&#x60;, &#x60;CMD&#x60;, and &#x60;ENTRYPOINT&#x60; uses a shell.  | [optional] [default to undefined]

## Example

```typescript
import { ContainerConfig } from './api';

const instance: ContainerConfig = {
    Hostname,
    Domainname,
    User,
    AttachStdin,
    AttachStdout,
    AttachStderr,
    ExposedPorts,
    Tty,
    OpenStdin,
    StdinOnce,
    Env,
    Cmd,
    Healthcheck,
    ArgsEscaped,
    Image,
    Volumes,
    WorkingDir,
    Entrypoint,
    NetworkDisabled,
    MacAddress,
    OnBuild,
    Labels,
    StopSignal,
    StopTimeout,
    Shell,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
