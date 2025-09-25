# HostConfig

Container configuration that depends on the host we are running on

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CpuShares** | **number** | An integer value representing this container\&#39;s relative CPU weight versus other containers.  | [optional] [default to undefined]
**Memory** | **number** | Memory limit in bytes. | [optional] [default to 0]
**CgroupParent** | **string** | Path to &#x60;cgroups&#x60; under which the container\&#39;s &#x60;cgroup&#x60; is created. If the path is not absolute, the path is considered to be relative to the &#x60;cgroups&#x60; path of the init process. Cgroups are created if they do not already exist.  | [optional] [default to undefined]
**BlkioWeight** | **number** | Block IO weight (relative weight). | [optional] [default to undefined]
**BlkioWeightDevice** | [**Array&lt;ResourcesBlkioWeightDeviceInner&gt;**](ResourcesBlkioWeightDeviceInner.md) | Block IO weight (relative device weight) in the form:  &#x60;&#x60;&#x60; [{\&quot;Path\&quot;: \&quot;device_path\&quot;, \&quot;Weight\&quot;: weight}] &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**BlkioDeviceReadBps** | [**Array&lt;ThrottleDevice&gt;**](ThrottleDevice.md) | Limit read rate (bytes per second) from a device, in the form:  &#x60;&#x60;&#x60; [{\&quot;Path\&quot;: \&quot;device_path\&quot;, \&quot;Rate\&quot;: rate}] &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**BlkioDeviceWriteBps** | [**Array&lt;ThrottleDevice&gt;**](ThrottleDevice.md) | Limit write rate (bytes per second) to a device, in the form:  &#x60;&#x60;&#x60; [{\&quot;Path\&quot;: \&quot;device_path\&quot;, \&quot;Rate\&quot;: rate}] &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**BlkioDeviceReadIOps** | [**Array&lt;ThrottleDevice&gt;**](ThrottleDevice.md) | Limit read rate (IO per second) from a device, in the form:  &#x60;&#x60;&#x60; [{\&quot;Path\&quot;: \&quot;device_path\&quot;, \&quot;Rate\&quot;: rate}] &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**BlkioDeviceWriteIOps** | [**Array&lt;ThrottleDevice&gt;**](ThrottleDevice.md) | Limit write rate (IO per second) to a device, in the form:  &#x60;&#x60;&#x60; [{\&quot;Path\&quot;: \&quot;device_path\&quot;, \&quot;Rate\&quot;: rate}] &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**CpuPeriod** | **number** | The length of a CPU period in microseconds. | [optional] [default to undefined]
**CpuQuota** | **number** | Microseconds of CPU time that the container can get in a CPU period.  | [optional] [default to undefined]
**CpuRealtimePeriod** | **number** | The length of a CPU real-time period in microseconds. Set to 0 to allocate no time allocated to real-time tasks.  | [optional] [default to undefined]
**CpuRealtimeRuntime** | **number** | The length of a CPU real-time runtime in microseconds. Set to 0 to allocate no time allocated to real-time tasks.  | [optional] [default to undefined]
**CpusetCpus** | **string** | CPUs in which to allow execution (e.g., &#x60;0-3&#x60;, &#x60;0,1&#x60;).  | [optional] [default to undefined]
**CpusetMems** | **string** | Memory nodes (MEMs) in which to allow execution (0-3, 0,1). Only effective on NUMA systems.  | [optional] [default to undefined]
**Devices** | [**Array&lt;DeviceMapping&gt;**](DeviceMapping.md) | A list of devices to add to the container. | [optional] [default to undefined]
**DeviceCgroupRules** | **Array&lt;string&gt;** | a list of cgroup rules to apply to the container | [optional] [default to undefined]
**DeviceRequests** | [**Array&lt;DeviceRequest&gt;**](DeviceRequest.md) | A list of requests for devices to be sent to device drivers.  | [optional] [default to undefined]
**KernelMemoryTCP** | **number** | Hard limit for kernel TCP buffer memory (in bytes). Depending on the OCI runtime in use, this option may be ignored. It is no longer supported by the default (runc) runtime.  This field is omitted when empty.  | [optional] [default to undefined]
**MemoryReservation** | **number** | Memory soft limit in bytes. | [optional] [default to undefined]
**MemorySwap** | **number** | Total memory limit (memory + swap). Set as &#x60;-1&#x60; to enable unlimited swap.  | [optional] [default to undefined]
**MemorySwappiness** | **number** | Tune a container\&#39;s memory swappiness behavior. Accepts an integer between 0 and 100.  | [optional] [default to undefined]
**NanoCpus** | **number** | CPU quota in units of 10&lt;sup&gt;-9&lt;/sup&gt; CPUs. | [optional] [default to undefined]
**OomKillDisable** | **boolean** | Disable OOM Killer for the container. | [optional] [default to undefined]
**Init** | **boolean** | Run an init inside the container that forwards signals and reaps processes. This field is omitted if empty, and the default (as configured on the daemon) is used.  | [optional] [default to undefined]
**PidsLimit** | **number** | Tune a container\&#39;s PIDs limit. Set &#x60;0&#x60; or &#x60;-1&#x60; for unlimited, or &#x60;null&#x60; to not change.  | [optional] [default to undefined]
**Ulimits** | [**Array&lt;ResourcesUlimitsInner&gt;**](ResourcesUlimitsInner.md) | A list of resource limits to set in the container. For example:  &#x60;&#x60;&#x60; {\&quot;Name\&quot;: \&quot;nofile\&quot;, \&quot;Soft\&quot;: 1024, \&quot;Hard\&quot;: 2048} &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**CpuCount** | **number** | The number of usable CPUs (Windows only).  On Windows Server containers, the processor resource controls are mutually exclusive. The order of precedence is &#x60;CPUCount&#x60; first, then &#x60;CPUShares&#x60;, and &#x60;CPUPercent&#x60; last.  | [optional] [default to undefined]
**CpuPercent** | **number** | The usable percentage of the available CPUs (Windows only).  On Windows Server containers, the processor resource controls are mutually exclusive. The order of precedence is &#x60;CPUCount&#x60; first, then &#x60;CPUShares&#x60;, and &#x60;CPUPercent&#x60; last.  | [optional] [default to undefined]
**IOMaximumIOps** | **number** | Maximum IOps for the container system drive (Windows only) | [optional] [default to undefined]
**IOMaximumBandwidth** | **number** | Maximum IO in bytes per second for the container system drive (Windows only).  | [optional] [default to undefined]
**Binds** | **Array&lt;string&gt;** | A list of volume bindings for this container. Each volume binding is a string in one of these forms:  - &#x60;host-src:container-dest[:options]&#x60; to bind-mount a host path   into the container. Both &#x60;host-src&#x60;, and &#x60;container-dest&#x60; must   be an _absolute_ path. - &#x60;volume-name:container-dest[:options]&#x60; to bind-mount a volume   managed by a volume driver into the container. &#x60;container-dest&#x60;   must be an _absolute_ path.  &#x60;options&#x60; is an optional, comma-delimited list of:  - &#x60;nocopy&#x60; disables automatic copying of data from the container   path to the volume. The &#x60;nocopy&#x60; flag only applies to named volumes. - &#x60;[ro|rw]&#x60; mounts a volume read-only or read-write, respectively.   If omitted or set to &#x60;rw&#x60;, volumes are mounted read-write. - &#x60;[z|Z]&#x60; applies SELinux labels to allow or deny multiple containers   to read and write to the same volume.     - &#x60;z&#x60;: a _shared_ content label is applied to the content. This       label indicates that multiple containers can share the volume       content, for both reading and writing.     - &#x60;Z&#x60;: a _private unshared_ label is applied to the content.       This label indicates that only the current container can use       a private volume. Labeling systems such as SELinux require       proper labels to be placed on volume content that is mounted       into a container. Without a label, the security system can       prevent a container\&#39;s processes from using the content. By       default, the labels set by the host operating system are not       modified. - &#x60;[[r]shared|[r]slave|[r]private]&#x60; specifies mount   [propagation behavior](https://www.kernel.org/doc/Documentation/filesystems/sharedsubtree.txt).   This only applies to bind-mounted volumes, not internal volumes   or named volumes. Mount propagation requires the source mount   point (the location where the source directory is mounted in the   host operating system) to have the correct propagation properties.   For shared volumes, the source mount point must be set to &#x60;shared&#x60;.   For slave volumes, the mount must be set to either &#x60;shared&#x60; or   &#x60;slave&#x60;.  | [optional] [default to undefined]
**ContainerIDFile** | **string** | Path to a file where the container ID is written | [optional] [default to undefined]
**LogConfig** | [**HostConfigAllOfLogConfig**](HostConfigAllOfLogConfig.md) |  | [optional] [default to undefined]
**NetworkMode** | **string** | Network mode to use for this container. Supported standard values are: &#x60;bridge&#x60;, &#x60;host&#x60;, &#x60;none&#x60;, and &#x60;container:&lt;name|id&gt;&#x60;. Any other value is taken as a custom network\&#39;s name to which this container should connect to.  | [optional] [default to undefined]
**PortBindings** | **{ [key: string]: Array&lt;PortBinding&gt; | null; }** | PortMap describes the mapping of container ports to host ports, using the container\&#39;s port-number and protocol as key in the format &#x60;&lt;port&gt;/&lt;protocol&gt;&#x60;, for example, &#x60;80/udp&#x60;.  If a container\&#39;s port is mapped for multiple protocols, separate entries are added to the mapping table.  | [optional] [default to undefined]
**RestartPolicy** | [**RestartPolicy**](RestartPolicy.md) |  | [optional] [default to undefined]
**AutoRemove** | **boolean** | Automatically remove the container when the container\&#39;s process exits. This has no effect if &#x60;RestartPolicy&#x60; is set.  | [optional] [default to undefined]
**VolumeDriver** | **string** | Driver that this container uses to mount volumes. | [optional] [default to undefined]
**VolumesFrom** | **Array&lt;string&gt;** | A list of volumes to inherit from another container, specified in the form &#x60;&lt;container name&gt;[:&lt;ro|rw&gt;]&#x60;.  | [optional] [default to undefined]
**Mounts** | [**Array&lt;Mount&gt;**](Mount.md) | Specification for mounts to be added to the container.  | [optional] [default to undefined]
**ConsoleSize** | **Array&lt;number&gt;** | Initial console size, as an &#x60;[height, width]&#x60; array.  | [optional] [default to undefined]
**Annotations** | **{ [key: string]: string; }** | Arbitrary non-identifying metadata attached to container and provided to the runtime when the container is started.  | [optional] [default to undefined]
**CapAdd** | **Array&lt;string&gt;** | A list of kernel capabilities to add to the container. Conflicts with option \&#39;Capabilities\&#39;.  | [optional] [default to undefined]
**CapDrop** | **Array&lt;string&gt;** | A list of kernel capabilities to drop from the container. Conflicts with option \&#39;Capabilities\&#39;.  | [optional] [default to undefined]
**CgroupnsMode** | **string** | cgroup namespace mode for the container. Possible values are:  - &#x60;\&quot;private\&quot;&#x60;: the container runs in its own private cgroup namespace - &#x60;\&quot;host\&quot;&#x60;: use the host system\&#39;s cgroup namespace  If not specified, the daemon default is used, which can either be &#x60;\&quot;private\&quot;&#x60; or &#x60;\&quot;host\&quot;&#x60;, depending on daemon version, kernel support and configuration.  | [optional] [default to undefined]
**Dns** | **Array&lt;string&gt;** | A list of DNS servers for the container to use. | [optional] [default to undefined]
**DnsOptions** | **Array&lt;string&gt;** | A list of DNS options. | [optional] [default to undefined]
**DnsSearch** | **Array&lt;string&gt;** | A list of DNS search domains. | [optional] [default to undefined]
**ExtraHosts** | **Array&lt;string&gt;** | A list of hostnames/IP mappings to add to the container\&#39;s &#x60;/etc/hosts&#x60; file. Specified in the form &#x60;[\&quot;hostname:IP\&quot;]&#x60;.  | [optional] [default to undefined]
**GroupAdd** | **Array&lt;string&gt;** | A list of additional groups that the container process will run as.  | [optional] [default to undefined]
**IpcMode** | **string** | IPC sharing mode for the container. Possible values are:  - &#x60;\&quot;none\&quot;&#x60;: own private IPC namespace, with /dev/shm not mounted - &#x60;\&quot;private\&quot;&#x60;: own private IPC namespace - &#x60;\&quot;shareable\&quot;&#x60;: own private IPC namespace, with a possibility to share it with other containers - &#x60;\&quot;container:&lt;name|id&gt;\&quot;&#x60;: join another (shareable) container\&#39;s IPC namespace - &#x60;\&quot;host\&quot;&#x60;: use the host system\&#39;s IPC namespace  If not specified, daemon default is used, which can either be &#x60;\&quot;private\&quot;&#x60; or &#x60;\&quot;shareable\&quot;&#x60;, depending on daemon version and configuration.  | [optional] [default to undefined]
**Cgroup** | **string** | Cgroup to use for the container. | [optional] [default to undefined]
**Links** | **Array&lt;string&gt;** | A list of links for the container in the form &#x60;container_name:alias&#x60;.  | [optional] [default to undefined]
**OomScoreAdj** | **number** | An integer value containing the score given to the container in order to tune OOM killer preferences.  | [optional] [default to undefined]
**PidMode** | **string** | Set the PID (Process) Namespace mode for the container. It can be either:  - &#x60;\&quot;container:&lt;name|id&gt;\&quot;&#x60;: joins another container\&#39;s PID namespace - &#x60;\&quot;host\&quot;&#x60;: use the host\&#39;s PID namespace inside the container  | [optional] [default to undefined]
**Privileged** | **boolean** | Gives the container full access to the host. | [optional] [default to undefined]
**PublishAllPorts** | **boolean** | Allocates an ephemeral host port for all of a container\&#39;s exposed ports.  Ports are de-allocated when the container stops and allocated when the container starts. The allocated port might be changed when restarting the container.  The port is selected from the ephemeral port range that depends on the kernel. For example, on Linux the range is defined by &#x60;/proc/sys/net/ipv4/ip_local_port_range&#x60;.  | [optional] [default to undefined]
**ReadonlyRootfs** | **boolean** | Mount the container\&#39;s root filesystem as read only. | [optional] [default to undefined]
**SecurityOpt** | **Array&lt;string&gt;** | A list of string values to customize labels for MLS systems, such as SELinux.  | [optional] [default to undefined]
**StorageOpt** | **{ [key: string]: string; }** | Storage driver options for this container, in the form &#x60;{\&quot;size\&quot;: \&quot;120G\&quot;}&#x60;.  | [optional] [default to undefined]
**Tmpfs** | **{ [key: string]: string; }** | A map of container directories which should be replaced by tmpfs mounts, and their corresponding mount options. For example:  &#x60;&#x60;&#x60; { \&quot;/run\&quot;: \&quot;rw,noexec,nosuid,size&#x3D;65536k\&quot; } &#x60;&#x60;&#x60;  | [optional] [default to undefined]
**UTSMode** | **string** | UTS namespace to use for the container. | [optional] [default to undefined]
**UsernsMode** | **string** | Sets the usernamespace mode for the container when usernamespace remapping option is enabled.  | [optional] [default to undefined]
**ShmSize** | **number** | Size of &#x60;/dev/shm&#x60; in bytes. If omitted, the system uses 64MB.  | [optional] [default to undefined]
**Sysctls** | **{ [key: string]: string; }** | A list of kernel parameters (sysctls) to set in the container.  This field is omitted if not set. | [optional] [default to undefined]
**Runtime** | **string** | Runtime to use with this container. | [optional] [default to undefined]
**Isolation** | **string** | Isolation technology of the container. (Windows only)  | [optional] [default to undefined]
**MaskedPaths** | **Array&lt;string&gt;** | The list of paths to be masked inside the container (this overrides the default set of paths).  | [optional] [default to undefined]
**ReadonlyPaths** | **Array&lt;string&gt;** | The list of paths to be set as read-only inside the container (this overrides the default set of paths).  | [optional] [default to undefined]

## Example

```typescript
import { HostConfig } from './api';

const instance: HostConfig = {
    CpuShares,
    Memory,
    CgroupParent,
    BlkioWeight,
    BlkioWeightDevice,
    BlkioDeviceReadBps,
    BlkioDeviceWriteBps,
    BlkioDeviceReadIOps,
    BlkioDeviceWriteIOps,
    CpuPeriod,
    CpuQuota,
    CpuRealtimePeriod,
    CpuRealtimeRuntime,
    CpusetCpus,
    CpusetMems,
    Devices,
    DeviceCgroupRules,
    DeviceRequests,
    KernelMemoryTCP,
    MemoryReservation,
    MemorySwap,
    MemorySwappiness,
    NanoCpus,
    OomKillDisable,
    Init,
    PidsLimit,
    Ulimits,
    CpuCount,
    CpuPercent,
    IOMaximumIOps,
    IOMaximumBandwidth,
    Binds,
    ContainerIDFile,
    LogConfig,
    NetworkMode,
    PortBindings,
    RestartPolicy,
    AutoRemove,
    VolumeDriver,
    VolumesFrom,
    Mounts,
    ConsoleSize,
    Annotations,
    CapAdd,
    CapDrop,
    CgroupnsMode,
    Dns,
    DnsOptions,
    DnsSearch,
    ExtraHosts,
    GroupAdd,
    IpcMode,
    Cgroup,
    Links,
    OomScoreAdj,
    PidMode,
    Privileged,
    PublishAllPorts,
    ReadonlyRootfs,
    SecurityOpt,
    StorageOpt,
    Tmpfs,
    UTSMode,
    UsernsMode,
    ShmSize,
    Sysctls,
    Runtime,
    Isolation,
    MaskedPaths,
    ReadonlyPaths,
};
```

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
