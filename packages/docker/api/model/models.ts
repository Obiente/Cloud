import localVarRequest from 'request';

export * from './address';
export * from './authConfig';
export * from './buildCache';
export * from './buildInfo';
export * from './buildPruneResponse';
export * from './changeType';
export * from './clusterInfo';
export * from './clusterVolume';
export * from './clusterVolumeInfo';
export * from './clusterVolumePublishStatusInner';
export * from './clusterVolumeSpec';
export * from './clusterVolumeSpecAccessMode';
export * from './clusterVolumeSpecAccessModeAccessibilityRequirements';
export * from './clusterVolumeSpecAccessModeCapacityRange';
export * from './clusterVolumeSpecAccessModeSecretsInner';
export * from './commit';
export * from './config';
export * from './configCreateRequest';
export * from './configReference';
export * from './configSpec';
export * from './containerBlkioStatEntry';
export * from './containerBlkioStats';
export * from './containerCPUStats';
export * from './containerCPUUsage';
export * from './containerConfig';
export * from './containerCreateRequest';
export * from './containerCreateResponse';
export * from './containerInspectResponse';
export * from './containerMemoryStats';
export * from './containerNetworkStats';
export * from './containerPidsStats';
export * from './containerPruneResponse';
export * from './containerState';
export * from './containerStatsResponse';
export * from './containerStatus';
export * from './containerStorageStats';
export * from './containerSummary';
export * from './containerSummaryHostConfig';
export * from './containerSummaryNetworkSettings';
export * from './containerThrottlingData';
export * from './containerTopResponse';
export * from './containerUpdateRequest';
export * from './containerUpdateResponse';
export * from './containerWaitExitError';
export * from './containerWaitResponse';
export * from './containerdInfo';
export * from './containerdInfoNamespaces';
export * from './createImageInfo';
export * from './deviceInfo';
export * from './deviceMapping';
export * from './deviceRequest';
export * from './distributionInspect';
export * from './driver';
export * from './driverData';
export * from './endpointIPAMConfig';
export * from './endpointPortConfig';
export * from './endpointSettings';
export * from './endpointSpec';
export * from './engineDescription';
export * from './engineDescriptionPluginsInner';
export * from './errorDetail';
export * from './errorResponse';
export * from './eventActor';
export * from './eventMessage';
export * from './execConfig';
export * from './execInspectResponse';
export * from './execStartConfig';
export * from './filesystemChange';
export * from './firewallInfo';
export * from './genericResourcesInner';
export * from './genericResourcesInnerDiscreteResourceSpec';
export * from './genericResourcesInnerNamedResourceSpec';
export * from './health';
export * from './healthConfig';
export * from './healthcheckResult';
export * from './historyResponseItem';
export * from './hostConfig';
export * from './hostConfigAllOfLogConfig';
export * from './iDResponse';
export * from './iPAM';
export * from './iPAMConfig';
export * from './imageConfig';
export * from './imageDeleteResponseItem';
export * from './imageID';
export * from './imageInspect';
export * from './imageInspectMetadata';
export * from './imageInspectRootFS';
export * from './imageManifestSummary';
export * from './imageManifestSummaryAttestationData';
export * from './imageManifestSummaryImageData';
export * from './imageManifestSummaryImageDataSize';
export * from './imageManifestSummarySize';
export * from './imagePruneResponse';
export * from './imageSearchResponseItem';
export * from './imageSummary';
export * from './indexInfo';
export * from './joinTokens';
export * from './limit';
export * from './localNodeState';
export * from './managerStatus';
export * from './mount';
export * from './mountBindOptions';
export * from './mountImageOptions';
export * from './mountPoint';
export * from './mountTmpfsOptions';
export * from './mountVolumeOptions';
export * from './mountVolumeOptionsDriverConfig';
export * from './network';
export * from './networkAttachmentConfig';
export * from './networkConnectRequest';
export * from './networkContainer';
export * from './networkCreateRequest';
export * from './networkCreateResponse';
export * from './networkDisconnectRequest';
export * from './networkPruneResponse';
export * from './networkSettings';
export * from './networkingConfig';
export * from './node';
export * from './nodeDescription';
export * from './nodeSpec';
export * from './nodeState';
export * from './nodeStatus';
export * from './oCIDescriptor';
export * from './oCIPlatform';
export * from './objectVersion';
export * from './peerInfo';
export * from './peerNode';
export * from './platform';
export * from './plugin';
export * from './pluginConfig';
export * from './pluginConfigArgs';
export * from './pluginConfigInterface';
export * from './pluginConfigLinux';
export * from './pluginConfigNetwork';
export * from './pluginConfigRootfs';
export * from './pluginConfigUser';
export * from './pluginDevice';
export * from './pluginEnv';
export * from './pluginInterfaceType';
export * from './pluginMount';
export * from './pluginPrivilege';
export * from './pluginSettings';
export * from './pluginsInfo';
export * from './port';
export * from './portBinding';
export * from './portStatus';
export * from './processConfig';
export * from './progressDetail';
export * from './pushImageInfo';
export * from './reachability';
export * from './registryServiceConfig';
export * from './resourceObject';
export * from './resources';
export * from './resourcesBlkioWeightDeviceInner';
export * from './resourcesUlimitsInner';
export * from './restartPolicy';
export * from './runtime';
export * from './secret';
export * from './secretCreateRequest';
export * from './secretSpec';
export * from './service';
export * from './serviceCreateRequest';
export * from './serviceCreateResponse';
export * from './serviceEndpoint';
export * from './serviceEndpointVirtualIPsInner';
export * from './serviceJobStatus';
export * from './serviceServiceStatus';
export * from './serviceSpec';
export * from './serviceSpecMode';
export * from './serviceSpecModeReplicated';
export * from './serviceSpecModeReplicatedJob';
export * from './serviceSpecRollbackConfig';
export * from './serviceSpecUpdateConfig';
export * from './serviceUpdateRequest';
export * from './serviceUpdateResponse';
export * from './serviceUpdateStatus';
export * from './swarm';
export * from './swarmInfo';
export * from './swarmInitRequest';
export * from './swarmJoinRequest';
export * from './swarmSpec';
export * from './swarmSpecCAConfig';
export * from './swarmSpecCAConfigExternalCAsInner';
export * from './swarmSpecDispatcher';
export * from './swarmSpecEncryptionConfig';
export * from './swarmSpecOrchestration';
export * from './swarmSpecRaft';
export * from './swarmSpecTaskDefaults';
export * from './swarmSpecTaskDefaultsLogDriver';
export * from './swarmUnlockRequest';
export * from './systemAuthResponse';
export * from './systemDataUsageResponse';
export * from './systemInfo';
export * from './systemInfoDefaultAddressPoolsInner';
export * from './systemVersion';
export * from './systemVersionComponentsInner';
export * from './systemVersionPlatform';
export * from './tLSInfo';
export * from './task';
export * from './taskSpec';
export * from './taskSpecContainerSpec';
export * from './taskSpecContainerSpecConfigsInner';
export * from './taskSpecContainerSpecConfigsInnerFile';
export * from './taskSpecContainerSpecDNSConfig';
export * from './taskSpecContainerSpecPrivileges';
export * from './taskSpecContainerSpecPrivilegesAppArmor';
export * from './taskSpecContainerSpecPrivilegesCredentialSpec';
export * from './taskSpecContainerSpecPrivilegesSELinuxContext';
export * from './taskSpecContainerSpecPrivilegesSeccomp';
export * from './taskSpecContainerSpecSecretsInner';
export * from './taskSpecContainerSpecSecretsInnerFile';
export * from './taskSpecLogDriver';
export * from './taskSpecNetworkAttachmentSpec';
export * from './taskSpecPlacement';
export * from './taskSpecPlacementPreferencesInner';
export * from './taskSpecPlacementPreferencesInnerSpread';
export * from './taskSpecPluginSpec';
export * from './taskSpecResources';
export * from './taskSpecRestartPolicy';
export * from './taskState';
export * from './taskStatus';
export * from './throttleDevice';
export * from './unlockKeyResponse';
export * from './volume';
export * from './volumeCreateOptions';
export * from './volumeListResponse';
export * from './volumePruneResponse';
export * from './volumeUpdateRequest';
export * from './volumeUsageData';

