# SystemVersion

Response of Engine API: GET \"/version\" 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Platform** | [**SystemVersionPlatform**](SystemVersionPlatform.md) |  | [optional] [default to undefined]
**Components** | [**Array&lt;SystemVersionComponentsInner&gt;**](SystemVersionComponentsInner.md) | Information about system components  | [optional] [default to undefined]
**Version** | **string** | The version of the daemon | [optional] [default to undefined]
**ApiVersion** | **string** | The default (and highest) API version that is supported by the daemon  | [optional] [default to undefined]
**MinAPIVersion** | **string** | The minimum API version that is supported by the daemon  | [optional] [default to undefined]
**GitCommit** | **string** | The Git commit of the source code that was used to build the daemon  | [optional] [default to undefined]
**GoVersion** | **string** | The version Go used to compile the daemon, and the version of the Go runtime in use.  | [optional] [default to undefined]
**Os** | **string** | The operating system that the daemon is running on (\&quot;linux\&quot; or \&quot;windows\&quot;)  | [optional] [default to undefined]
**Arch** | **string** | The architecture that the daemon is running on  | [optional] [default to undefined]
**KernelVersion** | **string** | The kernel version (&#x60;uname -r&#x60;) that the daemon is running on.  This field is omitted when empty.  | [optional] [default to undefined]
**Experimental** | **boolean** | Indicates if the daemon is started with experimental features enabled.  This field is omitted when empty / false.  | [optional] [default to undefined]
**BuildTime** | **string** | The date and time that the daemon was compiled.  | [optional] [default to undefined]

## Example

```typescript
import { SystemVersion } from './api';

const instance: SystemVersion = {
    Platform,
    Components,
    Version,
    ApiVersion,
    MinAPIVersion,
    GitCommit,
    GoVersion,
    Os,
    Arch,
    KernelVersion,
    Experimental,
    BuildTime,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
