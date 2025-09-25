# SecretApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**secretCreate**](#secretcreate) | **POST** /secrets/create | Create a secret|
|[**secretDelete**](#secretdelete) | **DELETE** /secrets/{id} | Delete a secret|
|[**secretInspect**](#secretinspect) | **GET** /secrets/{id} | Inspect a secret|
|[**secretList**](#secretlist) | **GET** /secrets | List secrets|
|[**secretUpdate**](#secretupdate) | **POST** /secrets/{id}/update | Update a Secret|

# **secretCreate**
> IDResponse secretCreate()


### Example

```typescript
import {
    SecretApi,
    Configuration,
    SecretCreateRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new SecretApi(configuration);

let body: SecretCreateRequest; // (optional)

const { status, data } = await apiInstance.secretCreate(
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SecretCreateRequest**|  | |


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
|**409** | name conflicts with an existing object |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **secretDelete**
> secretDelete()


### Example

```typescript
import {
    SecretApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SecretApi(configuration);

let id: string; //ID of the secret (default to undefined)

const { status, data } = await apiInstance.secretDelete(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID of the secret | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**404** | secret not found |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **secretInspect**
> Secret secretInspect()


### Example

```typescript
import {
    SecretApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SecretApi(configuration);

let id: string; //ID of the secret (default to undefined)

const { status, data } = await apiInstance.secretInspect(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | ID of the secret | defaults to undefined|


### Return type

**Secret**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | secret not found |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **secretList**
> Array<Secret> secretList()


### Example

```typescript
import {
    SecretApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new SecretApi(configuration);

let filters: string; //A JSON encoded value of the filters (a `map[string][]string`) to process on the secrets list.  Available filters:  - `id=<secret id>` - `label=<key> or label=<key>=value` - `name=<secret name>` - `names=<secret name>`  (optional) (default to undefined)

const { status, data } = await apiInstance.secretList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the secrets list.  Available filters:  - &#x60;id&#x3D;&lt;secret id&gt;&#x60; - &#x60;label&#x3D;&lt;key&gt; or label&#x3D;&lt;key&gt;&#x3D;value&#x60; - &#x60;name&#x3D;&lt;secret name&gt;&#x60; - &#x60;names&#x3D;&lt;secret name&gt;&#x60;  | (optional) defaults to undefined|


### Return type

**Array<Secret>**

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

# **secretUpdate**
> secretUpdate()


### Example

```typescript
import {
    SecretApi,
    Configuration,
    SecretSpec
} from './api';

const configuration = new Configuration();
const apiInstance = new SecretApi(configuration);

let id: string; //The ID or name of the secret (default to undefined)
let version: number; //The version number of the secret object being updated. This is required to avoid conflicting writes.  (default to undefined)
let body: SecretSpec; //The spec of the secret to update. Currently, only the Labels field can be updated. All other fields must remain unchanged from the [SecretInspect endpoint](#operation/SecretInspect) response values.  (optional)

const { status, data } = await apiInstance.secretUpdate(
    id,
    version,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **SecretSpec**| The spec of the secret to update. Currently, only the Labels field can be updated. All other fields must remain unchanged from the [SecretInspect endpoint](#operation/SecretInspect) response values.  | |
| **id** | [**string**] | The ID or name of the secret | defaults to undefined|
| **version** | [**number**] | The version number of the secret object being updated. This is required to avoid conflicting writes.  | defaults to undefined|


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
|**404** | no such secret |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