import * as fs from 'fs';

export interface RequestDetailedFile {
    value: Buffer;
    options?: {
        filename?: string;
        contentType?: string;
    }
}

export type RequestFile = string | Buffer | fs.ReadStream | RequestDetailedFile;


import { Address } from './address';
import { AuthConfig } from './authConfig';
import { BuildCache } from './buildCache';
import { BuildInfo } from './buildInfo';
import { BuildPruneResponse } from './buildPruneResponse';
import { ChangeType } from './changeType';
import { ClusterInfo } from './clusterInfo';
import { ClusterVolume } from './clusterVolume';
import { ClusterVolumeInfo } from './clusterVolumeInfo';
import { ClusterVolumePublishStatusInner } from './clusterVolumePublishStatusInner';
import { ClusterVolumeSpec } from './clusterVolumeSpec';
import { ClusterVolumeSpecAccessMode } from './clusterVolumeSpecAccessMode';
import { ClusterVolumeSpecAccessModeAccessibilityRequirements } from './clusterVolumeSpecAccessModeAccessibilityRequirements';
import { ClusterVolumeSpecAccessModeCapacityRange } from './clusterVolumeSpecAccessModeCapacityRange';
import { ClusterVolumeSpecAccessModeSecretsInner } from './clusterVolumeSpecAccessModeSecretsInner';
import { Commit } from './commit';
import { Config } from './config';
import { ConfigCreateRequest } from './configCreateRequest';
import { ConfigReference } from './configReference';
import { ConfigSpec } from './configSpec';
import { ContainerBlkioStatEntry } from './containerBlkioStatEntry';
import { ContainerBlkioStats } from './containerBlkioStats';
import { ContainerCPUStats } from './containerCPUStats';
import { ContainerCPUUsage } from './containerCPUUsage';
import { ContainerConfig } from './containerConfig';
import { ContainerCreateRequest } from './containerCreateRequest';
import { ContainerCreateResponse } from './containerCreateResponse';
import { ContainerInspectResponse } from './containerInspectResponse';
import { ContainerMemoryStats } from './containerMemoryStats';
import { ContainerNetworkStats } from './containerNetworkStats';
import { ContainerPidsStats } from './containerPidsStats';
import { ContainerPruneResponse } from './containerPruneResponse';
import { ContainerState } from './containerState';
import { ContainerStatsResponse } from './containerStatsResponse';
import { ContainerStatus } from './containerStatus';
import { ContainerStorageStats } from './containerStorageStats';
import { ContainerSummary } from './containerSummary';
import { ContainerSummaryHostConfig } from './containerSummaryHostConfig';
import { ContainerSummaryNetworkSettings } from './containerSummaryNetworkSettings';
import { ContainerThrottlingData } from './containerThrottlingData';
import { ContainerTopResponse } from './containerTopResponse';
import { ContainerUpdateRequest } from './containerUpdateRequest';
import { ContainerUpdateResponse } from './containerUpdateResponse';
import { ContainerWaitExitError } from './containerWaitExitError';
import { ContainerWaitResponse } from './containerWaitResponse';
import { ContainerdInfo } from './containerdInfo';
import { ContainerdInfoNamespaces } from './containerdInfoNamespaces';
import { CreateImageInfo } from './createImageInfo';
import { DeviceInfo } from './deviceInfo';
import { DeviceMapping } from './deviceMapping';
import { DeviceRequest } from './deviceRequest';
import { DistributionInspect } from './distributionInspect';
import { Driver } from './driver';
import { DriverData } from './driverData';
import { EndpointIPAMConfig } from './endpointIPAMConfig';
import { EndpointPortConfig } from './endpointPortConfig';
import { EndpointSettings } from './endpointSettings';
import { EndpointSpec } from './endpointSpec';
import { EngineDescription } from './engineDescription';
import { EngineDescriptionPluginsInner } from './engineDescriptionPluginsInner';
import { ErrorDetail } from './errorDetail';
import { ErrorResponse } from './errorResponse';
import { EventActor } from './eventActor';
import { EventMessage } from './eventMessage';
import { ExecConfig } from './execConfig';
import { ExecInspectResponse } from './execInspectResponse';
import { ExecStartConfig } from './execStartConfig';
import { FilesystemChange } from './filesystemChange';
import { FirewallInfo } from './firewallInfo';
import { GenericResourcesInner } from './genericResourcesInner';
import { GenericResourcesInnerDiscreteResourceSpec } from './genericResourcesInnerDiscreteResourceSpec';
import { GenericResourcesInnerNamedResourceSpec } from './genericResourcesInnerNamedResourceSpec';
import { Health } from './health';
import { HealthConfig } from './healthConfig';
import { HealthcheckResult } from './healthcheckResult';
import { HistoryResponseItem } from './historyResponseItem';
import { HostConfig } from './hostConfig';
import { HostConfigAllOfLogConfig } from './hostConfigAllOfLogConfig';
import { IDResponse } from './iDResponse';
import { IPAM } from './iPAM';
import { IPAMConfig } from './iPAMConfig';
import { ImageConfig } from './imageConfig';
import { ImageDeleteResponseItem } from './imageDeleteResponseItem';
import { ImageID } from './imageID';
import { ImageInspect } from './imageInspect';
import { ImageInspectMetadata } from './imageInspectMetadata';
import { ImageInspectRootFS } from './imageInspectRootFS';
import { ImageManifestSummary } from './imageManifestSummary';
import { ImageManifestSummaryAttestationData } from './imageManifestSummaryAttestationData';
import { ImageManifestSummaryImageData } from './imageManifestSummaryImageData';
import { ImageManifestSummaryImageDataSize } from './imageManifestSummaryImageDataSize';
import { ImageManifestSummarySize } from './imageManifestSummarySize';
import { ImagePruneResponse } from './imagePruneResponse';
import { ImageSearchResponseItem } from './imageSearchResponseItem';
import { ImageSummary } from './imageSummary';
import { IndexInfo } from './indexInfo';
import { JoinTokens } from './joinTokens';
import { Limit } from './limit';
import { LocalNodeState } from './localNodeState';
import { ManagerStatus } from './managerStatus';
import { Mount } from './mount';
import { MountBindOptions } from './mountBindOptions';
import { MountImageOptions } from './mountImageOptions';
import { MountPoint } from './mountPoint';
import { MountTmpfsOptions } from './mountTmpfsOptions';
import { MountVolumeOptions } from './mountVolumeOptions';
import { MountVolumeOptionsDriverConfig } from './mountVolumeOptionsDriverConfig';
import { Network } from './network';
import { NetworkAttachmentConfig } from './networkAttachmentConfig';
import { NetworkConnectRequest } from './networkConnectRequest';
import { NetworkContainer } from './networkContainer';
import { NetworkCreateRequest } from './networkCreateRequest';
import { NetworkCreateResponse } from './networkCreateResponse';
import { NetworkDisconnectRequest } from './networkDisconnectRequest';
import { NetworkPruneResponse } from './networkPruneResponse';
import { NetworkSettings } from './networkSettings';
import { NetworkingConfig } from './networkingConfig';
import { Node } from './node';
import { NodeDescription } from './nodeDescription';
import { NodeSpec } from './nodeSpec';
import { NodeState } from './nodeState';
import { NodeStatus } from './nodeStatus';
import { OCIDescriptor } from './oCIDescriptor';
import { OCIPlatform } from './oCIPlatform';
import { ObjectVersion } from './objectVersion';
import { PeerInfo } from './peerInfo';
import { PeerNode } from './peerNode';
import { Platform } from './platform';
import { Plugin } from './plugin';
import { PluginConfig } from './pluginConfig';
import { PluginConfigArgs } from './pluginConfigArgs';
import { PluginConfigInterface } from './pluginConfigInterface';
import { PluginConfigLinux } from './pluginConfigLinux';
import { PluginConfigNetwork } from './pluginConfigNetwork';
import { PluginConfigRootfs } from './pluginConfigRootfs';
import { PluginConfigUser } from './pluginConfigUser';
import { PluginDevice } from './pluginDevice';
import { PluginEnv } from './pluginEnv';
import { PluginInterfaceType } from './pluginInterfaceType';
import { PluginMount } from './pluginMount';
import { PluginPrivilege } from './pluginPrivilege';
import { PluginSettings } from './pluginSettings';
import { PluginsInfo } from './pluginsInfo';
import { Port } from './port';
import { PortBinding } from './portBinding';
import { PortStatus } from './portStatus';
import { ProcessConfig } from './processConfig';
import { ProgressDetail } from './progressDetail';
import { PushImageInfo } from './pushImageInfo';
import { Reachability } from './reachability';
import { RegistryServiceConfig } from './registryServiceConfig';
import { ResourceObject } from './resourceObject';
import { Resources } from './resources';
import { ResourcesBlkioWeightDeviceInner } from './resourcesBlkioWeightDeviceInner';
import { ResourcesUlimitsInner } from './resourcesUlimitsInner';
import { RestartPolicy } from './restartPolicy';
import { Runtime } from './runtime';
import { Secret } from './secret';
import { SecretCreateRequest } from './secretCreateRequest';
import { SecretSpec } from './secretSpec';
import { Service } from './service';
import { ServiceCreateRequest } from './serviceCreateRequest';
import { ServiceCreateResponse } from './serviceCreateResponse';
import { ServiceEndpoint } from './serviceEndpoint';
import { ServiceEndpointVirtualIPsInner } from './serviceEndpointVirtualIPsInner';
import { ServiceJobStatus } from './serviceJobStatus';
import { ServiceServiceStatus } from './serviceServiceStatus';
import { ServiceSpec } from './serviceSpec';
import { ServiceSpecMode } from './serviceSpecMode';
import { ServiceSpecModeReplicated } from './serviceSpecModeReplicated';
import { ServiceSpecModeReplicatedJob } from './serviceSpecModeReplicatedJob';
import { ServiceSpecRollbackConfig } from './serviceSpecRollbackConfig';
import { ServiceSpecUpdateConfig } from './serviceSpecUpdateConfig';
import { ServiceUpdateRequest } from './serviceUpdateRequest';
import { ServiceUpdateResponse } from './serviceUpdateResponse';
import { ServiceUpdateStatus } from './serviceUpdateStatus';
import { Swarm } from './swarm';
import { SwarmInfo } from './swarmInfo';
import { SwarmInitRequest } from './swarmInitRequest';
import { SwarmJoinRequest } from './swarmJoinRequest';
import { SwarmSpec } from './swarmSpec';
import { SwarmSpecCAConfig } from './swarmSpecCAConfig';
import { SwarmSpecCAConfigExternalCAsInner } from './swarmSpecCAConfigExternalCAsInner';
import { SwarmSpecDispatcher } from './swarmSpecDispatcher';
import { SwarmSpecEncryptionConfig } from './swarmSpecEncryptionConfig';
import { SwarmSpecOrchestration } from './swarmSpecOrchestration';
import { SwarmSpecRaft } from './swarmSpecRaft';
import { SwarmSpecTaskDefaults } from './swarmSpecTaskDefaults';
import { SwarmSpecTaskDefaultsLogDriver } from './swarmSpecTaskDefaultsLogDriver';
import { SwarmUnlockRequest } from './swarmUnlockRequest';
import { SystemAuthResponse } from './systemAuthResponse';
import { SystemDataUsageResponse } from './systemDataUsageResponse';
import { SystemInfo } from './systemInfo';
import { SystemInfoDefaultAddressPoolsInner } from './systemInfoDefaultAddressPoolsInner';
import { SystemVersion } from './systemVersion';
import { SystemVersionComponentsInner } from './systemVersionComponentsInner';
import { SystemVersionPlatform } from './systemVersionPlatform';
import { TLSInfo } from './tLSInfo';
import { Task } from './task';
import { TaskSpec } from './taskSpec';
import { TaskSpecContainerSpec } from './taskSpecContainerSpec';
import { TaskSpecContainerSpecConfigsInner } from './taskSpecContainerSpecConfigsInner';
import { TaskSpecContainerSpecConfigsInnerFile } from './taskSpecContainerSpecConfigsInnerFile';
import { TaskSpecContainerSpecDNSConfig } from './taskSpecContainerSpecDNSConfig';
import { TaskSpecContainerSpecPrivileges } from './taskSpecContainerSpecPrivileges';
import { TaskSpecContainerSpecPrivilegesAppArmor } from './taskSpecContainerSpecPrivilegesAppArmor';
import { TaskSpecContainerSpecPrivilegesCredentialSpec } from './taskSpecContainerSpecPrivilegesCredentialSpec';
import { TaskSpecContainerSpecPrivilegesSELinuxContext } from './taskSpecContainerSpecPrivilegesSELinuxContext';
import { TaskSpecContainerSpecPrivilegesSeccomp } from './taskSpecContainerSpecPrivilegesSeccomp';
import { TaskSpecContainerSpecSecretsInner } from './taskSpecContainerSpecSecretsInner';
import { TaskSpecContainerSpecSecretsInnerFile } from './taskSpecContainerSpecSecretsInnerFile';
import { TaskSpecLogDriver } from './taskSpecLogDriver';
import { TaskSpecNetworkAttachmentSpec } from './taskSpecNetworkAttachmentSpec';
import { TaskSpecPlacement } from './taskSpecPlacement';
import { TaskSpecPlacementPreferencesInner } from './taskSpecPlacementPreferencesInner';
import { TaskSpecPlacementPreferencesInnerSpread } from './taskSpecPlacementPreferencesInnerSpread';
import { TaskSpecPluginSpec } from './taskSpecPluginSpec';
import { TaskSpecResources } from './taskSpecResources';
import { TaskSpecRestartPolicy } from './taskSpecRestartPolicy';
import { TaskState } from './taskState';
import { TaskStatus } from './taskStatus';
import { ThrottleDevice } from './throttleDevice';
import { UnlockKeyResponse } from './unlockKeyResponse';
import { Volume } from './volume';
import { VolumeCreateOptions } from './volumeCreateOptions';
import { VolumeListResponse } from './volumeListResponse';
import { VolumePruneResponse } from './volumePruneResponse';
import { VolumeUpdateRequest } from './volumeUpdateRequest';
import { VolumeUsageData } from './volumeUsageData';

