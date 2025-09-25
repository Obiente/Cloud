# TaskSpecContainerSpecSecretsInnerFile

File represents a specific target that is backed by a file. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** | Name represents the final filename in the filesystem.  | [optional] [default to undefined]
**UID** | **string** | UID represents the file UID. | [optional] [default to undefined]
**GID** | **string** | GID represents the file GID. | [optional] [default to undefined]
**Mode** | **number** | Mode represents the FileMode of the file. | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecSecretsInnerFile } from './api';

const instance: TaskSpecContainerSpecSecretsInnerFile = {
    Name,
    UID,
    GID,
    Mode,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
