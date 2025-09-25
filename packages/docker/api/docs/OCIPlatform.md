# OCIPlatform

Describes the platform which the image in the manifest runs on, as defined in the [OCI Image Index Specification](https://github.com/opencontainers/image-spec/blob/v1.0.1/image-index.md). 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**architecture** | **string** | The CPU architecture, for example &#x60;amd64&#x60; or &#x60;ppc64&#x60;.  | [optional] [default to undefined]
**os** | **string** | The operating system, for example &#x60;linux&#x60; or &#x60;windows&#x60;.  | [optional] [default to undefined]
**os_version** | **string** | Optional field specifying the operating system version, for example on Windows &#x60;10.0.19041.1165&#x60;.  | [optional] [default to undefined]
**os_features** | **Array&lt;string&gt;** | Optional field specifying an array of strings, each listing a required OS feature (for example on Windows &#x60;win32k&#x60;).  | [optional] [default to undefined]
**variant** | **string** | Optional field specifying a variant of the CPU, for example &#x60;v7&#x60; to specify ARMv7 when architecture is &#x60;arm&#x60;.  | [optional] [default to undefined]

## Example

```typescript
import { OCIPlatform } from './api';

const instance: OCIPlatform = {
    architecture,
    os,
    os_version,
    os_features,
    variant,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
