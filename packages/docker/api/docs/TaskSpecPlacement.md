# TaskSpecPlacement


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Constraints** | **Array&lt;string&gt;** | An array of constraint expressions to limit the set of nodes where a task can be scheduled. Constraint expressions can either use a _match_ (&#x60;&#x3D;&#x3D;&#x60;) or _exclude_ (&#x60;!&#x3D;&#x60;) rule. Multiple constraints find nodes that satisfy every expression (AND match). Constraints can match node or Docker Engine labels as follows:  node attribute       | matches                        | example ---------------------|--------------------------------|----------------------------------------------- &#x60;node.id&#x60;            | Node ID                        | &#x60;node.id&#x3D;&#x3D;2ivku8v2gvtg4&#x60; &#x60;node.hostname&#x60;      | Node hostname                  | &#x60;node.hostname!&#x3D;node-2&#x60; &#x60;node.role&#x60;          | Node role (&#x60;manager&#x60;/&#x60;worker&#x60;) | &#x60;node.role&#x3D;&#x3D;manager&#x60; &#x60;node.platform.os&#x60;   | Node operating system          | &#x60;node.platform.os&#x3D;&#x3D;windows&#x60; &#x60;node.platform.arch&#x60; | Node architecture              | &#x60;node.platform.arch&#x3D;&#x3D;x86_64&#x60; &#x60;node.labels&#x60;        | User-defined node labels       | &#x60;node.labels.security&#x3D;&#x3D;high&#x60; &#x60;engine.labels&#x60;      | Docker Engine\&#39;s labels         | &#x60;engine.labels.operatingsystem&#x3D;&#x3D;ubuntu-24.04&#x60;  &#x60;engine.labels&#x60; apply to Docker Engine labels like operating system, drivers, etc. Swarm administrators add &#x60;node.labels&#x60; for operational purposes by using the [&#x60;node update endpoint&#x60;](#operation/NodeUpdate).  | [optional] [default to undefined]
**Preferences** | [**Array&lt;TaskSpecPlacementPreferencesInner&gt;**](TaskSpecPlacementPreferencesInner.md) | Preferences provide a way to make the scheduler aware of factors such as topology. They are provided in order from highest to lowest precedence.  | [optional] [default to undefined]
**MaxReplicas** | **number** | Maximum number of replicas for per node (default value is 0, which is unlimited)  | [optional] [default to 0]
**Platforms** | [**Array&lt;Platform&gt;**](Platform.md) | Platforms stores all the platforms that the service\&#39;s image can run on. This field is used in the platform filter for scheduling. If empty, then the platform filter is off, meaning there are no scheduling restrictions.  | [optional] [default to undefined]

## Example

```typescript
import { TaskSpecPlacement } from './api';

const instance: TaskSpecPlacement = {
    Constraints,
    Preferences,
    MaxReplicas,
    Platforms,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
