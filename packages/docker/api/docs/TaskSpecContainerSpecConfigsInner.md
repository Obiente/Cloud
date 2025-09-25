# TaskSpecContainerSpecConfigsInner


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**File** | [**TaskSpecContainerSpecConfigsInnerFile**](TaskSpecContainerSpecConfigsInnerFile.md) |  | [optional] [default to undefined]
**Runtime** | **object** | Runtime represents a target that is not mounted into the container but is used by the task  &lt;p&gt;&lt;br /&gt;&lt;p&gt;  &gt; **Note**: &#x60;Configs.File&#x60; and &#x60;Configs.Runtime&#x60; are mutually &gt; exclusive  | [optional] [default to undefined]
**ConfigID** | **string** | ConfigID represents the ID of the specific config that we\&#39;re referencing.  | [optional] [default to undefined]
**ConfigName** | **string** | ConfigName is the name of the config that this references, but this is just provided for lookup/display purposes. The config in the reference will be identified by its ID.  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecContainerSpecConfigsInner } from './api';

const instance: TaskSpecContainerSpecConfigsInner = {
    File,
    Runtime,
    ConfigID,
    ConfigName,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