/* tslint:disable:no-unused-variable */
let primitives = [
                    "string",
                    "boolean",
                    "double",
                    "integer",
                    "long",
                    "float",
                    "number",
                    "any"
                 ];

let enumsMap: {[index: string]: any} = {
        "BuildCache.TypeEnum": BuildCache.TypeEnum,
        "ChangeType": ChangeType,
        "ClusterVolumePublishStatusInner.StateEnum": ClusterVolumePublishStatusInner.StateEnum,
        "ClusterVolumeSpecAccessMode.ScopeEnum": ClusterVolumeSpecAccessMode.ScopeEnum,
        "ClusterVolumeSpecAccessMode.SharingEnum": ClusterVolumeSpecAccessMode.SharingEnum,
        "ClusterVolumeSpecAccessMode.AvailabilityEnum": ClusterVolumeSpecAccessMode.AvailabilityEnum,
        "ContainerState.StatusEnum": ContainerState.StatusEnum,
        "ContainerSummary.StateEnum": ContainerSummary.StateEnum,
        "EndpointPortConfig.ProtocolEnum": EndpointPortConfig.ProtocolEnum,
        "EndpointPortConfig.PublishModeEnum": EndpointPortConfig.PublishModeEnum,
        "EndpointSpec.ModeEnum": EndpointSpec.ModeEnum,
        "EventMessage.TypeEnum": EventMessage.TypeEnum,
        "EventMessage.ScopeEnum": EventMessage.ScopeEnum,
        "Health.StatusEnum": Health.StatusEnum,
        "HostConfig.CgroupnsModeEnum": HostConfig.CgroupnsModeEnum,
        "HostConfig.IsolationEnum": HostConfig.IsolationEnum,
        "HostConfigAllOfLogConfig.TypeEnum": HostConfigAllOfLogConfig.TypeEnum,
        "ImageManifestSummary.KindEnum": ImageManifestSummary.KindEnum,
        "LocalNodeState": LocalNodeState,
        "Mount.TypeEnum": Mount.TypeEnum,
        "MountBindOptions.PropagationEnum": MountBindOptions.PropagationEnum,
        "MountPoint.TypeEnum": MountPoint.TypeEnum,
        "NodeSpec.RoleEnum": NodeSpec.RoleEnum,
        "NodeSpec.AvailabilityEnum": NodeSpec.AvailabilityEnum,
        "NodeState": NodeState,
        "PluginConfigInterface.ProtocolSchemeEnum": PluginConfigInterface.ProtocolSchemeEnum,
        "Port.TypeEnum": Port.TypeEnum,
        "Reachability": Reachability,
        "RestartPolicy.NameEnum": RestartPolicy.NameEnum,
        "ServiceSpecRollbackConfig.FailureActionEnum": ServiceSpecRollbackConfig.FailureActionEnum,
        "ServiceSpecRollbackConfig.OrderEnum": ServiceSpecRollbackConfig.OrderEnum,
        "ServiceSpecUpdateConfig.FailureActionEnum": ServiceSpecUpdateConfig.FailureActionEnum,
        "ServiceSpecUpdateConfig.OrderEnum": ServiceSpecUpdateConfig.OrderEnum,
        "ServiceUpdateStatus.StateEnum": ServiceUpdateStatus.StateEnum,
        "SwarmSpecCAConfigExternalCAsInner.ProtocolEnum": SwarmSpecCAConfigExternalCAsInner.ProtocolEnum,
        "SystemInfo.CgroupDriverEnum": SystemInfo.CgroupDriverEnum,
        "SystemInfo.CgroupVersionEnum": SystemInfo.CgroupVersionEnum,
        "SystemInfo.IsolationEnum": SystemInfo.IsolationEnum,
        "TaskSpecContainerSpec.IsolationEnum": TaskSpecContainerSpec.IsolationEnum,
        "TaskSpecContainerSpecPrivilegesAppArmor.ModeEnum": TaskSpecContainerSpecPrivilegesAppArmor.ModeEnum,
        "TaskSpecContainerSpecPrivilegesSeccomp.ModeEnum": TaskSpecContainerSpecPrivilegesSeccomp.ModeEnum,
        "TaskSpecRestartPolicy.ConditionEnum": TaskSpecRestartPolicy.ConditionEnum,
        "TaskState": TaskState,
        "Volume.ScopeEnum": Volume.ScopeEnum,
}

