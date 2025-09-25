# DistributionApi

All URIs are relative to *http://localhost/v1.51*

|Method | HTTP request | Description|
|------------- | ------------- | -------------|
|[**distributionInspect**](#distributioninspect) | **GET** /distribution/{name}/json | Get image information from the registry|

# **distributionInspect**
> DistributionInspect distributionInspect()

Return image digest and platform information by contacting the registry. 

### Example

```typescript
import {
    DistributionApi,
    Configuration
} from './api';

const configuration = new Configuration();
const apiInstance = new DistributionApi(configuration);

let name: string; //Image name or id (default to undefined)

const { status, data } = await apiInstance.distributionInspect(
    name
);
```

### Parameters

|Name | Type | Description  | Notes|
|------------- | ------------- | ------------- | -------------|
| **name** | [**string**] | Image name or id | defaults to undefined|


### Return type

**DistributionInspect**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
|**200** | descriptor and platform information |  -  |
|**401** | Failed authentication or no image found |  -  |
|**500** | Server error |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

