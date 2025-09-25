# PluginApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**getPluginPrivileges**](#getpluginprivileges) | **GET** /plugins/privileges | Get plugin privileges|
|[**pluginCreate**](#plugincreate) | **POST** /plugins/create | Create a plugin|
|[**pluginDelete**](#plugindelete) | **DELETE** /plugins/{name} | Remove a plugin|
|[**pluginDisable**](#plugindisable) | **POST** /plugins/{name}/disable | Disable a plugin|
|[**pluginEnable**](#pluginenable) | **POST** /plugins/{name}/enable | Enable a plugin|
|[**pluginInspect**](#plugininspect) | **GET** /plugins/{name}/json | Inspect a plugin|
|[**pluginList**](#pluginlist) | **GET** /plugins | List plugins|
|[**pluginPull**](#pluginpull) | **POST** /plugins/pull | Install a plugin|
|[**pluginPush**](#pluginpush) | **POST** /plugins/{name}/push | Push a plugin|
|[**pluginSet**](#pluginset) | **POST** /plugins/{name}/set | Configure a plugin|
|[**pluginUpgrade**](#pluginupgrade) | **POST** /plugins/{name}/upgrade | Upgrade a plugin|

# **getPluginPrivileges**
> Array<PluginPrivilege> getPluginPrivileges()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let remote: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)

const { status, data } = await apiInstance.getPluginPrivileges(
    remote
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **remote** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|


### Return type

**Array<PluginPrivilege>**

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

# **pluginCreate**
> pluginCreate()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let tarContext: File; //Path to tar containing plugin rootfs and manifest (optional)

const { status, data } = await apiInstance.pluginCreate(
    name,
    tarContext
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **tarContext** | **File**| Path to tar containing plugin rootfs and manifest | |
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/x-tar
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginDelete**
> Plugin pluginDelete()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let force: boolean; //Disable the plugin before removing. This may result in issues if the plugin is in use by a container.  (optional) (default to false)

const { status, data } = await apiInstance.pluginDelete(
    name,
    force
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|
| **force** | [**boolean**] | Disable the plugin before removing. This may result in issues if the plugin is in use by a container.  | (optional) defaults to false|


### Return type

**Plugin**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | plugin is not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginDisable**
> pluginDisable()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let force: boolean; //Force disable a plugin even if still in use.  (optional) (default to undefined)

const { status, data } = await apiInstance.pluginDisable(
    name,
    force
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|
| **force** | [**boolean**] | Force disable a plugin even if still in use.  | (optional) defaults to undefined|


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
|**404** | plugin is not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginEnable**
> pluginEnable()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let timeout: number; //Set the HTTP client timeout (in seconds) (optional) (default to 0)

const { status, data } = await apiInstance.pluginEnable(
    name,
    timeout
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|
| **timeout** | [**number**] | Set the HTTP client timeout (in seconds) | (optional) defaults to 0|


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
|**404** | plugin is not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginInspect**
> Plugin pluginInspect()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)

const { status, data } = await apiInstance.pluginInspect(
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|


### Return type

**Plugin**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | no error |  -  |
|**404** | plugin is not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginList**
> Array<Plugin> pluginList()

Returns information about installed plugins.

### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let filters: string; //A JSON encoded value of the filters (a `map[string][]string`) to process on the plugin list.  Available filters:  - `capability=<capability name>` - `enable=<true>|<false>`  (optional) (default to undefined)

const { status, data } = await apiInstance.pluginList(
    filters
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **filters** | [**string**] | A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the plugin list.  Available filters:  - &#x60;capability&#x3D;&lt;capability name&gt;&#x60; - &#x60;enable&#x3D;&lt;true&gt;|&lt;false&gt;&#x60;  | (optional) defaults to undefined|


### Return type

**Array<Plugin>**

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

# **pluginPull**
> pluginPull()

Pulls and installs a plugin. After the plugin is installed, it can be enabled using the [`POST /plugins/{name}/enable` endpoint](#operation/PostPluginsEnable). 

### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let remote: string; //Remote reference for plugin to install.  The `:latest` tag is optional, and is used as the default if omitted.  (default to undefined)
let name: string; //Local name for the pulled plugin.  The `:latest` tag is optional, and is used as the default if omitted.  (optional) (default to undefined)
let xRegistryAuth: string; //A base64url-encoded auth configuration to use when pulling a plugin from a registry.  Refer to the [authentication section](#section/Authentication) for details.  (optional) (default to undefined)
let body: Array<PluginPrivilege>; // (optional)

const { status, data } = await apiInstance.pluginPull(
    remote,
    name,
    xRegistryAuth,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **Array<PluginPrivilege>**|  | |
| **remote** | [**string**] | Remote reference for plugin to install.  The &#x60;:latest&#x60; tag is optional, and is used as the default if omitted.  | defaults to undefined|
| **name** | [**string**] | Local name for the pulled plugin.  The &#x60;:latest&#x60; tag is optional, and is used as the default if omitted.  | (optional) defaults to undefined|
| **xRegistryAuth** | [**string**] | A base64url-encoded auth configuration to use when pulling a plugin from a registry.  Refer to the [authentication section](#section/Authentication) for details.  | (optional) defaults to undefined|


### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, text/plain
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**204** | no error |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginPush**
> pluginPush()

Push a plugin to the registry. 

### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)

const { status, data } = await apiInstance.pluginPush(
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|


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
|**404** | plugin not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginSet**
> pluginSet()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let body: Array<string>; // (optional)

const { status, data } = await apiInstance.pluginSet(
    name,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **Array<string>**|  | |
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|


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
|**204** | No error |  -  |
|**404** | Plugin not installed |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **pluginUpgrade**
> pluginUpgrade()


### Example

```typescript
import {
    PluginApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new PluginApi(configuration);

let name: string; //The name of the plugin. The `:latest` tag is optional, and is the default if omitted.  (default to undefined)
let remote: string; //Remote reference to upgrade to.  The `:latest` tag is optional, and is used as the default if omitted.  (default to undefined)
let xRegistryAuth: string; //A base64url-encoded auth configuration to use when pulling a plugin from a registry.  Refer to the [authentication section](#section/Authentication) for details.  (optional) (default to undefined)
let body: Array<PluginPrivilege>; // (optional)

const { status, data } = await apiInstance.pluginUpgrade(
    name,
    remote,
    xRegistryAuth,
    body
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **body** | **Array<PluginPrivilege>**|  | |
| **name** | [**string**] | The name of the plugin. The &#x60;:latest&#x60; tag is optional, and is the default if omitted.  | defaults to undefined|
| **remote** | [**string**] | Remote reference to upgrade to.  The &#x60;:latest&#x60; tag is optional, and is used as the default if omitted.  | defaults to undefined|
| **xRegistryAuth** | [**string**] | A base64url-encoded auth configuration to use when pulling a plugin from a registry.  Refer to the [authentication section](#section/Authentication) for details.  | (optional) defaults to undefined|


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
|**204** | no error |  -  |
|**404** | plugin not installed |  -  |
|**500** | server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