let typeMap: {[index: string]: any} = {
    "Address": Address,
    "AuthConfig": AuthConfig,
    "BuildCache": BuildCache,
    "BuildInfo": BuildInfo,
    "BuildPruneResponse": BuildPruneResponse,
    "ClusterInfo": ClusterInfo,
    "ClusterVolume": ClusterVolume,
    "ClusterVolumeInfo": ClusterVolumeInfo,
    "ClusterVolumePublishStatusInner": ClusterVolumePublishStatusInner,
    "ClusterVolumeSpec": ClusterVolumeSpec,
    "ClusterVolumeSpecAccessMode": ClusterVolumeSpecAccessMode,
    "ClusterVolumeSpecAccessModeAccessibilityRequirements": ClusterVolumeSpecAccessModeAccessibilityRequirements,
    "ClusterVolumeSpecAccessModeCapacityRange": ClusterVolumeSpecAccessModeCapacityRange,
    "ClusterVolumeSpecAccessModeSecretsInner": ClusterVolumeSpecAccessModeSecretsInner,
    "Commit": Commit,
    "Config": Config,
    "ConfigCreateRequest": ConfigCreateRequest,
    "ConfigReference": ConfigReference,
    "ConfigSpec": ConfigSpec,
    "ContainerBlkioStatEntry": ContainerBlkioStatEntry,
    "ContainerBlkioStats": ContainerBlkioStats,
    "ContainerCPUStats": ContainerCPUStats,
    "ContainerCPUUsage": ContainerCPUUsage,
    "ContainerConfig": ContainerConfig,
    "ContainerCreateRequest": ContainerCreateRequest,
    "ContainerCreateResponse": ContainerCreateResponse,
    "ContainerInspectResponse": ContainerInspectResponse,
    "ContainerMemoryStats": ContainerMemoryStats,
    "ContainerNetworkStats": ContainerNetworkStats,
    "ContainerPidsStats": ContainerPidsStats,
    "ContainerPruneResponse": ContainerPruneResponse,
    "ContainerState": ContainerState,
    "ContainerStatsResponse": ContainerStatsResponse,
    "ContainerStatus": ContainerStatus,
    "ContainerStorageStats": ContainerStorageStats,
    "ContainerSummary": ContainerSummary,
    "ContainerSummaryHostConfig": ContainerSummaryHostConfig,
    "ContainerSummaryNetworkSettings": ContainerSummaryNetworkSettings,
    "ContainerThrottlingData": ContainerThrottlingData,
    "ContainerTopResponse": ContainerTopResponse,
    "ContainerUpdateRequest": ContainerUpdateRequest,
    "ContainerUpdateResponse": ContainerUpdateResponse,
    "ContainerWaitExitError": ContainerWaitExitError,
    "ContainerWaitResponse": ContainerWaitResponse,
    "ContainerdInfo": ContainerdInfo,
    "ContainerdInfoNamespaces": ContainerdInfoNamespaces,
    "CreateImageInfo": CreateImageInfo,
    "DeviceInfo": DeviceInfo,
    "DeviceMapping": DeviceMapping,
    "DeviceRequest": DeviceRequest,
    "DistributionInspect": DistributionInspect,
    "Driver": Driver,
    "DriverData": DriverData,
    "EndpointIPAMConfig": EndpointIPAMConfig,
    "EndpointPortConfig": EndpointPortConfig,
    "EndpointSettings": EndpointSettings,
    "EndpointSpec": EndpointSpec,
    "EngineDescription": EngineDescription,
    "EngineDescriptionPluginsInner": EngineDescriptionPluginsInner,
    "ErrorDetail": ErrorDetail,
    "ErrorResponse": ErrorResponse,
    "EventActor": EventActor,
    "EventMessage": EventMessage,
    "ExecConfig": ExecConfig,
    "ExecInspectResponse": ExecInspectResponse,
    "ExecStartConfig": ExecStartConfig,
    "FilesystemChange": FilesystemChange,
    "FirewallInfo": FirewallInfo,
    "GenericResourcesInner": GenericResourcesInner,
    "GenericResourcesInnerDiscreteResourceSpec": GenericResourcesInnerDiscreteResourceSpec,
    "GenericResourcesInnerNamedResourceSpec": GenericResourcesInnerNamedResourceSpec,
    "Health": Health,
    "HealthConfig": HealthConfig,
    "HealthcheckResult": HealthcheckResult,
    "HistoryResponseItem": HistoryResponseItem,
    "HostConfig": HostConfig,
    "HostConfigAllOfLogConfig": HostConfigAllOfLogConfig,
    "IDResponse": IDResponse,
    "IPAM": IPAM,
    "IPAMConfig": IPAMConfig,
    "ImageConfig": ImageConfig,
    "ImageDeleteResponseItem": ImageDeleteResponseItem,
    "ImageID": ImageID,
    "ImageInspect": ImageInspect,
    "ImageInspectMetadata": ImageInspectMetadata,
    "ImageInspectRootFS": ImageInspectRootFS,
    "ImageManifestSummary": ImageManifestSummary,
    "ImageManifestSummaryAttestationData": ImageManifestSummaryAttestationData,
    "ImageManifestSummaryImageData": ImageManifestSummaryImageData,
    "ImageManifestSummaryImageDataSize": ImageManifestSummaryImageDataSize,
    "ImageManifestSummarySize": ImageManifestSummarySize,
    "ImagePruneResponse": ImagePruneResponse,
    "ImageSearchResponseItem": ImageSearchResponseItem,
    "ImageSummary": ImageSummary,
    "IndexInfo": IndexInfo,
    "JoinTokens": JoinTokens,
    "Limit": Limit,
    "ManagerStatus": ManagerStatus,
    "Mount": Mount,
    "MountBindOptions": MountBindOptions,
    "MountImageOptions": MountImageOptions,
    "MountPoint": MountPoint,
    "MountTmpfsOptions": MountTmpfsOptions,
    "MountVolumeOptions": MountVolumeOptions,
    "MountVolumeOptionsDriverConfig": MountVolumeOptionsDriverConfig,
    "Network": Network,
    "NetworkAttachmentConfig": NetworkAttachmentConfig,
    "NetworkConnectRequest": NetworkConnectRequest,
    "NetworkContainer": NetworkContainer,
    "NetworkCreateRequest": NetworkCreateRequest,
    "NetworkCreateResponse": NetworkCreateResponse,
    "NetworkDisconnectRequest": NetworkDisconnectRequest,
    "NetworkPruneResponse": NetworkPruneResponse,
    "NetworkSettings": NetworkSettings,
    "NetworkingConfig": NetworkingConfig,
    "Node": Node,
    "NodeDescription": NodeDescription,
    "NodeSpec": NodeSpec,
    "NodeStatus": NodeStatus,
    "OCIDescriptor": OCIDescriptor,
    "OCIPlatform": OCIPlatform,
    "ObjectVersion": ObjectVersion,
    "PeerInfo": PeerInfo,
    "PeerNode": PeerNode,
    "Platform": Platform,
    "Plugin": Plugin,
    "PluginConfig": PluginConfig,
    "PluginConfigArgs": PluginConfigArgs,
    "PluginConfigInterface": PluginConfigInterface,
    "PluginConfigLinux": PluginConfigLinux,
    "PluginConfigNetwork": PluginConfigNetwork,
    "PluginConfigRootfs": PluginConfigRootfs,
    "PluginConfigUser": PluginConfigUser,
    "PluginDevice": PluginDevice,
    "PluginEnv": PluginEnv,
    "PluginInterfaceType": PluginInterfaceType,
    "PluginMount": PluginMount,
    "PluginPrivilege": PluginPrivilege,
    "PluginSettings": PluginSettings,
    "PluginsInfo": PluginsInfo,
    "Port": Port,
    "PortBinding": PortBinding,
    "PortStatus": PortStatus,
    "ProcessConfig": ProcessConfig,
    "ProgressDetail": ProgressDetail,
    "PushImageInfo": PushImageInfo,
    "RegistryServiceConfig": RegistryServiceConfig,
    "ResourceObject": ResourceObject,
    "Resources": Resources,
    "ResourcesBlkioWeightDeviceInner": ResourcesBlkioWeightDeviceInner,
    "ResourcesUlimitsInner": ResourcesUlimitsInner,
    "RestartPolicy": RestartPolicy,
    "Runtime": Runtime,
    "Secret": Secret,
    "SecretCreateRequest": SecretCreateRequest,
    "SecretSpec": SecretSpec,
    "Service": Service,
    "ServiceCreateRequest": ServiceCreateRequest,
    "ServiceCreateResponse": ServiceCreateResponse,
    "ServiceEndpoint": ServiceEndpoint,
    "ServiceEndpointVirtualIPsInner": ServiceEndpointVirtualIPsInner,
    "ServiceJobStatus": ServiceJobStatus,
    "ServiceServiceStatus": ServiceServiceStatus,
    "ServiceSpec": ServiceSpec,
    "ServiceSpecMode": ServiceSpecMode,
    "ServiceSpecModeReplicated": ServiceSpecModeReplicated,
    "ServiceSpecModeReplicatedJob": ServiceSpecModeReplicatedJob,
    "ServiceSpecRollbackConfig": ServiceSpecRollbackConfig,
    "ServiceSpecUpdateConfig": ServiceSpecUpdateConfig,
    "ServiceUpdateRequest": ServiceUpdateRequest,
    "ServiceUpdateResponse": ServiceUpdateResponse,
    "ServiceUpdateStatus": ServiceUpdateStatus,
    "Swarm": Swarm,
    "SwarmInfo": SwarmInfo,
    "SwarmInitRequest": SwarmInitRequest,
    "SwarmJoinRequest": SwarmJoinRequest,
    "SwarmSpec": SwarmSpec,
    "SwarmSpecCAConfig": SwarmSpecCAConfig,
    "SwarmSpecCAConfigExternalCAsInner": SwarmSpecCAConfigExternalCAsInner,
    "SwarmSpecDispatcher": SwarmSpecDispatcher,
    "SwarmSpecEncryptionConfig": SwarmSpecEncryptionConfig,
    "SwarmSpecOrchestration": SwarmSpecOrchestration,
    "SwarmSpecRaft": SwarmSpecRaft,
    "SwarmSpecTaskDefaults": SwarmSpecTaskDefaults,
    "SwarmSpecTaskDefaultsLogDriver": SwarmSpecTaskDefaultsLogDriver,
    "SwarmUnlockRequest": SwarmUnlockRequest,
    "SystemAuthResponse": SystemAuthResponse,
    "SystemDataUsageResponse": SystemDataUsageResponse,
    "SystemInfo": SystemInfo,
    "SystemInfoDefaultAddressPoolsInner": SystemInfoDefaultAddressPoolsInner,
    "SystemVersion": SystemVersion,
    "SystemVersionComponentsInner": SystemVersionComponentsInner,
    "SystemVersionPlatform": SystemVersionPlatform,
    "TLSInfo": TLSInfo,
    "Task": Task,
    "TaskSpec": TaskSpec,
    "TaskSpecContainerSpec": TaskSpecContainerSpec,
    "TaskSpecContainerSpecConfigsInner": TaskSpecContainerSpecConfigsInner,
    "TaskSpecContainerSpecConfigsInnerFile": TaskSpecContainerSpecConfigsInnerFile,
    "TaskSpecContainerSpecDNSConfig": TaskSpecContainerSpecDNSConfig,
    "TaskSpecContainerSpecPrivileges": TaskSpecContainerSpecPrivileges,
    "TaskSpecContainerSpecPrivilegesAppArmor": TaskSpecContainerSpecPrivilegesAppArmor,
    "TaskSpecContainerSpecPrivilegesCredentialSpec": TaskSpecContainerSpecPrivilegesCredentialSpec,
    "TaskSpecContainerSpecPrivilegesSELinuxContext": TaskSpecContainerSpecPrivilegesSELinuxContext,
    "TaskSpecContainerSpecPrivilegesSeccomp": TaskSpecContainerSpecPrivilegesSeccomp,
    "TaskSpecContainerSpecSecretsInner": TaskSpecContainerSpecSecretsInner,
    "TaskSpecContainerSpecSecretsInnerFile": TaskSpecContainerSpecSecretsInnerFile,
    "TaskSpecLogDriver": TaskSpecLogDriver,
    "TaskSpecNetworkAttachmentSpec": TaskSpecNetworkAttachmentSpec,
    "TaskSpecPlacement": TaskSpecPlacement,
    "TaskSpecPlacementPreferencesInner": TaskSpecPlacementPreferencesInner,
    "TaskSpecPlacementPreferencesInnerSpread": TaskSpecPlacementPreferencesInnerSpread,
    "TaskSpecPluginSpec": TaskSpecPluginSpec,
    "TaskSpecResources": TaskSpecResources,
    "TaskSpecRestartPolicy": TaskSpecRestartPolicy,
    "TaskStatus": TaskStatus,
    "ThrottleDevice": ThrottleDevice,
    "UnlockKeyResponse": UnlockKeyResponse,
    "Volume": Volume,
    "VolumeCreateOptions": VolumeCreateOptions,
    "VolumeListResponse": VolumeListResponse,
    "VolumePruneResponse": VolumePruneResponse,
    "VolumeUpdateRequest": VolumeUpdateRequest,
    "VolumeUsageData": VolumeUsageData,
}

