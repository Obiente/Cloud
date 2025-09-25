# TaskApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**taskInspect**](#taskinspect) | **GET** /tasks/{id} | Inspect a task|
|[**taskList**](#tasklist) | **GET** /tasks | List tasks|
|[**taskLogs**](#tasklogs) | **GET** /tasks/{id}/logs | Get task logs|

# **taskInspect**
> Task taskInspect()


### Example

```typescript
import {
    TaskApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new TaskApi(configuration);

let id: string; //ID of the task (default to undefined)

const { status, data } = await apiInstance.taskInspect(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID of the task | defaults to undefined|


### Return type

**Task**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such task |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **taskList**
> Array<Task> taskList()


### Example

```typescript
import {
    TaskApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new TaskApi(configuration);

let filters: string; //A JSON encoded value of the filters (a `map[string][]string`) to process on the tasks list.  Available filters:  - `desired-state=(running | shutdown | accepted)` - `id=<task id>` - `label=key` or `label=\"key=value\"` - `name=<task name>` - `node=<node id or name>` - `service=<service name>`  (optional) (default to undefined)

const { status, data } = await apiInstance.taskList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the tasks list.  Available filters:  - &#x60;desired-state&#x3D;(running | shutdown | accepted)&#x60; - &#x60;id&#x3D;&lt;task id&gt;&#x60; - &#x60;label&#x3D;key&#x60; or &#x60;label&#x3D;\&quot;key&#x3D;value\&quot;&#x60; - &#x60;name&#x3D;&lt;task name&gt;&#x60; - &#x60;node&#x3D;&lt;node id or name&gt;&#x60; - &#x60;service&#x3D;&lt;service name&gt;&#x60;  | (optional) defaults to undefined|


### Return type

**Array<Task>**

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
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **taskLogs**
> File taskLogs()

Get `stdout` and `stderr` logs from a task. See also [`/containers/{id}/logs`](#operation/ContainerLogs).  **Note**: This endpoint works only for services with the `local`, `json-file` or `journald` logging drivers. 

### Example

```typescript
import {
    TaskApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new TaskApi(configuration);

let id: string; //ID of the task (default to undefined)
let details: boolean; //Show task context and extra details provided to logs. (optional) (default to false)
let follow: boolean; //Keep connection after returning logs. (optional) (default to false)
let stdout: boolean; //Return logs from `stdout` (optional) (default to false)
let stderr: boolean; //Return logs from `stderr` (optional) (default to false)
let since: number; //Only return logs since this time, as a UNIX timestamp (optional) (default to 0)
let timestamps: boolean; //Add timestamps to every log line (optional) (default to false)
let tail: string; //Only return this number of log lines from the end of the logs. Specify as an integer or `all` to output all log lines.  (optional) (default to 'all')

const { status, data } = await apiInstance.taskLogs(
    id,
    details,
    follow,
    stdout,
    stderr,
    since,
    timestamps,
    tail
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID of the task | defaults to undefined|
| **details** | [**boolean**] | Show task context and extra details provided to logs. | (optional) defaults to false|
| **follow** | [**boolean**] | Keep connection after returning logs. | (optional) defaults to false|
| **stdout** | [**boolean**] | Return logs from &#x60;stdout&#x60; | (optional) defaults to false|
| **stderr** | [**boolean**] | Return logs from &#x60;stderr&#x60; | (optional) defaults to false|
| **since** | [**number**] | Only return logs since this time, as a UNIX timestamp | (optional) defaults to 0|
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
|**200** | logs returned as a stream in response body |  -  |
|**404** | no such task |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

