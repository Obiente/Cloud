# NodeApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**nodeDelete**](#nodedelete) | **DELETE** /nodes/{id} | Delete a node|
|[**nodeInspect**](#nodeinspect) | **GET** /nodes/{id} | Inspect a node|
|[**nodeList**](#nodelist) | **GET** /nodes | List nodes|
|[**nodeUpdate**](#nodeupdate) | **POST** /nodes/{id}/update | Update a node|

# **nodeDelete**
> nodeDelete()


### Example

```typescript
import {
    NodeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NodeApi(configuration);

let id: string; //The ID or name of the node (default to undefined)
let force: boolean; //Force remove a node from the swarm (optional) (default to false)

const { status, data } = await apiInstance.nodeDelete(
    id,
    force
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | The ID or name of the node | defaults to undefined|
| **force** | [**boolean**] | Force remove a node from the swarm | (optional) defaults to false|


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
|**404** | no such node |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **nodeInspect**
> Node nodeInspect()


### Example

```typescript
import {
    NodeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NodeApi(configuration);

let id: string; //The ID or name of the node (default to undefined)

const { status, data } = await apiInstance.nodeInspect(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | The ID or name of the node | defaults to undefined|


### Return type

**Node**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | no such node |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **nodeList**
> Array<Node> nodeList()


### Example

```typescript
import {
    NodeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NodeApi(configuration);

let filters: string; //Filters to process on the nodes list, encoded as JSON (a `map[string][]string`).  Available filters: - `id=<node id>` - `label=<engine label>` - `membership=`(`accepted`|`pending`)` - `name=<node name>` - `node.label=<node label>` - `role=`(`manager`|`worker`)`  (optional) (default to undefined)

const { status, data } = await apiInstance.nodeList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | Filters to process on the nodes list, encoded as JSON (a &#x60;map[string][]string&#x60;).  Available filters: - &#x60;id&#x3D;&lt;node id&gt;&#x60; - &#x60;label&#x3D;&lt;engine label&gt;&#x60; - &#x60;membership&#x3D;&#x60;(&#x60;accepted&#x60;|&#x60;pending&#x60;)&#x60; - &#x60;name&#x3D;&lt;node name&gt;&#x60; - &#x60;node.label&#x3D;&lt;node label&gt;&#x60; - &#x60;role&#x3D;&#x60;(&#x60;manager&#x60;|&#x60;worker&#x60;)&#x60;  | (optional) defaults to undefined|


### Return type

**Array<Node>**

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

# **nodeUpdate**
> nodeUpdate()


### Example

```typescript
import {
    NodeApi,
    Configuration,
    NodeSpec
} from './api';

const configuration = new Configuration();
const apiInstance = new NodeApi(configuration);

let id: string; //The ID of the node (default to undefined)
let version: number; //The version number of the node object being updated. This is required to avoid conflicting writes.  (default to undefined)
let body: NodeSpec; // (optional)

const { status, data } = await apiInstance.nodeUpdate(
    id,
    version,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **NodeSpec**|  | |
| **id** | [**string**] | The ID of the node | defaults to undefined|
| **version** | [**number**] | The version number of the node object being updated. This is required to avoid conflicting writes.  | defaults to undefined|


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
|**404** | no such node |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