// Check if a string starts with another string without using es6 features
function startsWith(str: string, match: string): boolean {
    return str.substring(0, match.length) === match;
}

// Check if a string ends with another string without using es6 features
function endsWith(str: string, match: string): boolean {
    return str.length >= match.length && str.substring(str.length - match.length) === match;
}

const nullableSuffix = " | null";
const optionalSuffix = " | undefined";
const arrayPrefix = "Array<";
const arraySuffix = ">";
const mapPrefix = "{ [key: string]: ";
const mapSuffix = "; }";

export class ObjectSerializer {
    public static findCorrectType(data: any, expectedType: string) {
        if (data == undefined) {
            return expectedType;
        } else if (primitives.indexOf(expectedType.toLowerCase()) !== -1) {
            return expectedType;
        } else if (expectedType === "Date") {
            return expectedType;
        } else {
            if (enumsMap[expectedType]) {
                return expectedType;
            }

            if (!typeMap[expectedType]) {
                return expectedType; // w/e we don't know the type
            }

            // Check the discriminator
            let discriminatorProperty = typeMap[expectedType].discriminator;
            if (discriminatorProperty == null) {
                return expectedType; // the type does not have a discriminator. use it.
            } else {
                if (data[discriminatorProperty]) {
                    var discriminatorType = data[discriminatorProperty];
                    if(typeMap[discriminatorType]){
                        return discriminatorType; // use the type given in the discriminator
                    } else {
                        return expectedType; // discriminator did not map to a type
                    }
                } else {
                    return expectedType; // discriminator was not present (or an empty string)
                }
            }
        }
    }

