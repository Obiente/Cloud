# VolumeApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**volumeCreate**](#volumecreate) | **POST** /volumes/create | Create a volume|
|[**volumeDelete**](#volumedelete) | **DELETE** /volumes/{name} | Remove a volume|
|[**volumeInspect**](#volumeinspect) | **GET** /volumes/{name} | Inspect a volume|
|[**volumeList**](#volumelist) | **GET** /volumes | List volumes|
|[**volumePrune**](#volumeprune) | **POST** /volumes/prune | Delete unused volumes|
|[**volumeUpdate**](#volumeupdate) | **PUT** /volumes/{name} | \&quot;Update a volume. Valid only for Swarm cluster volumes\&quot; |

# **volumeCreate**
> Volume volumeCreate(volumeConfig)


### Example

```typescript
import {
    VolumeApi,
    Configuration,
    VolumeCreateOptions
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let volumeConfig: VolumeCreateOptions; //Volume configuration

const { status, data } = await apiInstance.volumeCreate(
    volumeConfig
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **volumeConfig** | **VolumeCreateOptions**| Volume configuration | |


### Return type

**Volume**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**201** | The volume was created successfully |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **volumeDelete**
> volumeDelete()

Instruct the driver to remove the volume.

### Example

```typescript
import {
    VolumeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let name: string; //Volume name or ID (default to undefined)
let force: boolean; //Force the removal of the volume (optional) (default to false)

const { status, data } = await apiInstance.volumeDelete(
    name,
    force
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | Volume name or ID | defaults to undefined|
| **force** | [**boolean**] | Force the removal of the volume | (optional) defaults to false|


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
|**204** | The volume was removed |  -  |
|**404** | No such volume or volume driver |  -  |
|**409** | Volume is in use and cannot be removed |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **volumeInspect**
> Volume volumeInspect()


### Example

```typescript
import {
    VolumeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let name: string; //Volume name or ID (default to undefined)

const { status, data } = await apiInstance.volumeInspect(
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | Volume name or ID | defaults to undefined|


### Return type

**Volume**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | No error |  -  |
|**404** | No such volume |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **volumeList**
> VolumeListResponse volumeList()


### Example

```typescript
import {
    VolumeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let filters: string; //JSON encoded value of the filters (a `map[string][]string`) to process on the volumes list. Available filters:  - `dangling=<boolean>` When set to `true` (or `1`), returns all    volumes that are not in use by a container. When set to `false`    (or `0`), only volumes that are in use by one or more    containers are returned. - `driver=<volume-driver-name>` Matches volumes based on their driver. - `label=<key>` or `label=<key>:<value>` Matches volumes based on    the presence of a `label` alone or a `label` and a value. - `name=<volume-name>` Matches all or part of a volume name.  (optional) (default to undefined)

const { status, data } = await apiInstance.volumeList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the volumes list. Available filters:  - &#x60;dangling&#x3D;&lt;boolean&gt;&#x60; When set to &#x60;true&#x60; (or &#x60;1&#x60;), returns all    volumes that are not in use by a container. When set to &#x60;false&#x60;    (or &#x60;0&#x60;), only volumes that are in use by one or more    containers are returned. - &#x60;driver&#x3D;&lt;volume-driver-name&gt;&#x60; Matches volumes based on their driver. - &#x60;label&#x3D;&lt;key&gt;&#x60; or &#x60;label&#x3D;&lt;key&gt;:&lt;value&gt;&#x60; Matches volumes based on    the presence of a &#x60;label&#x60; alone or a &#x60;label&#x60; and a value. - &#x60;name&#x3D;&lt;volume-name&gt;&#x60; Matches all or part of a volume name.  | (optional) defaults to undefined|


### Return type

**VolumeListResponse**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | Summary volume data that matches the query |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **volumePrune**
> VolumePruneResponse volumePrune()


### Example

```typescript
import {
    VolumeApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let filters: string; //Filters to process on the prune list, encoded as JSON (a `map[string][]string`).  Available filters: - `label` (`label=<key>`, `label=<key>=<value>`, `label!=<key>`, or `label!=<key>=<value>`) Prune volumes with (or without, in case `label!=...` is used) the specified labels. - `all` (`all=true`) - Consider all (local) volumes for pruning and not just anonymous volumes.  (optional) (default to undefined)

const { status, data } = await apiInstance.volumePrune(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | Filters to process on the prune list, encoded as JSON (a &#x60;map[string][]string&#x60;).  Available filters: - &#x60;label&#x60; (&#x60;label&#x3D;&lt;key&gt;&#x60;, &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;, &#x60;label!&#x3D;&lt;key&gt;&#x60;, or &#x60;label!&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;) Prune volumes with (or without, in case &#x60;label!&#x3D;...&#x60; is used) the specified labels. - &#x60;all&#x60; (&#x60;all&#x3D;true&#x60;) - Consider all (local) volumes for pruning and not just anonymous volumes.  | (optional) defaults to undefined|


### Return type

**VolumePruneResponse**

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

# **volumeUpdate**
> volumeUpdate()


### Example

```typescript
import {
    VolumeApi,
    Configuration,
    VolumeUpdateRequest
} from './api';

const configuration = new Configuration();
const apiInstance = new VolumeApi(configuration);

let name: string; //The name or ID of the volume (default to undefined)
let version: number; //The version number of the volume being updated. This is required to avoid conflicting writes. Found in the volume\'s `ClusterVolume` field.  (default to undefined)
let body: VolumeUpdateRequest; //The spec of the volume to update. Currently, only Availability may change. All other fields must remain unchanged.  (optional)

const { status, data } = await apiInstance.volumeUpdate(
    name,
    version,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **VolumeUpdateRequest**| The spec of the volume to update. Currently, only Availability may change. All other fields must remain unchanged.  | |
| **name** | [**string**] | The name or ID of the volume | defaults to undefined|
| **version** | [**number**] | The version number of the volume being updated. This is required to avoid conflicting writes. Found in the volume\&#39;s &#x60;ClusterVolume&#x60; field.  | defaults to undefined|


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
|**400** | bad parameter |  -  |
|**404** | no such volume |  -  |
|**500** | server error |  -  |
|**503** | node is not part of a swarm |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

