# FirewallInfo

Information about the daemon\'s firewalling configuration.  This field is currently only used on Linux, and omitted on other platforms. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | **string** | The name of the firewall backend driver.  | [optional] [default to undefined]
**Info** | **Array&lt;Array&lt;string&gt;&gt;** | Information about the firewall backend, provided as \&quot;label\&quot; / \&quot;value\&quot; pairs.  &lt;p&gt;&lt;br /&gt;&lt;/p&gt;  &gt; **Note**: The information returned in this field, including the &gt; formatting of values and labels, should not be considered stable, &gt; and may change without notice.  | [optional] [default to undefined]

## Example

```typescript
import { FirewallInfo } from './api';

const instance: FirewallInfo = {
    Driver,
    Info,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