    public static serialize(data: any, type: string): any {
        if (data == undefined) {
            return data;
        } else if (primitives.indexOf(type.toLowerCase()) !== -1) {
            return data;
        } else if (endsWith(type, nullableSuffix)) {
            let subType: string = type.slice(0, -nullableSuffix.length); // Type | null => Type
            return ObjectSerializer.serialize(data, subType);
        } else if (endsWith(type, optionalSuffix)) {
            let subType: string = type.slice(0, -optionalSuffix.length); // Type | undefined => Type
            return ObjectSerializer.serialize(data, subType);
        } else if (startsWith(type, arrayPrefix)) {
            let subType: string = type.slice(arrayPrefix.length, -arraySuffix.length); // Array<Type> => Type
            let transformedData: any[] = [];
            for (let index = 0; index < data.length; index++) {
                let datum = data[index];
                transformedData.push(ObjectSerializer.serialize(datum, subType));
            }
            return transformedData;
        } else if (startsWith(type, mapPrefix)) {
            let subType: string = type.slice(mapPrefix.length, -mapSuffix.length); // { [key: string]: Type; } => Type
            let transformedData: { [key: string]: any } = {};
            for (let key in data) {
                transformedData[key] = ObjectSerializer.serialize(
                    data[key],
                    subType,
                );
            }
            return transformedData;
        } else if (type === "Date") {
            return data.toISOString();
        } else {
            if (enumsMap[type]) {
                return data;
            }
            if (!typeMap[type]) { // in case we dont know the type
                return data;
            }

            // Get the actual type of this object
            type = this.findCorrectType(data, type);

            // get the map for the correct type.
            let attributeTypes = typeMap[type].getAttributeTypeMap();
            let instance: {[index: string]: any} = {};
            for (let index = 0; index < attributeTypes.length; index++) {
                let attributeType = attributeTypes[index];
                instance[attributeType.baseName] = ObjectSerializer.serialize(data[attributeType.name], attributeType.type);
            }
            return instance;
        }
    }

