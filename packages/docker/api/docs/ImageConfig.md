# ImageConfig

Configuration of the image. These fields are used as defaults when starting a container from the image. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**User** | **string** | The user that commands are run as inside the container. | [optional] [default to undefined]
**ExposedPorts** | **{ [key: string]: object; }** | An object mapping ports to an empty object in the form:  &#x60;{\&quot;&lt;port&gt;/&lt;tcp|udp|sctp&gt;\&quot;: {}}&#x60;  | [optional] [default to undefined]
**Env** | **Array&lt;string&gt;** | A list of environment variables to set inside the container in the form &#x60;[\&quot;VAR&#x3D;value\&quot;, ...]&#x60;. A variable without &#x60;&#x3D;&#x60; is removed from the environment, rather than to have an empty value.  | [optional] [default to undefined]
**Cmd** | **Array&lt;string&gt;** | Command to run specified as a string or an array of strings.  | [optional] [default to undefined]
**Healthcheck** | [**HealthConfig**](HealthConfig.md) |  | [optional] [default to undefined]
**ArgsEscaped** | **boolean** | Command is already escaped (Windows only) | [optional] [default to false]
**Volumes** | **{ [key: string]: object; }** | An object mapping mount point paths inside the container to empty objects.  | [optional] [default to undefined]
**WorkingDir** | **string** | The working directory for commands to run in. | [optional] [default to undefined]
**Entrypoint** | **Array&lt;string&gt;** | The entry point for the container as a string or an array of strings.  If the array consists of exactly one empty string (&#x60;[\&quot;\&quot;]&#x60;) then the entry point is reset to system default (i.e., the entry point used by docker when there is no &#x60;ENTRYPOINT&#x60; instruction in the &#x60;Dockerfile&#x60;).  | [optional] [default to undefined]
**OnBuild** | **Array&lt;string&gt;** | &#x60;ONBUILD&#x60; metadata that were defined in the image\&#39;s &#x60;Dockerfile&#x60;.  | [optional] [default to undefined]
**Labels** | **{ [key: string]: string; }** | User-defined key/value metadata. | [optional] [default to undefined]
**StopSignal** | **string** | Signal to stop a container as a string or unsigned integer.  | [optional] [default to undefined]
**Shell** | **Array&lt;string&gt;** | Shell for when &#x60;RUN&#x60;, &#x60;CMD&#x60;, and &#x60;ENTRYPOINT&#x60; uses a shell.  | [optional] [default to undefined]

## Example

```typescript
import { ImageConfig } from './api';

const instance: ImageConfig = {
    User,
    ExposedPorts,
    Env,
    Cmd,
    Healthcheck,
    ArgsEscaped,
    Volumes,
    WorkingDir,
    Entrypoint,
    OnBuild,
    Labels,
    StopSignal,
    Shell,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
