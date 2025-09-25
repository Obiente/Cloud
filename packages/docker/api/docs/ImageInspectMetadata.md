# ImageInspectMetadata

Additional metadata of the image in the local cache. This information is local to the daemon, and not part of the image itself. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LastTagTime** | **string** | Date and time at which the image was last tagged in [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format with nano-seconds.  This information is only available if the image was tagged locally, and omitted otherwise.  | [optional] [default to undefined]

## Example

```typescript
import { ImageInspectMetadata } from './api';

const instance: ImageInspectMetadata = {
    LastTagTime,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