    public static deserialize(data: any, type: string): any {
        // polymorphism may change the actual type.
        type = ObjectSerializer.findCorrectType(data, type);
        if (data == undefined) {
            return data;
        } else if (primitives.indexOf(type.toLowerCase()) !== -1) {
            return data;
        } else if (endsWith(type, nullableSuffix)) {
            let subType: string = type.slice(0, -nullableSuffix.length); // Type | null => Type
            return ObjectSerializer.deserialize(data, subType);
        } else if (endsWith(type, optionalSuffix)) {
            let subType: string = type.slice(0, -optionalSuffix.length); // Type | undefined => Type
            return ObjectSerializer.deserialize(data, subType);
        } else if (startsWith(type, arrayPrefix)) {
            let subType: string = type.slice(arrayPrefix.length, -arraySuffix.length); // Array<Type> => Type
            let transformedData: any[] = [];
            for (let index = 0; index < data.length; index++) {
                let datum = data[index];
                transformedData.push(ObjectSerializer.deserialize(datum, subType));
            }
            return transformedData;
        } else if (startsWith(type, mapPrefix)) {
            let subType: string = type.slice(mapPrefix.length, -mapSuffix.length); // { [key: string]: Type; } => Type
            let transformedData: { [key: string]: any } = {};
            for (let key in data) {
                transformedData[key] = ObjectSerializer.deserialize(
                    data[key],
                    subType,
                );
            }
            return transformedData;
        } else if (type === "Date") {
            return new Date(data);
        } else {
            if (enumsMap[type]) {// is Enum
                return data;
            }

            if (!typeMap[type]) { // dont know the type
                return data;
            }
            let instance = new typeMap[type]();
            let attributeTypes = typeMap[type].getAttributeTypeMap();
            for (let index = 0; index < attributeTypes.length; index++) {
                let attributeType = attributeTypes[index];
                instance[attributeType.name] = ObjectSerializer.deserialize(data[attributeType.baseName], attributeType.type);
            }
            return instance;
        }
    }
}

