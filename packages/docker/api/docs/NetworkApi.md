# NetworkApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**networkConnect**](#networkconnect) | **POST** /networks/{id}/connect | Connect a container to a network|
|[**networkCreate**](#networkcreate) | **POST** /networks/create | Create a network|
|[**networkDelete**](#networkdelete) | **DELETE** /networks/{id} | Remove a network|
|[**networkDisconnect**](#networkdisconnect) | **POST** /networks/{id}/disconnect | Disconnect a container from a network|
|[**networkInspect**](#networkinspect) | **GET** /networks/{id} | Inspect a network|
|[**networkList**](#networklist) | **GET** /networks | List networks|
|[**networkPrune**](#networkprune) | **POST** /networks/prune | Delete unused networks|

# **networkConnect**
> networkConnect(container)

The network must be either a local-scoped network or a swarm-scoped network with the `attachable` option set. A network cannot be re-attached to a running container

### Example

```typescript
import {
    NetworkApi,
    Configuration,
    NetworkConnectRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let id: string; //Network ID or name (default to undefined)
let container: NetworkConnectRequest; //

const { status, data } = await apiInstance.networkConnect(
    id,
    container
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **container** | **NetworkConnectRequest**|  | |
| **id** | [**string**] | Network ID or name | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**400** | bad parameter |  -  |
|**403** | Operation forbidden |  -  |
|**404** | Network or container not found |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **networkCreate**
> NetworkCreateResponse networkCreate(networkConfig)


### Example

```typescript
import {
    NetworkApi,
    Configuration,
    NetworkCreateRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let networkConfig: NetworkCreateRequest; //Network configuration

const { status, data } = await apiInstance.networkCreate(
    networkConfig
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **networkConfig** | **NetworkCreateRequest**| Network configuration | |


### Return type

**NetworkCreateResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**201** | Network created successfully |  -  |
|**400** | bad parameter |  -  |
|**403** | Forbidden operation. This happens when trying to create a network named after a pre-defined network, or when trying to create an overlay network on a daemon which is not part of a Swarm cluster.  |  -  |
|**404** | plugin not found |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **networkDelete**
> networkDelete()


### Example

```typescript
import {
    NetworkApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let id: string; //Network ID or name (default to undefined)

const { status, data } = await apiInstance.networkDelete(
    id
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | Network ID or name | defaults to undefined|


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
|**204** | No error |  -  |
|**403** | operation not supported for pre-defined networks |  -  |
|**404** | no such network |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **networkDisconnect**
> networkDisconnect(container)


### Example

```typescript
import {
    NetworkApi,
    Configuration,
    NetworkDisconnectRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let id: string; //Network ID or name (default to undefined)
let container: NetworkDisconnectRequest; //

const { status, data } = await apiInstance.networkDisconnect(
    id,
    container
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **container** | **NetworkDisconnectRequest**|  | |
| **id** | [**string**] | Network ID or name | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**403** | Operation not supported for swarm scoped networks |  -  |
|**404** | Network or container not found |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **networkInspect**
> Network networkInspect()


### Example

```typescript
import {
    NetworkApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let id: string; //Network ID or name (default to undefined)
let verbose: boolean; //Detailed inspect output for troubleshooting (optional) (default to false)
let scope: string; //Filter the network by scope (swarm, global, or local) (optional) (default to undefined)

const { status, data } = await apiInstance.networkInspect(
    id,
    verbose,
    scope
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **id** | [**string**] | Network ID or name | defaults to undefined|
| **verbose** | [**boolean**] | Detailed inspect output for troubleshooting | (optional) defaults to false|
| **scope** | [**string**] | Filter the network by scope (swarm, global, or local) | (optional) defaults to undefined|


### Return type

**Network**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**404** | Network not found |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **networkList**
> Array<Network> networkList()

Returns a list of networks. For details on the format, see the [network inspect endpoint](#operation/NetworkInspect).  Note that it uses a different, smaller representation of a network than inspecting a single network. For example, the list of containers attached to the network is not propagated in API versions 1.28 and up. 

### Example

```typescript
import {
    NetworkApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let filters: string; //JSON encoded value of the filters (a `map[string][]string`) to process on the networks list.  Available filters:  - `dangling=<boolean>` When set to `true` (or `1`), returns all    networks that are not in use by a container. When set to `false`    (or `0`), only networks that are in use by one or more    containers are returned. - `driver=<driver-name>` Matches a network\'s driver. - `id=<network-id>` Matches all or part of a network ID. - `label=<key>` or `label=<key>=<value>` of a network label. - `name=<network-name>` Matches all or part of a network name. - `scope=[\"swarm\"|\"global\"|\"local\"]` Filters networks by scope (`swarm`, `global`, or `local`). - `type=[\"custom\"|\"builtin\"]` Filters networks by type. The `custom` keyword returns all user-defined networks.  (optional) (default to undefined)

const { status, data } = await apiInstance.networkList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the networks list.  Available filters:  - &#x60;dangling&#x3D;&lt;boolean&gt;&#x60; When set to &#x60;true&#x60; (or &#x60;1&#x60;), returns all    networks that are not in use by a container. When set to &#x60;false&#x60;    (or &#x60;0&#x60;), only networks that are in use by one or more    containers are returned. - &#x60;driver&#x3D;&lt;driver-name&gt;&#x60; Matches a network\&#39;s driver. - &#x60;id&#x3D;&lt;network-id&gt;&#x60; Matches all or part of a network ID. - &#x60;label&#x3D;&lt;key&gt;&#x60; or &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60; of a network label. - &#x60;name&#x3D;&lt;network-name&gt;&#x60; Matches all or part of a network name. - &#x60;scope&#x3D;[\&quot;swarm\&quot;|\&quot;global\&quot;|\&quot;local\&quot;]&#x60; Filters networks by scope (&#x60;swarm&#x60;, &#x60;global&#x60;, or &#x60;local&#x60;). - &#x60;type&#x3D;[\&quot;custom\&quot;|\&quot;builtin\&quot;]&#x60; Filters networks by type. The &#x60;custom&#x60; keyword returns all user-defined networks.  | (optional) defaults to undefined|


### Return type

**Array<Network>**

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

# **networkPrune**
> NetworkPruneResponse networkPrune()


### Example

```typescript
import {
    NetworkApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new NetworkApi(configuration);

let filters: string; //Filters to process on the prune list, encoded as JSON (a `map[string][]string`).  Available filters: - `until=<timestamp>` Prune networks created before this timestamp. The `<timestamp>` can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. `10m`, `1h30m`) computed relative to the daemon machine’s time. - `label` (`label=<key>`, `label=<key>=<value>`, `label!=<key>`, or `label!=<key>=<value>`) Prune networks with (or without, in case `label!=...` is used) the specified labels.  (optional) (default to undefined)

const { status, data } = await apiInstance.networkPrune(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | Filters to process on the prune list, encoded as JSON (a &#x60;map[string][]string&#x60;).  Available filters: - &#x60;until&#x3D;&lt;timestamp&gt;&#x60; Prune networks created before this timestamp. The &#x60;&lt;timestamp&gt;&#x60; can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. &#x60;10m&#x60;, &#x60;1h30m&#x60;) computed relative to the daemon machine’s time. - &#x60;label&#x60; (&#x60;label&#x3D;&lt;key&gt;&#x60;, &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;, &#x60;label!&#x3D;&lt;key&gt;&#x60;, or &#x60;label!&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;) Prune networks with (or without, in case &#x60;label!&#x3D;...&#x60; is used) the specified labels.  | (optional) defaults to undefined|


### Return type

**NetworkPruneResponse**

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

