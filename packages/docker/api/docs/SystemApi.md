# SystemApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**systemAuth**](#systemauth) | **POST** /auth | Check auth configuration|
|[**systemDataUsage**](#systemdatausage) | **GET** /system/df | Get data usage information|
|[**systemEvents**](#systemevents) | **GET** /events | Monitor events|
|[**systemInfo**](#systeminfo) | **GET** /info | Get system information|
|[**systemPing**](#systemping) | **GET** /_ping | Ping|
|[**systemPingHead**](#systempinghead) | **HEAD** /_ping | Ping|
|[**systemVersion**](#systemversion) | **GET** /version | Get version|

# **systemAuth**
> SystemAuthResponse systemAuth()

Validate credentials for a registry and, if available, get an identity token for accessing the registry without password. 

### Example

```typescript
import {
    SystemApi,
    Configuration,
    AuthConfig
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

let authConfig: AuthConfig; //Authentication to check (optional)

const { status, data } = await apiInstance.systemAuth(
    authConfig
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **authConfig** | **AuthConfig**| Authentication to check | |


### Return type

**SystemAuthResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | An identity token was generated successfully. |  -  |
|**204** | No error |  -  |
|**401** | Auth error |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **systemDataUsage**
> SystemDataUsageResponse systemDataUsage()


### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

let type: Array<'container' | 'image' | 'volume' | 'build-cache'>; //Object types, for which to compute and return data.  (optional) (default to undefined)

const { status, data } = await apiInstance.systemDataUsage(
    type
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **type** | **Array<&#39;container&#39; &#124; &#39;image&#39; &#124; &#39;volume&#39; &#124; &#39;build-cache&#39;>** | Object types, for which to compute and return data.  | (optional) defaults to undefined|


### Return type

**SystemDataUsageResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **systemEvents**
> EventMessage systemEvents()

Stream real-time events from the server.  Various objects within Docker report events when something happens to them.  Containers report these events: `attach`, `commit`, `copy`, `create`, `destroy`, `detach`, `die`, `exec_create`, `exec_detach`, `exec_start`, `exec_die`, `export`, `health_status`, `kill`, `oom`, `pause`, `rename`, `resize`, `restart`, `start`, `stop`, `top`, `unpause`, `update`, and `prune`  Images report these events: `create`, `delete`, `import`, `load`, `pull`, `push`, `save`, `tag`, `untag`, and `prune`  Volumes report these events: `create`, `mount`, `unmount`, `destroy`, and `prune`  Networks report these events: `create`, `connect`, `disconnect`, `destroy`, `update`, `remove`, and `prune`  The Docker daemon reports these events: `reload`  Services report these events: `create`, `update`, and `remove`  Nodes report these events: `create`, `update`, and `remove`  Secrets report these events: `create`, `update`, and `remove`  Configs report these events: `create`, `update`, and `remove`  The Builder reports `prune` events 

### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

let since: string; //Show events created since this timestamp then stream new events. (optional) (default to undefined)
let until: string; //Show events created until this timestamp then stop streaming. (optional) (default to undefined)
let filters: string; //A JSON encoded value of filters (a `map[string][]string`) to process on the event list. Available filters:  - `config=<string>` config name or ID - `container=<string>` container name or ID - `daemon=<string>` daemon name or ID - `event=<string>` event type - `image=<string>` image name or ID - `label=<string>` image or container label - `network=<string>` network name or ID - `node=<string>` node ID - `plugin`=<string> plugin name or ID - `scope`=<string> local or swarm - `secret=<string>` secret name or ID - `service=<string>` service name or ID - `type=<string>` object to filter by, one of `container`, `image`, `volume`, `network`, `daemon`, `plugin`, `node`, `service`, `secret` or `config` - `volume=<string>` volume name  (optional) (default to undefined)

const { status, data } = await apiInstance.systemEvents(
    since,
    until,
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **since** | [**string**] | Show events created since this timestamp then stream new events. | (optional) defaults to undefined|
| **until** | [**string**] | Show events created until this timestamp then stop streaming. | (optional) defaults to undefined|
| **filters** | [**string**] | A JSON encoded value of filters (a &#x60;map[string][]string&#x60;) to process on the event list. Available filters:  - &#x60;config&#x3D;&lt;string&gt;&#x60; config name or ID - &#x60;container&#x3D;&lt;string&gt;&#x60; container name or ID - &#x60;daemon&#x3D;&lt;string&gt;&#x60; daemon name or ID - &#x60;event&#x3D;&lt;string&gt;&#x60; event type - &#x60;image&#x3D;&lt;string&gt;&#x60; image name or ID - &#x60;label&#x3D;&lt;string&gt;&#x60; image or container label - &#x60;network&#x3D;&lt;string&gt;&#x60; network name or ID - &#x60;node&#x3D;&lt;string&gt;&#x60; node ID - &#x60;plugin&#x60;&#x3D;&lt;string&gt; plugin name or ID - &#x60;scope&#x60;&#x3D;&lt;string&gt; local or swarm - &#x60;secret&#x3D;&lt;string&gt;&#x60; secret name or ID - &#x60;service&#x3D;&lt;string&gt;&#x60; service name or ID - &#x60;type&#x3D;&lt;string&gt;&#x60; object to filter by, one of &#x60;container&#x60;, &#x60;image&#x60;, &#x60;volume&#x60;, &#x60;network&#x60;, &#x60;daemon&#x60;, &#x60;plugin&#x60;, &#x60;node&#x60;, &#x60;service&#x60;, &#x60;secret&#x60; or &#x60;config&#x60; - &#x60;volume&#x3D;&lt;string&gt;&#x60; volume name  | (optional) defaults to undefined|


### Return type

**EventMessage**

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

# **systemInfo**
> SystemInfo systemInfo()


### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

const { status, data } = await apiInstance.systemInfo();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**SystemInfo**

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

# **systemPing**
> string systemPing()

This is a dummy endpoint you can use to test if the server is accessible.

### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

const { status, data } = await apiInstance.systemPing();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  * Swarm - Contains information about Swarm status of the daemon, and if the daemon is acting as a manager or worker node.  <br>  * Api-Version - Max API Version the server supports <br>  * Docker-Experimental - If the server is running with experimental mode enabled <br>  * Cache-Control -  <br>  * Pragma -  <br>  * Builder-Version - Default version of docker image builder  The default on Linux is version \&quot;2\&quot; (BuildKit), but the daemon can be configured to recommend version \&quot;1\&quot; (classic Builder). Windows does not yet support BuildKit for native Windows images, and uses \&quot;1\&quot; (classic builder) as a default.  This value is a recommendation as advertised by the daemon, and it is up to the client to choose which builder to use.  <br>  |
|**500** | server error |  * Cache-Control -  <br>  * Pragma -  <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **systemPingHead**
> string systemPingHead()

This is a dummy endpoint you can use to test if the server is accessible.

### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

const { status, data } = await apiInstance.systemPingHead();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  * Swarm - Contains information about Swarm status of the daemon, and if the daemon is acting as a manager or worker node.  <br>  * Api-Version - Max API Version the server supports <br>  * Docker-Experimental - If the server is running with experimental mode enabled <br>  * Cache-Control -  <br>  * Pragma -  <br>  * Builder-Version - Default version of docker image builder <br>  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **systemVersion**
> SystemVersion systemVersion()

Returns the version of Docker that is running and various information about the system that Docker is running on.

### Example

```typescript
import {
    SystemApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SystemApi(configuration);

const { status, data } = await apiInstance.systemVersion();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**SystemVersion**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

