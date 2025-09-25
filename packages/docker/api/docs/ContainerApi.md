# ContainerApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**containerArchive**](#containerarchive) | **GET** /containers/{id}/archive | Get an archive of a filesystem resource in a container|
|[**containerArchiveInfo**](#containerarchiveinfo) | **HEAD** /containers/{id}/archive | Get information about files in a container|
|[**containerAttach**](#containerattach) | **POST** /containers/{id}/attach | Attach to a container|
|[**containerAttachWebsocket**](#containerattachwebsocket) | **GET** /containers/{id}/attach/ws | Attach to a container via a websocket|
|[**containerChanges**](#containerchanges) | **GET** /containers/{id}/changes | Get changes on a container’s filesystem|
|[**containerCreate**](#containercreate) | **POST** /containers/create | Create a container|
|[**containerDelete**](#containerdelete) | **DELETE** /containers/{id} | Remove a container|
|[**containerExport**](#containerexport) | **GET** /containers/{id}/export | Export a container|
|[**containerInspect**](#containerinspect) | **GET** /containers/{id}/json | Inspect a container|
|[**containerKill**](#containerkill) | **POST** /containers/{id}/kill | Kill a container|
|[**containerList**](#containerlist) | **GET** /containers/json | List containers|
|[**containerLogs**](#containerlogs) | **GET** /containers/{id}/logs | Get container logs|
|[**containerPause**](#containerpause) | **POST** /containers/{id}/pause | Pause a container|
|[**containerPrune**](#containerprune) | **POST** /containers/prune | Delete stopped containers|
|[**containerRename**](#containerrename) | **POST** /containers/{id}/rename | Rename a container|
|[**containerResize**](#containerresize) | **POST** /containers/{id}/resize | Resize a container TTY|
|[**containerRestart**](#containerrestart) | **POST** /containers/{id}/restart | Restart a container|
|[**containerStart**](#containerstart) | **POST** /containers/{id}/start | Start a container|
|[**containerStats**](#containerstats) | **GET** /containers/{id}/stats | Get container stats based on resource usage|
|[**containerStop**](#containerstop) | **POST** /containers/{id}/stop | Stop a container|
|[**containerTop**](#containertop) | **GET** /containers/{id}/top | List processes running inside a container|
|[**containerUnpause**](#containerunpause) | **POST** /containers/{id}/unpause | Unpause a container|
|[**containerUpdate**](#containerupdate) | **POST** /containers/{id}/update | Update a container|
|[**containerWait**](#containerwait) | **POST** /containers/{id}/wait | Wait for a container|
|[**putContainerArchive**](#putcontainerarchive) | **PUT** /containers/{id}/archive | Extract an archive of files or folders to a directory in a container|

# **containerArchive**
> containerArchive()

Get a tar archive of a resource in the filesystem of container id.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let path: string; //Resource in the container’s filesystem to archive. (default to undefined)

const { status, data } = await apiInstance.containerArchive(
    id,
    path
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **path** | [**string**] | Resource in the container’s filesystem to archive. | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/x-tar, application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**400** | Bad parameter |  -  |
|**404** | Container or path does not exist |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerArchiveInfo**
> containerArchiveInfo()

A response header `X-Docker-Container-Path-Stat` is returned, containing a base64 - encoded JSON object with some filesystem header information about the path. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let path: string; //Resource in the container’s filesystem to archive. (default to undefined)

const { status, data } = await apiInstance.containerArchiveInfo(
    id,
    path
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **path** | [**string**] | Resource in the container’s filesystem to archive. | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  * X-Docker-Container-Path-Stat - A base64 - encoded JSON object with some filesystem header information about the path  <br>  |
|**400** | Bad parameter |  -  |
|**404** | Container or path does not exist |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerAttach**
> containerAttach()

Attach to a container to read its output or send it input. You can attach to the same container multiple times and you can reattach to containers that have been detached.  Either the `stream` or `logs` parameter must be `true` for this endpoint to do anything.  See the [documentation for the `docker attach` command](https://docs.docker.com/engine/reference/commandline/attach/) for more details.  ### Hijacking  This endpoint hijacks the HTTP connection to transport `stdin`, `stdout`, and `stderr` on the same socket.  This is the response from the daemon for an attach request:  ``` HTTP/1.1 200 OK Content-Type: application/vnd.docker.raw-stream  [STREAM] ```  After the headers and two new lines, the TCP connection can now be used for raw, bidirectional communication between the client and server.  To hint potential proxies about connection hijacking, the Docker client can also optionally send connection upgrade headers.  For example, the client sends this request to upgrade the connection:  ``` POST /containers/16253994b7c4/attach?stream=1&stdout=1 HTTP/1.1 Upgrade: tcp Connection: Upgrade ```  The Docker daemon will respond with a `101 UPGRADED` response, and will similarly follow with the raw stream:  ``` HTTP/1.1 101 UPGRADED Content-Type: application/vnd.docker.raw-stream Connection: Upgrade Upgrade: tcp  [STREAM] ```  ### Stream format  When the TTY setting is disabled in [`POST /containers/create`](#operation/ContainerCreate), the HTTP Content-Type header is set to application/vnd.docker.multiplexed-stream and the stream over the hijacked connected is multiplexed to separate out `stdout` and `stderr`. The stream consists of a series of frames, each containing a header and a payload.  The header contains the information which the stream writes (`stdout` or `stderr`). It also contains the size of the associated frame encoded in the last four bytes (`uint32`).  It is encoded on the first eight bytes like this:  ```go header := [8]byte{STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4} ```  `STREAM_TYPE` can be:  - 0: `stdin` (is written on `stdout`) - 1: `stdout` - 2: `stderr`  `SIZE1, SIZE2, SIZE3, SIZE4` are the four bytes of the `uint32` size encoded as big endian.  Following the header is the payload, which is the specified number of bytes of `STREAM_TYPE`.  The simplest way to implement this protocol is the following:  1. Read 8 bytes. 2. Choose `stdout` or `stderr` depending on the first byte. 3. Extract the frame size from the last four bytes. 4. Read the extracted size and output it on the correct output. 5. Goto 1.  ### Stream format when using a TTY  When the TTY setting is enabled in [`POST /containers/create`](#operation/ContainerCreate), the stream is not multiplexed. The data exchanged over the hijacked connection is simply the raw data from the process PTY and client\'s `stdin`. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let detachKeys: string; //Override the key sequence for detaching a container.Format is a single character `[a-Z]` or `ctrl-<value>` where `<value>` is one of: `a-z`, `@`, `^`, `[`, `,` or `_`.  (optional) (default to undefined)
let logs: boolean; //Replay previous logs from the container.  This is useful for attaching to a container that has started and you want to output everything since the container started.  If `stream` is also enabled, once all the previous output has been returned, it will seamlessly transition into streaming current output.  (optional) (default to false)
let stream: boolean; //Stream attached streams from the time the request was made onwards.  (optional) (default to false)
let stdin: boolean; //Attach to `stdin` (optional) (default to false)
let stdout: boolean; //Attach to `stdout` (optional) (default to false)
let stderr: boolean; //Attach to `stderr` (optional) (default to false)

const { status, data } = await apiInstance.containerAttach(
    id,
    detachKeys,
    logs,
    stream,
    stdin,
    stdout,
    stderr
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **detachKeys** | [**string**] | Override the key sequence for detaching a container.Format is a single character &#x60;[a-Z]&#x60; or &#x60;ctrl-&lt;value&gt;&#x60; where &#x60;&lt;value&gt;&#x60; is one of: &#x60;a-z&#x60;, &#x60;@&#x60;, &#x60;^&#x60;, &#x60;[&#x60;, &#x60;,&#x60; or &#x60;_&#x60;.  | (optional) defaults to undefined|
| **logs** | [**boolean**] | Replay previous logs from the container.  This is useful for attaching to a container that has started and you want to output everything since the container started.  If &#x60;stream&#x60; is also enabled, once all the previous output has been returned, it will seamlessly transition into streaming current output.  | (optional) defaults to false|
| **stream** | [**boolean**] | Stream attached streams from the time the request was made onwards.  | (optional) defaults to false|
| **stdin** | [**boolean**] | Attach to &#x60;stdin&#x60; | (optional) defaults to false|
| **stdout** | [**boolean**] | Attach to &#x60;stdout&#x60; | (optional) defaults to false|
| **stderr** | [**boolean**] | Attach to &#x60;stderr&#x60; | (optional) defaults to false|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/vnd.docker.raw-stream, application/vnd.docker.multiplexed-stream, application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**101** | no error, hints proxy about hijacking |  -  |
|**200** | no error, no upgrade header found |  -  |
|**400** | bad parameter |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerAttachWebsocket**
> containerAttachWebsocket()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let detachKeys: string; //Override the key sequence for detaching a container.Format is a single character `[a-Z]` or `ctrl-<value>` where `<value>` is one of: `a-z`, `@`, `^`, `[`, `,`, or `_`.  (optional) (default to undefined)
let logs: boolean; //Return logs (optional) (default to false)
let stream: boolean; //Return stream (optional) (default to false)
let stdin: boolean; //Attach to `stdin` (optional) (default to false)
let stdout: boolean; //Attach to `stdout` (optional) (default to false)
let stderr: boolean; //Attach to `stderr` (optional) (default to false)

const { status, data } = await apiInstance.containerAttachWebsocket(
    id,
    detachKeys,
    logs,
    stream,
    stdin,
    stdout,
    stderr
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **detachKeys** | [**string**] | Override the key sequence for detaching a container.Format is a single character &#x60;[a-Z]&#x60; or &#x60;ctrl-&lt;value&gt;&#x60; where &#x60;&lt;value&gt;&#x60; is one of: &#x60;a-z&#x60;, &#x60;@&#x60;, &#x60;^&#x60;, &#x60;[&#x60;, &#x60;,&#x60;, or &#x60;_&#x60;.  | (optional) defaults to undefined|
| **logs** | [**boolean**] | Return logs | (optional) defaults to false|
| **stream** | [**boolean**] | Return stream | (optional) defaults to false|
| **stdin** | [**boolean**] | Attach to &#x60;stdin&#x60; | (optional) defaults to false|
| **stdout** | [**boolean**] | Attach to &#x60;stdout&#x60; | (optional) defaults to false|
| **stderr** | [**boolean**] | Attach to &#x60;stderr&#x60; | (optional) defaults to false|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**101** | no error, hints proxy about hijacking |  -  |
|**200** | no error, no upgrade header found |  -  |
|**400** | bad parameter |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerChanges**
> Array<FilesystemChange> containerChanges()

Returns which files in a container\'s filesystem have been added, deleted, or modified. The `Kind` of modification can be one of:  - `0`: Modified (\"C\") - `1`: Added (\"A\") - `2`: Deleted (\"D\") 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)

const { status, data } = await apiInstance.containerChanges(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|


### Return type

**Array<FilesystemChange>**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | The list of changes |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerCreate**
> ContainerCreateResponse containerCreate(body)


### Example

```typescript
import {
    ContainerApi,
    Configuration,
    ContainerCreateRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let body: ContainerCreateRequest; //Container to create
let name: string; //Assign the specified name to the container. Must match `/?[a-zA-Z0-9][a-zA-Z0-9_.-]+`.  (optional) (default to undefined)
let platform: string; //Platform in the format `os[/arch[/variant]]` used for image lookup.  When specified, the daemon checks if the requested image is present in the local image cache with the given OS and Architecture, and otherwise returns a `404` status.  If the option is not set, the host\'s native OS and Architecture are used to look up the image in the image cache. However, if no platform is passed and the given image does exist in the local image cache, but its OS or architecture does not match, the container is created with the available image, and a warning is added to the `Warnings` field in the response, for example;      WARNING: The requested image\'s platform (linux/arm64/v8) does not              match the detected host platform (linux/amd64) and no              specific platform was requested  (optional) (default to undefined)

const { status, data } = await apiInstance.containerCreate(
    body,
    name,
    platform
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **ContainerCreateRequest**| Container to create | |
| **name** | [**string**] | Assign the specified name to the container. Must match &#x60;/?[a-zA-Z0-9][a-zA-Z0-9_.-]+&#x60;.  | (optional) defaults to undefined|
| **platform** | [**string**] | Platform in the format &#x60;os[/arch[/variant]]&#x60; used for image lookup.  When specified, the daemon checks if the requested image is present in the local image cache with the given OS and Architecture, and otherwise returns a &#x60;404&#x60; status.  If the option is not set, the host\&#39;s native OS and Architecture are used to look up the image in the image cache. However, if no platform is passed and the given image does exist in the local image cache, but its OS or architecture does not match, the container is created with the available image, and a warning is added to the &#x60;Warnings&#x60; field in the response, for example;      WARNING: The requested image\&#39;s platform (linux/arm64/v8) does not              match the detected host platform (linux/amd64) and no              specific platform was requested  | (optional) defaults to undefined|


### Return type

**ContainerCreateResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/octet-stream
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**201** | Container created successfully |  -  |
|**400** | bad parameter |  -  |
|**404** | no such image |  -  |
|**409** | conflict |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerDelete**
> containerDelete()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let v: boolean; //Remove anonymous volumes associated with the container. (optional) (default to false)
let force: boolean; //If the container is running, kill it before removing it. (optional) (default to false)
let link: boolean; //Remove the specified link associated with the container. (optional) (default to false)

const { status, data } = await apiInstance.containerDelete(
    id,
    v,
    force,
    link
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **v** | [**boolean**] | Remove anonymous volumes associated with the container. | (optional) defaults to false|
| **force** | [**boolean**] | If the container is running, kill it before removing it. | (optional) defaults to false|
| **link** | [**boolean**] | Remove the specified link associated with the container. | (optional) defaults to false|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**400** | bad parameter |  -  |
|**404** | no such container |  -  |
|**409** | conflict |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerExport**
> containerExport()

Export the contents of a container as a tarball.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)

const { status, data } = await apiInstance.containerExport(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream, application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerInspect**
> ContainerInspectResponse containerInspect()

Return low-level information about a container.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let size: boolean; //Return the size of container as fields `SizeRw` and `SizeRootFs` (optional) (default to false)

const { status, data } = await apiInstance.containerInspect(
    id,
    size
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **size** | [**boolean**] | Return the size of container as fields &#x60;SizeRw&#x60; and &#x60;SizeRootFs&#x60; | (optional) defaults to false|


### Return type

**ContainerInspectResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerKill**
> containerKill()

Send a POSIX signal to a container, defaulting to killing to the container. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let signal: string; //Signal to send to the container as an integer or string (e.g. `SIGINT`).  (optional) (default to 'SIGKILL')

const { status, data } = await apiInstance.containerKill(
    id,
    signal
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **signal** | [**string**] | Signal to send to the container as an integer or string (e.g. &#x60;SIGINT&#x60;).  | (optional) defaults to 'SIGKILL'|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | no such container |  -  |
|**409** | container is not running |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerList**
> Array<ContainerSummary> containerList()

Returns a list of containers. For details on the format, see the [inspect endpoint](#operation/ContainerInspect).  Note that it uses a different, smaller representation of a container than inspecting a single container. For example, the list of linked containers is not propagated . 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let all: boolean; //Return all containers. By default, only running containers are shown.  (optional) (default to false)
let limit: number; //Return this number of most recently created containers, including non-running ones.  (optional) (default to undefined)
let size: boolean; //Return the size of container as fields `SizeRw` and `SizeRootFs`.  (optional) (default to false)
let filters: string; //Filters to process on the container list, encoded as JSON (a `map[string][]string`). For example, `{\"status\": [\"paused\"]}` will only return paused containers.  Available filters:  - `ancestor`=(`<image-name>[:<tag>]`, `<image id>`, or `<image@digest>`) - `before`=(`<container id>` or `<container name>`) - `expose`=(`<port>[/<proto>]`|`<startport-endport>/[<proto>]`) - `exited=<int>` containers with exit code of `<int>` - `health`=(`starting`|`healthy`|`unhealthy`|`none`) - `id=<ID>` a container\'s ID - `isolation=`(`default`|`process`|`hyperv`) (Windows daemon only) - `is-task=`(`true`|`false`) - `label=key` or `label=\"key=value\"` of a container label - `name=<name>` a container\'s name - `network`=(`<network id>` or `<network name>`) - `publish`=(`<port>[/<proto>]`|`<startport-endport>/[<proto>]`) - `since`=(`<container id>` or `<container name>`) - `status=`(`created`|`restarting`|`running`|`removing`|`paused`|`exited`|`dead`) - `volume`=(`<volume name>` or `<mount point destination>`)  (optional) (default to undefined)

const { status, data } = await apiInstance.containerList(
    all,
    limit,
    size,
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **all** | [**boolean**] | Return all containers. By default, only running containers are shown.  | (optional) defaults to false|
| **limit** | [**number**] | Return this number of most recently created containers, including non-running ones.  | (optional) defaults to undefined|
| **size** | [**boolean**] | Return the size of container as fields &#x60;SizeRw&#x60; and &#x60;SizeRootFs&#x60;.  | (optional) defaults to false|
| **filters** | [**string**] | Filters to process on the container list, encoded as JSON (a &#x60;map[string][]string&#x60;). For example, &#x60;{\&quot;status\&quot;: [\&quot;paused\&quot;]}&#x60; will only return paused containers.  Available filters:  - &#x60;ancestor&#x60;&#x3D;(&#x60;&lt;image-name&gt;[:&lt;tag&gt;]&#x60;, &#x60;&lt;image id&gt;&#x60;, or &#x60;&lt;image@digest&gt;&#x60;) - &#x60;before&#x60;&#x3D;(&#x60;&lt;container id&gt;&#x60; or &#x60;&lt;container name&gt;&#x60;) - &#x60;expose&#x60;&#x3D;(&#x60;&lt;port&gt;[/&lt;proto&gt;]&#x60;|&#x60;&lt;startport-endport&gt;/[&lt;proto&gt;]&#x60;) - &#x60;exited&#x3D;&lt;int&gt;&#x60; containers with exit code of &#x60;&lt;int&gt;&#x60; - &#x60;health&#x60;&#x3D;(&#x60;starting&#x60;|&#x60;healthy&#x60;|&#x60;unhealthy&#x60;|&#x60;none&#x60;) - &#x60;id&#x3D;&lt;ID&gt;&#x60; a container\&#39;s ID - &#x60;isolation&#x3D;&#x60;(&#x60;default&#x60;|&#x60;process&#x60;|&#x60;hyperv&#x60;) (Windows daemon only) - &#x60;is-task&#x3D;&#x60;(&#x60;true&#x60;|&#x60;false&#x60;) - &#x60;label&#x3D;key&#x60; or &#x60;label&#x3D;\&quot;key&#x3D;value\&quot;&#x60; of a container label - &#x60;name&#x3D;&lt;name&gt;&#x60; a container\&#39;s name - &#x60;network&#x60;&#x3D;(&#x60;&lt;network id&gt;&#x60; or &#x60;&lt;network name&gt;&#x60;) - &#x60;publish&#x60;&#x3D;(&#x60;&lt;port&gt;[/&lt;proto&gt;]&#x60;|&#x60;&lt;startport-endport&gt;/[&lt;proto&gt;]&#x60;) - &#x60;since&#x60;&#x3D;(&#x60;&lt;container id&gt;&#x60; or &#x60;&lt;container name&gt;&#x60;) - &#x60;status&#x3D;&#x60;(&#x60;created&#x60;|&#x60;restarting&#x60;|&#x60;running&#x60;|&#x60;removing&#x60;|&#x60;paused&#x60;|&#x60;exited&#x60;|&#x60;dead&#x60;) - &#x60;volume&#x60;&#x3D;(&#x60;&lt;volume name&gt;&#x60; or &#x60;&lt;mount point destination&gt;&#x60;)  | (optional) defaults to undefined|


### Return type

**Array<ContainerSummary>**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**400** | bad parameter |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerLogs**
> File containerLogs()

Get `stdout` and `stderr` logs from a container.  Note: This endpoint works only for containers with the `json-file` or `journald` logging driver. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let follow: boolean; //Keep connection after returning logs. (optional) (default to false)
let stdout: boolean; //Return logs from `stdout` (optional) (default to false)
let stderr: boolean; //Return logs from `stderr` (optional) (default to false)
let since: number; //Only return logs since this time, as a UNIX timestamp (optional) (default to 0)
let until: number; //Only return logs before this time, as a UNIX timestamp (optional) (default to 0)
let timestamps: boolean; //Add timestamps to every log line (optional) (default to false)
let tail: string; //Only return this number of log lines from the end of the logs. Specify as an integer or `all` to output all log lines.  (optional) (default to 'all')

const { status, data } = await apiInstance.containerLogs(
    id,
    follow,
    stdout,
    stderr,
    since,
    until,
    timestamps,
    tail
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **follow** | [**boolean**] | Keep connection after returning logs. | (optional) defaults to false|
| **stdout** | [**boolean**] | Return logs from &#x60;stdout&#x60; | (optional) defaults to false|
| **stderr** | [**boolean**] | Return logs from &#x60;stderr&#x60; | (optional) defaults to false|
| **since** | [**number**] | Only return logs since this time, as a UNIX timestamp | (optional) defaults to 0|
| **until** | [**number**] | Only return logs before this time, as a UNIX timestamp | (optional) defaults to 0|
| **timestamps** | [**boolean**] | Add timestamps to every log line | (optional) defaults to false|
| **tail** | [**string**] | Only return this number of log lines from the end of the logs. Specify as an integer or &#x60;all&#x60; to output all log lines.  | (optional) defaults to 'all'|


### Return type

**File**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/vnd.docker.raw-stream, application/vnd.docker.multiplexed-stream, application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | logs returned as a stream in response body. For the stream format, [see the documentation for the attach endpoint](#operation/ContainerAttach). Note that unlike the attach endpoint, the logs endpoint does not upgrade the connection and does not set Content-Type.  |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerPause**
> containerPause()

Use the freezer cgroup to suspend all processes in a container.  Traditionally, when suspending a process the `SIGSTOP` signal is used, which is observable by the process being suspended. With the freezer cgroup the process is unaware, and unable to capture, that it is being suspended, and subsequently resumed. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)

const { status, data } = await apiInstance.containerPause(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerPrune**
> ContainerPruneResponse containerPrune()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let filters: string; //Filters to process on the prune list, encoded as JSON (a `map[string][]string`).  Available filters: - `until=<timestamp>` Prune containers created before this timestamp. The `<timestamp>` can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. `10m`, `1h30m`) computed relative to the daemon machine’s time. - `label` (`label=<key>`, `label=<key>=<value>`, `label!=<key>`, or `label!=<key>=<value>`) Prune containers with (or without, in case `label!=...` is used) the specified labels.  (optional) (default to undefined)

const { status, data } = await apiInstance.containerPrune(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | Filters to process on the prune list, encoded as JSON (a &#x60;map[string][]string&#x60;).  Available filters: - &#x60;until&#x3D;&lt;timestamp&gt;&#x60; Prune containers created before this timestamp. The &#x60;&lt;timestamp&gt;&#x60; can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. &#x60;10m&#x60;, &#x60;1h30m&#x60;) computed relative to the daemon machine’s time. - &#x60;label&#x60; (&#x60;label&#x3D;&lt;key&gt;&#x60;, &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;, &#x60;label!&#x3D;&lt;key&gt;&#x60;, or &#x60;label!&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;) Prune containers with (or without, in case &#x60;label!&#x3D;...&#x60; is used) the specified labels.  | (optional) defaults to undefined|


### Return type

**ContainerPruneResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerRename**
> containerRename()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let name: string; //New name for the container (default to undefined)

const { status, data } = await apiInstance.containerRename(
    id,
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **name** | [**string**] | New name for the container | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | no such container |  -  |
|**409** | name already in use |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerResize**
> containerResize()

Resize the TTY for a container.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let h: number; //Height of the TTY session in characters (default to undefined)
let w: number; //Width of the TTY session in characters (default to undefined)

const { status, data } = await apiInstance.containerResize(
    id,
    h,
    w
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **h** | [**number**] | Height of the TTY session in characters | defaults to undefined|
| **w** | [**number**] | Width of the TTY session in characters | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain, application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such container |  -  |
|**500** | cannot resize container |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerRestart**
> containerRestart()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let signal: string; //Signal to send to the container as an integer or string (e.g. `SIGINT`).  (optional) (default to undefined)
let t: number; //Number of seconds to wait before killing the container (optional) (default to undefined)

const { status, data } = await apiInstance.containerRestart(
    id,
    signal,
    t
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **signal** | [**string**] | Signal to send to the container as an integer or string (e.g. &#x60;SIGINT&#x60;).  | (optional) defaults to undefined|
| **t** | [**number**] | Number of seconds to wait before killing the container | (optional) defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerStart**
> containerStart()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let detachKeys: string; //Override the key sequence for detaching a container. Format is a single character `[a-Z]` or `ctrl-<value>` where `<value>` is one of: `a-z`, `@`, `^`, `[`, `,` or `_`.  (optional) (default to undefined)

const { status, data } = await apiInstance.containerStart(
    id,
    detachKeys
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **detachKeys** | [**string**] | Override the key sequence for detaching a container. Format is a single character &#x60;[a-Z]&#x60; or &#x60;ctrl-&lt;value&gt;&#x60; where &#x60;&lt;value&gt;&#x60; is one of: &#x60;a-z&#x60;, &#x60;@&#x60;, &#x60;^&#x60;, &#x60;[&#x60;, &#x60;,&#x60; or &#x60;_&#x60;.  | (optional) defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**304** | container already started |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerStats**
> ContainerStatsResponse containerStats()

This endpoint returns a live stream of a container’s resource usage statistics.  The `precpu_stats` is the CPU statistic of the *previous* read, and is used to calculate the CPU usage percentage. It is not an exact copy of the `cpu_stats` field.  If either `precpu_stats.online_cpus` or `cpu_stats.online_cpus` is nil then for compatibility with older daemons the length of the corresponding `cpu_usage.percpu_usage` array should be used.  On a cgroup v2 host, the following fields are not set * `blkio_stats`: all fields other than `io_service_bytes_recursive` * `cpu_stats`: `cpu_usage.percpu_usage` * `memory_stats`: `max_usage` and `failcnt` Also, `memory_stats.stats` fields are incompatible with cgroup v1.  To calculate the values shown by the `stats` command of the docker cli tool the following formulas can be used: * used_memory = `memory_stats.usage - memory_stats.stats.cache` * available_memory = `memory_stats.limit` * Memory usage % = `(used_memory / available_memory) * 100.0` * cpu_delta = `cpu_stats.cpu_usage.total_usage - precpu_stats.cpu_usage.total_usage` * system_cpu_delta = `cpu_stats.system_cpu_usage - precpu_stats.system_cpu_usage` * number_cpus = `length(cpu_stats.cpu_usage.percpu_usage)` or `cpu_stats.online_cpus` * CPU usage % = `(cpu_delta / system_cpu_delta) * number_cpus * 100.0` 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let stream: boolean; //Stream the output. If false, the stats will be output once and then it will disconnect.  (optional) (default to true)
let oneShot: boolean; //Only get a single stat instead of waiting for 2 cycles. Must be used with `stream=false`.  (optional) (default to false)

const { status, data } = await apiInstance.containerStats(
    id,
    stream,
    oneShot
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **stream** | [**boolean**] | Stream the output. If false, the stats will be output once and then it will disconnect.  | (optional) defaults to true|
| **oneShot** | [**boolean**] | Only get a single stat instead of waiting for 2 cycles. Must be used with &#x60;stream&#x3D;false&#x60;.  | (optional) defaults to false|


### Return type

**ContainerStatsResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerStop**
> containerStop()


### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let signal: string; //Signal to send to the container as an integer or string (e.g. `SIGINT`).  (optional) (default to undefined)
let t: number; //Number of seconds to wait before killing the container (optional) (default to undefined)

const { status, data } = await apiInstance.containerStop(
    id,
    signal,
    t
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **signal** | [**string**] | Signal to send to the container as an integer or string (e.g. &#x60;SIGINT&#x60;).  | (optional) defaults to undefined|
| **t** | [**number**] | Number of seconds to wait before killing the container | (optional) defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**304** | container already stopped |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerTop**
> ContainerTopResponse containerTop()

On Unix systems, this is done by running the `ps` command. This endpoint is not supported on Windows. 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let psArgs: string; //The arguments to pass to `ps`. For example, `aux` (optional) (default to '-ef')

const { status, data } = await apiInstance.containerTop(
    id,
    psArgs
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **psArgs** | [**string**] | The arguments to pass to &#x60;ps&#x60;. For example, &#x60;aux&#x60; | (optional) defaults to '-ef'|


### Return type

**ContainerTopResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerUnpause**
> containerUnpause()

Resume a container which has been paused.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)

const { status, data } = await apiInstance.containerUnpause(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerUpdate**
> ContainerUpdateResponse containerUpdate(update)

Change various configuration options of a container without having to recreate it. 

### Example

```typescript
import {
    ContainerApi,
    Configuration,
    ContainerUpdateRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let update: ContainerUpdateRequest; //

const { status, data } = await apiInstance.containerUpdate(
    id,
    update
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **update** | **ContainerUpdateRequest**|  | |
| **id** | [**string**] | ID or name of the container | defaults to undefined|


### Return type

**ContainerUpdateResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | The container has been updated. |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **containerWait**
> ContainerWaitResponse containerWait()

Block until a container stops, then returns the exit code.

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let condition: 'not-running' | 'next-exit' | 'removed'; //Wait until a container state reaches the given condition.  Defaults to `not-running` if omitted or empty.  (optional) (default to 'not-running')

const { status, data } = await apiInstance.containerWait(
    id,
    condition
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **condition** | [**&#39;not-running&#39; | &#39;next-exit&#39; | &#39;removed&#39;**]**Array<&#39;not-running&#39; &#124; &#39;next-exit&#39; &#124; &#39;removed&#39;>** | Wait until a container state reaches the given condition.  Defaults to &#x60;not-running&#x60; if omitted or empty.  | (optional) defaults to 'not-running'|


### Return type

**ContainerWaitResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | The container has exit. |  -  |
|**400** | bad parameter |  -  |
|**404** | no such container |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **putContainerArchive**
> putContainerArchive(inputStream)

Upload a tar archive to be extracted to a path in the filesystem of container id. `path` parameter is asserted to be a directory. If it exists as a file, 400 error will be returned with message \"not a directory\". 

### Example

```typescript
import {
    ContainerApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ContainerApi(configuration);

let id: string; //ID or name of the container (default to undefined)
let path: string; //Path to a directory in the container to extract the archive’s contents into.  (default to undefined)
let inputStream: File; //The input stream must be a tar archive compressed with one of the following algorithms: `identity` (no compression), `gzip`, `bzip2`, or `xz`. 
let noOverwriteDirNonDir: string; //If `1`, `true`, or `True` then it will be an error if unpacking the given content would cause an existing directory to be replaced with a non-directory and vice versa.  (optional) (default to undefined)
let copyUIDGID: string; //If `1`, `true`, then it will copy UID/GID maps to the dest file or dir  (optional) (default to undefined)

const { status, data } = await apiInstance.putContainerArchive(
    id,
    path,
    inputStream,
    noOverwriteDirNonDir,
    copyUIDGID
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **inputStream** | **File**| The input stream must be a tar archive compressed with one of the following algorithms: &#x60;identity&#x60; (no compression), &#x60;gzip&#x60;, &#x60;bzip2&#x60;, or &#x60;xz&#x60;.  | |
| **id** | [**string**] | ID or name of the container | defaults to undefined|
| **path** | [**string**] | Path to a directory in the container to extract the archive’s contents into.  | defaults to undefined|
| **noOverwriteDirNonDir** | [**string**] | If &#x60;1&#x60;, &#x60;true&#x60;, or &#x60;True&#x60; then it will be an error if unpacking the given content would cause an existing directory to be replaced with a non-directory and vice versa.  | (optional) defaults to undefined|
| **copyUIDGID** | [**string**] | If &#x60;1&#x60;, &#x60;true&#x60;, then it will copy UID/GID maps to the dest file or dir  | (optional) defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/x-tar, application/octet-stream
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | The content was extracted successfully |  -  |
|**400** | Bad parameter |  -  |
|**403** | Permission denied, the volume or container rootfs is marked as read-only. |  -  |
|**404** | No such container or path does not exist inside the container |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

