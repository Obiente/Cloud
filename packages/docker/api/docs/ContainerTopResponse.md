# ContainerTopResponse

Container \"top\" response.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Titles** | **Array&lt;string&gt;** | The ps column titles | [optional] [default to undefined]
**Processes** | **Array&lt;Array&lt;string&gt;&gt;** | Each process running in the container, where each process is an array of values corresponding to the titles. | [optional] [default to undefined]

## Example

```typescript
import { ContainerTopResponse } from './api';

const instance: ContainerTopResponse = {
    Titles,
    Processes,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