export interface Authentication {
    /**
    * Apply authentication settings to header and query params.
    */
    applyToRequest(requestOptions: localVarRequest.Options): Promise<void> | void;
}

export class HttpBasicAuth implements Authentication {
    public username: string = '';
    public password: string = '';

    applyToRequest(requestOptions: localVarRequest.Options): void {
        requestOptions.auth = {
            username: this.username, password: this.password
        }
    }
}

export class HttpBearerAuth implements Authentication {
    public accessToken: string | (() => string) = '';

    applyToRequest(requestOptions: localVarRequest.Options): void {
        if (requestOptions && requestOptions.headers) {
            const accessToken = typeof this.accessToken === 'function'
                            ? this.accessToken()
                            : this.accessToken;
            requestOptions.headers["Authorization"] = "Bearer " + accessToken;
        }
    }
}

export class ApiKeyAuth implements Authentication {
    public apiKey: string = '';

    constructor(private location: string, private paramName: string) {
    }

    applyToRequest(requestOptions: localVarRequest.Options): void {
        if (this.location == "query") {
            (<any>requestOptions.qs)[this.paramName] = this.apiKey;
        } else if (this.location == "header" && requestOptions && requestOptions.headers) {
            requestOptions.headers[this.paramName] = this.apiKey;
        } else if (this.location == 'cookie' && requestOptions && requestOptions.headers) {
            if (requestOptions.headers['Cookie']) {
                requestOptions.headers['Cookie'] += '; ' + this.paramName + '=' + encodeURIComponent(this.apiKey);
            }
            else {
                requestOptions.headers['Cookie'] = this.paramName + '=' + encodeURIComponent(this.apiKey);
            }
        }
    }
}

export class OAuth implements Authentication {
    public accessToken: string = '';

    applyToRequest(requestOptions: localVarRequest.Options): void {
        if (requestOptions && requestOptions.headers) {
            requestOptions.headers["Authorization"] = "Bearer " + this.accessToken;
        }
    }
}

export class VoidAuth implements Authentication {
    public username: string = '';
    public password: string = '';

    applyToRequest(_: localVarRequest.Options): void {
        // Do nothing
    }
}

export type Interceptor = (requestOptions: localVarRequest.Options) => (Promise<void> | void);
