# SwarmApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**swarmInit**](#swarminit) | **POST** /swarm/init | Initialize a new swarm|
|[**swarmInspect**](#swarminspect) | **GET** /swarm | Inspect swarm|
|[**swarmJoin**](#swarmjoin) | **POST** /swarm/join | Join an existing swarm|
|[**swarmLeave**](#swarmleave) | **POST** /swarm/leave | Leave a swarm|
|[**swarmUnlock**](#swarmunlock) | **POST** /swarm/unlock | Unlock a locked manager|
|[**swarmUnlockkey**](#swarmunlockkey) | **GET** /swarm/unlockkey | Get the unlock key|
|[**swarmUpdate**](#swarmupdate) | **POST** /swarm/update | Update a swarm|

# **swarmInit**
> string swarmInit(body)


### Example

```typescript
import {
    SwarmApi,
    Configuration,
    SwarmInitRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

let body: SwarmInitRequest; //

const { status, data } = await apiInstance.swarmInit(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SwarmInitRequest**|  | |


### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, text/plain
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**400** | bad parameter |  -  |
|**500** | server error |  -  |
|**503** | node is already part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmInspect**
> Swarm swarmInspect()


### Example

```typescript
import {
    SwarmApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

const { status, data } = await apiInstance.swarmInspect();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**Swarm**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such swarm |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmJoin**
> swarmJoin(body)


### Example

```typescript
import {
    SwarmApi,
    Configuration,
    SwarmJoinRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

let body: SwarmJoinRequest; //

const { status, data } = await apiInstance.swarmJoin(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SwarmJoinRequest**|  | |


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, text/plain
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**400** | bad parameter |  -  |
|**500** | server error |  -  |
|**503** | node is already part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmLeave**
> swarmLeave()


### Example

```typescript
import {
    SwarmApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

let force: boolean; //Force leave swarm, even if this is the last manager or that it will break the cluster.  (optional) (default to false)

const { status, data } = await apiInstance.swarmLeave(
    force
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **force** | [**boolean**] | Force leave swarm, even if this is the last manager or that it will break the cluster.  | (optional) defaults to false|


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
|**200** | no error |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmUnlock**
> swarmUnlock(body)


### Example

```typescript
import {
    SwarmApi,
    Configuration,
    SwarmUnlockRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

let body: SwarmUnlockRequest; //

const { status, data } = await apiInstance.swarmUnlock(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SwarmUnlockRequest**|  | |


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmUnlockkey**
> UnlockKeyResponse swarmUnlockkey()


### Example

```typescript
import {
    SwarmApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

const { status, data } = await apiInstance.swarmUnlockkey();
```

### Parameters
This endpoint does not have any parameters.


### Return type

**UnlockKeyResponse**

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
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **swarmUpdate**
> swarmUpdate(body)


### Example

```typescript
import {
    SwarmApi,
    Configuration,
    SwarmSpec
} from './api';

const configuration = new Configuration();
const apiInstance = new SwarmApi(configuration);

let version: number; //The version number of the swarm object being updated. This is required to avoid conflicting writes.  (default to undefined)
let body: SwarmSpec; //
let rotateWorkerToken: boolean; //Rotate the worker join token. (optional) (default to false)
let rotateManagerToken: boolean; //Rotate the manager join token. (optional) (default to false)
let rotateManagerUnlockKey: boolean; //Rotate the manager unlock key. (optional) (default to false)

const { status, data } = await apiInstance.swarmUpdate(
    version,
    body,
    rotateWorkerToken,
    rotateManagerToken,
    rotateManagerUnlockKey
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SwarmSpec**|  | |
| **version** | [**number**] | The version number of the swarm object being updated. This is required to avoid conflicting writes.  | defaults to undefined|
| **rotateWorkerToken** | [**boolean**] | Rotate the worker join token. | (optional) defaults to false|
| **rotateManagerToken** | [**boolean**] | Rotate the manager join token. | (optional) defaults to false|
| **rotateManagerUnlockKey** | [**boolean**] | Rotate the manager unlock key. | (optional) defaults to false|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, text/plain
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**400** | bad parameter |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

