# ExecApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**containerExec**](#containerexec) | **POST** /containers/{id}/exec | Create an exec instance|
|[**execInspect**](#execinspect) | **GET** /exec/{id}/json | Inspect an exec instance|
|[**execResize**](#execresize) | **POST** /exec/{id}/resize | Resize an exec instance|
|[**execStart**](#execstart) | **POST** /exec/{id}/start | Start an exec instance|

# **containerExec**
> IDResponse containerExec(execConfig)

Run a command inside a running container.

### Example

```typescript
import {
    ExecApi,
    Configuration,
    ExecConfig
} from './api';

const configuration = new Configuration();
const apiInstance = new ExecApi(configuration);

let id: string; //ID or name of container (default to undefined)
let execConfig: ExecConfig; //Exec configuration

const { status, data } = await apiInstance.containerExec(
    id,
    execConfig
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **execConfig** | **ExecConfig**| Exec configuration | |
| **id** | [**string**] | ID or name of container | defaults to undefined|


### Return type

**IDResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**201** | no error |  -  |
|**404** | no such container |  -  |
|**409** | container is paused |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **execInspect**
> ExecInspectResponse execInspect()

Return low-level information about an exec instance.

### Example

```typescript
import {
    ExecApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ExecApi(configuration);

let id: string; //Exec instance ID (default to undefined)

const { status, data } = await apiInstance.execInspect(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | Exec instance ID | defaults to undefined|


### Return type

**ExecInspectResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**404** | No such exec instance |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **execResize**
> execResize()

Resize the TTY session used by an exec instance. This endpoint only works if `tty` was specified as part of creating and starting the exec instance. 

### Example

```typescript
import {
    ExecApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new ExecApi(configuration);

let id: string; //Exec instance ID (default to undefined)
let h: number; //Height of the TTY session in characters (default to undefined)
let w: number; //Width of the TTY session in characters (default to undefined)

const { status, data } = await apiInstance.execResize(
    id,
    h,
    w
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | Exec instance ID | defaults to undefined|
| **h** | [**number**] | Height of the TTY session in characters | defaults to undefined|
| **w** | [**number**] | Width of the TTY session in characters | defaults to undefined|


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
|**200** | No error |  -  |
|**400** | bad parameter |  -  |
|**404** | No such exec instance |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **execStart**
> execStart()

Starts a previously set up exec instance. If detach is true, this endpoint returns immediately after starting the command. Otherwise, it sets up an interactive session with the command. 

### Example

```typescript
import {
    ExecApi,
    Configuration,
    ExecStartConfig
} from './api';

const configuration = new Configuration();
const apiInstance = new ExecApi(configuration);

let id: string; //Exec instance ID (default to undefined)
let execStartConfig: ExecStartConfig; // (optional)

const { status, data } = await apiInstance.execStart(
    id,
    execStartConfig
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **execStartConfig** | **ExecStartConfig**|  | |
| **id** | [**string**] | Exec instance ID | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/vnd.docker.raw-stream, application/vnd.docker.multiplexed-stream


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**404** | No such exec instance |  -  |
|**409** | Container is stopped or paused |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

