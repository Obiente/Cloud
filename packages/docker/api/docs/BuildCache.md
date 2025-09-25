# BuildCache

BuildCache contains information about a build cache record. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ID** | **string** | Unique ID of the build cache record.  | [optional] [default to undefined]
**Parent** | **string** | ID of the parent build cache record.  &gt; **Deprecated**: This field is deprecated, and omitted if empty.  | [optional] [default to undefined]
**Parents** | **Array&lt;string&gt;** | List of parent build cache record IDs.  | [optional] [default to undefined]
**Type** | **string** | Cache record type.  | [optional] [default to undefined]
**Description** | **string** | Description of the build-step that produced the build cache.  | [optional] [default to undefined]
**InUse** | **boolean** | Indicates if the build cache is in use.  | [optional] [default to undefined]
**Shared** | **boolean** | Indicates if the build cache is shared.  | [optional] [default to undefined]
**Size** | **number** | Amount of disk space used by the build cache (in bytes).  | [optional] [default to undefined]
**CreatedAt** | **string** | Date and time at which the build cache was created in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**LastUsedAt** | **string** | Date and time at which the build cache was last used in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  | [optional] [default to undefined]
**UsageCount** | **number** |  | [optional] [default to undefined]

## Example

```typescript
import { BuildCache } from './api';

const instance: BuildCache = {
    ID,
    Parent,
    Parents,
    Type,
    Description,
    InUse,
    Shared,
    Size,
    CreatedAt,
    LastUsedAt,
    UsageCount,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
