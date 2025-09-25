# ExecConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AttachStdin** | **boolean** | Attach to &#x60;stdin&#x60; of the exec command. | [optional] [default to undefined]
**AttachStdout** | **boolean** | Attach to &#x60;stdout&#x60; of the exec command. | [optional] [default to undefined]
**AttachStderr** | **boolean** | Attach to &#x60;stderr&#x60; of the exec command. | [optional] [default to undefined]
**ConsoleSize** | **Array&lt;number&gt;** | Initial console size, as an &#x60;[height, width]&#x60; array. | [optional] [default to undefined]
**DetachKeys** | **string** | Override the key sequence for detaching a container. Format is a single character &#x60;[a-Z]&#x60; or &#x60;ctrl-&lt;value&gt;&#x60; where &#x60;&lt;value&gt;&#x60; is one of: &#x60;a-z&#x60;, &#x60;@&#x60;, &#x60;^&#x60;, &#x60;[&#x60;, &#x60;,&#x60; or &#x60;_&#x60;.  | [optional] [default to undefined]
**Tty** | **boolean** | Allocate a pseudo-TTY. | [optional] [default to undefined]
**Env** | **Array&lt;string&gt;** | A list of environment variables in the form &#x60;[\&quot;VAR&#x3D;value\&quot;, ...]&#x60;.  | [optional] [default to undefined]
**Cmd** | **Array&lt;string&gt;** | Command to run, as a string or array of strings. | [optional] [default to undefined]
**Privileged** | **boolean** | Runs the exec process with extended privileges. | [optional] [default to false]
**User** | **string** | The user, and optionally, group to run the exec process inside the container. Format is one of: &#x60;user&#x60;, &#x60;user:group&#x60;, &#x60;uid&#x60;, or &#x60;uid:gid&#x60;.  | [optional] [default to undefined]
**WorkingDir** | **string** | The working directory for the exec process inside the container.  | [optional] [default to undefined]

## Example

```typescript
import { ExecConfig } from './api';

const instance: ExecConfig = {
    AttachStdin,
    AttachStdout,
    AttachStderr,
    ConsoleSize,
    DetachKeys,
    Tty,
    Env,
    Cmd,
    Privileged,
    User,
    WorkingDir,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
