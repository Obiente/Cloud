import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Pagination, VPSSize } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/vps/v1/vps_service.proto.
 */
export declare const file_obiente_cloud_vps_v1_vps_service: GenFile;
/**
 * @generated from message obiente.cloud.vps.v1.ListVPSRequest
 */
export type ListVPSRequest = Message<"obiente.cloud.vps.v1.ListVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: int32 page = 2;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 3;
     */
    perPage: number;
    /**
     * Optional: filter by status
     *
     * @generated from field: optional obiente.cloud.vps.v1.VPSStatus status = 4;
     */
    status?: VPSStatus;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSRequest.
 * Use `create(ListVPSRequestSchema)` to create a new message.
 */
export declare const ListVPSRequestSchema: GenMessage<ListVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ListVPSResponse
 */
export type ListVPSResponse = Message<"obiente.cloud.vps.v1.ListVPSResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSInstance vps_instances = 1;
     */
    vpsInstances: VPSInstance[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSResponse.
 * Use `create(ListVPSResponseSchema)` to create a new message.
 */
export declare const ListVPSResponseSchema: GenMessage<ListVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.CreateVPSRequest
 */
export type CreateVPSRequest = Message<"obiente.cloud.vps.v1.CreateVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: optional string description = 3;
     */
    description?: string;
    /**
     * Obiente Cloud region (e.g., "us-east-1", "eu-west-1")
     *
     * @generated from field: string region = 4;
     */
    region: string;
    /**
     * OS image
     *
     * @generated from field: obiente.cloud.vps.v1.VPSImage image = 5;
     */
    image: VPSImage;
    /**
     * Custom image ID (if image is CUSTOM)
     *
     * @generated from field: optional string image_id = 6;
     */
    imageId?: string;
    /**
     * VPS size/spec (e.g., "small", "medium", "large", or specific like "2cpu-4gb")
     *
     * @generated from field: string size = 7;
     */
    size: string;
    /**
     * SSH key ID for initial access (deprecated: use users instead)
     *
     * @generated from field: optional string ssh_key_id = 8;
     */
    sshKeyId?: string;
    /**
     * Additional metadata/tags
     *
     * @generated from field: map<string, string> metadata = 9;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * Cloud-init configuration
     *
     * @generated from field: optional obiente.cloud.vps.v1.CloudInitConfig cloud_init = 10;
     */
    cloudInit?: CloudInitConfig;
    /**
     * Root password configuration (if not set, password will be auto-generated)
     *
     * Custom root password (optional, auto-generated if not provided)
     *
     * @generated from field: optional string root_password = 11;
     */
    rootPassword?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSRequest.
 * Use `create(CreateVPSRequestSchema)` to create a new message.
 */
export declare const CreateVPSRequestSchema: GenMessage<CreateVPSRequest>;
/**
 * CloudInitConfig contains cloud-init configuration options
 *
 * @generated from message obiente.cloud.vps.v1.CloudInitConfig
 */
export type CloudInitConfig = Message<"obiente.cloud.vps.v1.CloudInitConfig"> & {
    /**
     * User configurations
     *
     * @generated from field: repeated obiente.cloud.vps.v1.CloudInitUser users = 1;
     */
    users: CloudInitUser[];
    /**
     * System configuration
     *
     * System hostname
     *
     * @generated from field: optional string hostname = 2;
     */
    hostname?: string;
    /**
     * Timezone (e.g., "America/New_York", "UTC")
     *
     * @generated from field: optional string timezone = 3;
     */
    timezone?: string;
    /**
     * Locale (e.g., "en_US.UTF-8")
     *
     * @generated from field: optional string locale = 4;
     */
    locale?: string;
    /**
     * Package management
     *
     * Additional packages to install
     *
     * @generated from field: repeated string packages = 5;
     */
    packages: string[];
    /**
     * Update package database (default: true)
     *
     * @generated from field: optional bool package_update = 6;
     */
    packageUpdate?: boolean;
    /**
     * Upgrade packages (default: false)
     *
     * @generated from field: optional bool package_upgrade = 7;
     */
    packageUpgrade?: boolean;
    /**
     * Custom commands to run on first boot
     *
     * Commands to run (executed in order)
     *
     * @generated from field: repeated string runcmd = 8;
     */
    runcmd: string[];
    /**
     * Write files
     *
     * @generated from field: repeated obiente.cloud.vps.v1.CloudInitWriteFile write_files = 9;
     */
    writeFiles: CloudInitWriteFile[];
    /**
     * SSH configuration
     *
     * Install SSH server (default: true)
     *
     * @generated from field: optional bool ssh_install_server = 10;
     */
    sshInstallServer?: boolean;
    /**
     * Allow password authentication (default: true)
     *
     * @generated from field: optional bool ssh_allow_pw = 11;
     */
    sshAllowPw?: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.CloudInitConfig.
 * Use `create(CloudInitConfigSchema)` to create a new message.
 */
export declare const CloudInitConfigSchema: GenMessage<CloudInitConfig>;
/**
 * CloudInitUser represents a user to be created via cloud-init
 *
 * @generated from message obiente.cloud.vps.v1.CloudInitUser
 */
export type CloudInitUser = Message<"obiente.cloud.vps.v1.CloudInitUser"> & {
    /**
     * Username
     *
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * Password (if not set, user can only login via SSH keys)
     *
     * @generated from field: optional string password = 2;
     */
    password?: string;
    /**
     * SSH public keys for this user
     *
     * @generated from field: repeated string ssh_authorized_keys = 3;
     */
    sshAuthorizedKeys: string[];
    /**
     * Grant sudo access (default: false)
     *
     * @generated from field: optional bool sudo = 4;
     */
    sudo?: boolean;
    /**
     * Sudo without password (default: false)
     *
     * @generated from field: optional bool sudo_nopasswd = 5;
     */
    sudoNopasswd?: boolean;
    /**
     * Additional groups to add user to
     *
     * @generated from field: repeated string groups = 6;
     */
    groups: string[];
    /**
     * Shell (default: /bin/bash)
     *
     * @generated from field: optional string shell = 7;
     */
    shell?: string;
    /**
     * Lock password (default: false)
     *
     * @generated from field: optional bool lock_passwd = 8;
     */
    lockPasswd?: boolean;
    /**
     * Full name/comment
     *
     * @generated from field: optional string gecos = 9;
     */
    gecos?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.CloudInitUser.
 * Use `create(CloudInitUserSchema)` to create a new message.
 */
export declare const CloudInitUserSchema: GenMessage<CloudInitUser>;
/**
 * CloudInitWriteFile represents a file to be written via cloud-init
 *
 * @generated from message obiente.cloud.vps.v1.CloudInitWriteFile
 */
export type CloudInitWriteFile = Message<"obiente.cloud.vps.v1.CloudInitWriteFile"> & {
    /**
     * File path
     *
     * @generated from field: string path = 1;
     */
    path: string;
    /**
     * File content
     *
     * @generated from field: string content = 2;
     */
    content: string;
    /**
     * File owner (default: root:root)
     *
     * @generated from field: optional string owner = 3;
     */
    owner?: string;
    /**
     * File permissions (default: "0644")
     *
     * @generated from field: optional string permissions = 4;
     */
    permissions?: string;
    /**
     * Append to existing file (default: false)
     *
     * @generated from field: optional bool append = 5;
     */
    append?: boolean;
    /**
     * Defer writing until after package installation
     *
     * @generated from field: optional bool defer = 6;
     */
    defer?: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.CloudInitWriteFile.
 * Use `create(CloudInitWriteFileSchema)` to create a new message.
 */
export declare const CloudInitWriteFileSchema: GenMessage<CloudInitWriteFile>;
/**
 * @generated from message obiente.cloud.vps.v1.CreateVPSResponse
 */
export type CreateVPSResponse = Message<"obiente.cloud.vps.v1.CreateVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSResponse.
 * Use `create(CreateVPSResponseSchema)` to create a new message.
 */
export declare const CreateVPSResponseSchema: GenMessage<CreateVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSRequest
 */
export type GetVPSRequest = Message<"obiente.cloud.vps.v1.GetVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSRequest.
 * Use `create(GetVPSRequestSchema)` to create a new message.
 */
export declare const GetVPSRequestSchema: GenMessage<GetVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSResponse
 */
export type GetVPSResponse = Message<"obiente.cloud.vps.v1.GetVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSResponse.
 * Use `create(GetVPSResponseSchema)` to create a new message.
 */
export declare const GetVPSResponseSchema: GenMessage<GetVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateVPSRequest
 */
export type UpdateVPSRequest = Message<"obiente.cloud.vps.v1.UpdateVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: optional string name = 3;
     */
    name?: string;
    /**
     * @generated from field: optional string description = 4;
     */
    description?: string;
    /**
     * Update metadata/tags
     *
     * @generated from field: map<string, string> metadata = 5;
     */
    metadata: {
        [key: string]: string;
    };
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSRequest.
 * Use `create(UpdateVPSRequestSchema)` to create a new message.
 */
export declare const UpdateVPSRequestSchema: GenMessage<UpdateVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateVPSResponse
 */
export type UpdateVPSResponse = Message<"obiente.cloud.vps.v1.UpdateVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSResponse.
 * Use `create(UpdateVPSResponseSchema)` to create a new message.
 */
export declare const UpdateVPSResponseSchema: GenMessage<UpdateVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.DeleteVPSRequest
 */
export type DeleteVPSRequest = Message<"obiente.cloud.vps.v1.DeleteVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * If true, force delete even if VPS is running
     *
     * @generated from field: bool force = 3;
     */
    force: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSRequest.
 * Use `create(DeleteVPSRequestSchema)` to create a new message.
 */
export declare const DeleteVPSRequestSchema: GenMessage<DeleteVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.DeleteVPSResponse
 */
export type DeleteVPSResponse = Message<"obiente.cloud.vps.v1.DeleteVPSResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSResponse.
 * Use `create(DeleteVPSResponseSchema)` to create a new message.
 */
export declare const DeleteVPSResponseSchema: GenMessage<DeleteVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.StartVPSRequest
 */
export type StartVPSRequest = Message<"obiente.cloud.vps.v1.StartVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.StartVPSRequest.
 * Use `create(StartVPSRequestSchema)` to create a new message.
 */
export declare const StartVPSRequestSchema: GenMessage<StartVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.StartVPSResponse
 */
export type StartVPSResponse = Message<"obiente.cloud.vps.v1.StartVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.StartVPSResponse.
 * Use `create(StartVPSResponseSchema)` to create a new message.
 */
export declare const StartVPSResponseSchema: GenMessage<StartVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.StopVPSRequest
 */
export type StopVPSRequest = Message<"obiente.cloud.vps.v1.StopVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.StopVPSRequest.
 * Use `create(StopVPSRequestSchema)` to create a new message.
 */
export declare const StopVPSRequestSchema: GenMessage<StopVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.StopVPSResponse
 */
export type StopVPSResponse = Message<"obiente.cloud.vps.v1.StopVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.StopVPSResponse.
 * Use `create(StopVPSResponseSchema)` to create a new message.
 */
export declare const StopVPSResponseSchema: GenMessage<StopVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.RebootVPSRequest
 */
export type RebootVPSRequest = Message<"obiente.cloud.vps.v1.RebootVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RebootVPSRequest.
 * Use `create(RebootVPSRequestSchema)` to create a new message.
 */
export declare const RebootVPSRequestSchema: GenMessage<RebootVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.RebootVPSResponse
 */
export type RebootVPSResponse = Message<"obiente.cloud.vps.v1.RebootVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
};
/**
 * Describes the message obiente.cloud.vps.v1.RebootVPSResponse.
 * Use `create(RebootVPSResponseSchema)` to create a new message.
 */
export declare const RebootVPSResponseSchema: GenMessage<RebootVPSResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.StreamVPSStatusRequest
 */
export type StreamVPSStatusRequest = Message<"obiente.cloud.vps.v1.StreamVPSStatusRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.StreamVPSStatusRequest.
 * Use `create(StreamVPSStatusRequestSchema)` to create a new message.
 */
export declare const StreamVPSStatusRequestSchema: GenMessage<StreamVPSStatusRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSStatusUpdate
 */
export type VPSStatusUpdate = Message<"obiente.cloud.vps.v1.VPSStatusUpdate"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSStatus status = 2;
     */
    status: VPSStatus;
    /**
     * @generated from field: optional string message = 3;
     */
    message?: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 4;
     */
    timestamp?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSStatusUpdate.
 * Use `create(VPSStatusUpdateSchema)` to create a new message.
 */
export declare const VPSStatusUpdateSchema: GenMessage<VPSStatusUpdate>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSMetricsRequest
 */
export type GetVPSMetricsRequest = Message<"obiente.cloud.vps.v1.GetVPSMetricsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: google.protobuf.Timestamp start_time = 3;
     */
    startTime?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp end_time = 4;
     */
    endTime?: Timestamp;
    /**
     * Optional: aggregation interval (e.g., "1m", "5m", "1h")
     *
     * @generated from field: optional string interval = 5;
     */
    interval?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSMetricsRequest.
 * Use `create(GetVPSMetricsRequestSchema)` to create a new message.
 */
export declare const GetVPSMetricsRequestSchema: GenMessage<GetVPSMetricsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSMetricsResponse
 */
export type GetVPSMetricsResponse = Message<"obiente.cloud.vps.v1.GetVPSMetricsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSMetric metrics = 1;
     */
    metrics: VPSMetric[];
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSMetricsResponse.
 * Use `create(GetVPSMetricsResponseSchema)` to create a new message.
 */
export declare const GetVPSMetricsResponseSchema: GenMessage<GetVPSMetricsResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.StreamVPSMetricsRequest
 */
export type StreamVPSMetricsRequest = Message<"obiente.cloud.vps.v1.StreamVPSMetricsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.StreamVPSMetricsRequest.
 * Use `create(StreamVPSMetricsRequestSchema)` to create a new message.
 */
export declare const StreamVPSMetricsRequestSchema: GenMessage<StreamVPSMetricsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSMetric
 */
export type VPSMetric = Message<"obiente.cloud.vps.v1.VPSMetric"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 2;
     */
    timestamp?: Timestamp;
    /**
     * CPU usage percentage (0-100)
     *
     * @generated from field: double cpu_usage_percent = 3;
     */
    cpuUsagePercent: number;
    /**
     * Memory used in bytes
     *
     * @generated from field: int64 memory_used_bytes = 4;
     */
    memoryUsedBytes: bigint;
    /**
     * Total memory in bytes
     *
     * @generated from field: int64 memory_total_bytes = 5;
     */
    memoryTotalBytes: bigint;
    /**
     * Disk used in bytes
     *
     * @generated from field: int64 disk_used_bytes = 6;
     */
    diskUsedBytes: bigint;
    /**
     * Total disk in bytes
     *
     * @generated from field: int64 disk_total_bytes = 7;
     */
    diskTotalBytes: bigint;
    /**
     * Network received bytes
     *
     * @generated from field: int64 network_rx_bytes = 8;
     */
    networkRxBytes: bigint;
    /**
     * Network transmitted bytes
     *
     * @generated from field: int64 network_tx_bytes = 9;
     */
    networkTxBytes: bigint;
    /**
     * Disk read IOPS
     *
     * @generated from field: double disk_read_iops = 10;
     */
    diskReadIops: number;
    /**
     * Disk write IOPS
     *
     * @generated from field: double disk_write_iops = 11;
     */
    diskWriteIops: number;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSMetric.
 * Use `create(VPSMetricSchema)` to create a new message.
 */
export declare const VPSMetricSchema: GenMessage<VPSMetric>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSUsageRequest
 */
export type GetVPSUsageRequest = Message<"obiente.cloud.vps.v1.GetVPSUsageRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * Optional: specify a month (YYYY-MM format). Defaults to current month.
     *
     * @generated from field: optional string month = 3;
     */
    month?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSUsageRequest.
 * Use `create(GetVPSUsageRequestSchema)` to create a new message.
 */
export declare const GetVPSUsageRequestSchema: GenMessage<GetVPSUsageRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.FindVPSByLeaseRequest
 */
export type FindVPSByLeaseRequest = Message<"obiente.cloud.vps.v1.FindVPSByLeaseRequest"> & {
    /**
     * IP address to match (optional)
     *
     * @generated from field: string ip = 1;
     */
    ip: string;
    /**
     * MAC address to match (optional)
     *
     * @generated from field: string mac = 2;
     */
    mac: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.FindVPSByLeaseRequest.
 * Use `create(FindVPSByLeaseRequestSchema)` to create a new message.
 */
export declare const FindVPSByLeaseRequestSchema: GenMessage<FindVPSByLeaseRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.FindVPSByLeaseResponse
 */
export type FindVPSByLeaseResponse = Message<"obiente.cloud.vps.v1.FindVPSByLeaseResponse"> & {
    /**
     * VPS ID if found, empty otherwise
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * Organization ID owning the VPS (if any)
     *
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.FindVPSByLeaseResponse.
 * Use `create(FindVPSByLeaseResponseSchema)` to create a new message.
 */
export declare const FindVPSByLeaseResponseSchema: GenMessage<FindVPSByLeaseResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSUsageResponse
 */
export type GetVPSUsageResponse = Message<"obiente.cloud.vps.v1.GetVPSUsageResponse"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * YYYY-MM format
     *
     * @generated from field: string month = 2;
     */
    month: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSUsageMetrics current = 3;
     */
    current?: VPSUsageMetrics;
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSUsageMetrics estimated_monthly = 4;
     */
    estimatedMonthly?: VPSUsageMetrics;
    /**
     * Estimated cost in cents (e.g., 2347 = $23.47)
     *
     * @generated from field: int64 estimated_cost_cents = 5;
     */
    estimatedCostCents: bigint;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSUsageResponse.
 * Use `create(GetVPSUsageResponseSchema)` to create a new message.
 */
export declare const GetVPSUsageResponseSchema: GenMessage<GetVPSUsageResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSUsageMetrics
 */
export type VPSUsageMetrics = Message<"obiente.cloud.vps.v1.VPSUsageMetrics"> & {
    /**
     * @generated from field: int64 cpu_core_seconds = 1;
     */
    cpuCoreSeconds: bigint;
    /**
     * @generated from field: int64 memory_byte_seconds = 2;
     */
    memoryByteSeconds: bigint;
    /**
     * @generated from field: int64 bandwidth_rx_bytes = 3;
     */
    bandwidthRxBytes: bigint;
    /**
     * @generated from field: int64 bandwidth_tx_bytes = 4;
     */
    bandwidthTxBytes: bigint;
    /**
     * @generated from field: int64 disk_bytes = 5;
     */
    diskBytes: bigint;
    /**
     * Total uptime in seconds for the period
     *
     * @generated from field: int64 uptime_seconds = 6;
     */
    uptimeSeconds: bigint;
    /**
     * Estimated cost in cents
     *
     * @generated from field: int64 estimated_cost_cents = 7;
     */
    estimatedCostCents: bigint;
    /**
     * Per-resource cost breakdown in cents
     *
     * @generated from field: optional int64 cpu_cost_cents = 8;
     */
    cpuCostCents?: bigint;
    /**
     * @generated from field: optional int64 memory_cost_cents = 9;
     */
    memoryCostCents?: bigint;
    /**
     * @generated from field: optional int64 bandwidth_cost_cents = 10;
     */
    bandwidthCostCents?: bigint;
    /**
     * @generated from field: optional int64 storage_cost_cents = 11;
     */
    storageCostCents?: bigint;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSUsageMetrics.
 * Use `create(VPSUsageMetricsSchema)` to create a new message.
 */
export declare const VPSUsageMetricsSchema: GenMessage<VPSUsageMetrics>;
/**
 * @generated from message obiente.cloud.vps.v1.ListAvailableVPSSizesRequest
 */
export type ListAvailableVPSSizesRequest = Message<"obiente.cloud.vps.v1.ListAvailableVPSSizesRequest"> & {
    /**
     * Optional: filter by region
     *
     * @generated from field: optional string region = 1;
     */
    region?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListAvailableVPSSizesRequest.
 * Use `create(ListAvailableVPSSizesRequestSchema)` to create a new message.
 */
export declare const ListAvailableVPSSizesRequestSchema: GenMessage<ListAvailableVPSSizesRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ListAvailableVPSSizesResponse
 */
export type ListAvailableVPSSizesResponse = Message<"obiente.cloud.vps.v1.ListAvailableVPSSizesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.common.v1.VPSSize sizes = 1;
     */
    sizes: VPSSize[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ListAvailableVPSSizesResponse.
 * Use `create(ListAvailableVPSSizesResponseSchema)` to create a new message.
 */
export declare const ListAvailableVPSSizesResponseSchema: GenMessage<ListAvailableVPSSizesResponse>;
/**
 * Empty - returns all available Obiente Cloud regions
 *
 * @generated from message obiente.cloud.vps.v1.ListVPSRegionsRequest
 */
export type ListVPSRegionsRequest = Message<"obiente.cloud.vps.v1.ListVPSRegionsRequest"> & {};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSRegionsRequest.
 * Use `create(ListVPSRegionsRequestSchema)` to create a new message.
 */
export declare const ListVPSRegionsRequestSchema: GenMessage<ListVPSRegionsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ListVPSRegionsResponse
 */
export type ListVPSRegionsResponse = Message<"obiente.cloud.vps.v1.ListVPSRegionsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSRegion regions = 1;
     */
    regions: VPSRegion[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSRegionsResponse.
 * Use `create(ListVPSRegionsResponseSchema)` to create a new message.
 */
export declare const ListVPSRegionsResponseSchema: GenMessage<ListVPSRegionsResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSProxyInfoRequest
 */
export type GetVPSProxyInfoRequest = Message<"obiente.cloud.vps.v1.GetVPSProxyInfoRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSProxyInfoRequest.
 * Use `create(GetVPSProxyInfoRequestSchema)` to create a new message.
 */
export declare const GetVPSProxyInfoRequestSchema: GenMessage<GetVPSProxyInfoRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSProxyInfoResponse
 */
export type GetVPSProxyInfoResponse = Message<"obiente.cloud.vps.v1.GetVPSProxyInfoResponse"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * WebSocket URL for terminal access (e.g., wss://obiente.cloud/vps/{vps_id}/terminal)
     *
     * @generated from field: string terminal_ws_url = 2;
     */
    terminalWsUrl: string;
    /**
     * SSH proxy connection string (e.g., ssh -J proxy@obiente.cloud -p 2222 user@vps-{vps_id})
     *
     * @generated from field: string ssh_proxy_command = 3;
     */
    sshProxyCommand: string;
    /**
     * SSH port for direct connection (if available)
     *
     * @generated from field: optional int32 ssh_port = 4;
     */
    sshPort?: number;
    /**
     * Instructions for connecting via proxy
     *
     * @generated from field: string connection_instructions = 5;
     */
    connectionInstructions: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSProxyInfoResponse.
 * Use `create(GetVPSProxyInfoResponseSchema)` to create a new message.
 */
export declare const GetVPSProxyInfoResponseSchema: GenMessage<GetVPSProxyInfoResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSRegion
 */
export type VPSRegion = Message<"obiente.cloud.vps.v1.VPSRegion"> & {
    /**
     * Region ID (e.g., "nbg1", "nyc1")
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Human-readable name
     *
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * Country code (e.g., "DE", "US")
     *
     * @generated from field: string country = 3;
     */
    country: string;
    /**
     * City name
     *
     * @generated from field: string city = 4;
     */
    city: string;
    /**
     * Whether this region is available
     *
     * @generated from field: bool available = 5;
     */
    available: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSRegion.
 * Use `create(VPSRegionSchema)` to create a new message.
 */
export declare const VPSRegionSchema: GenMessage<VPSRegion>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSInstance
 */
export type VPSInstance = Message<"obiente.cloud.vps.v1.VPSInstance"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: optional string description = 3;
     */
    description?: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSStatus status = 4;
     */
    status: VPSStatus;
    /**
     * @generated from field: string region = 5;
     */
    region: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSImage image = 6;
     */
    image: VPSImage;
    /**
     * @generated from field: optional string image_id = 7;
     */
    imageId?: string;
    /**
     * @generated from field: string size = 8;
     */
    size: string;
    /**
     * Resource specifications
     *
     * @generated from field: int32 cpu_cores = 9;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 10;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int64 disk_bytes = 11;
     */
    diskBytes: bigint;
    /**
     * Network information
     *
     * @generated from field: repeated string ipv4_addresses = 12;
     */
    ipv4Addresses: string[];
    /**
     * @generated from field: repeated string ipv6_addresses = 13;
     */
    ipv6Addresses: string[];
    /**
     * Infrastructure information
     *
     * Internal instance ID
     *
     * @generated from field: optional string instance_id = 14;
     */
    instanceId?: string;
    /**
     * Docker Swarm node ID where VPS is running
     *
     * @generated from field: optional string node_id = 15;
     */
    nodeId?: string;
    /**
     * SSH access
     *
     * @generated from field: optional string ssh_key_id = 16;
     */
    sshKeyId?: string;
    /**
     * Only returned on creation, never stored
     *
     * @generated from field: optional string root_password = 17;
     */
    rootPassword?: string;
    /**
     * Metadata and tags
     *
     * @generated from field: map<string, string> metadata = 18;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * Timestamps
     *
     * @generated from field: google.protobuf.Timestamp created_at = 19;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 20;
     */
    updatedAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp last_started_at = 21;
     */
    lastStartedAt?: Timestamp;
    /**
     * Soft delete
     *
     * @generated from field: optional google.protobuf.Timestamp deleted_at = 22;
     */
    deletedAt?: Timestamp;
    /**
     * Organization and ownership
     *
     * @generated from field: string organization_id = 23;
     */
    organizationId: string;
    /**
     * @generated from field: string created_by = 24;
     */
    createdBy: string;
    /**
     * Current resource usage (if available)
     *
     * @generated from field: optional obiente.cloud.vps.v1.VPSMetric current_metrics = 25;
     */
    currentMetrics?: VPSMetric;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSInstance.
 * Use `create(VPSInstanceSchema)` to create a new message.
 */
export declare const VPSInstanceSchema: GenMessage<VPSInstance>;
/**
 * Firewall rule management messages
 *
 * @generated from message obiente.cloud.vps.v1.ListFirewallRulesRequest
 */
export type ListFirewallRulesRequest = Message<"obiente.cloud.vps.v1.ListFirewallRulesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListFirewallRulesRequest.
 * Use `create(ListFirewallRulesRequestSchema)` to create a new message.
 */
export declare const ListFirewallRulesRequestSchema: GenMessage<ListFirewallRulesRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ListFirewallRulesResponse
 */
export type ListFirewallRulesResponse = Message<"obiente.cloud.vps.v1.ListFirewallRulesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.FirewallRule rules = 1;
     */
    rules: FirewallRule[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ListFirewallRulesResponse.
 * Use `create(ListFirewallRulesResponseSchema)` to create a new message.
 */
export declare const ListFirewallRulesResponseSchema: GenMessage<ListFirewallRulesResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.GetFirewallRuleRequest
 */
export type GetFirewallRuleRequest = Message<"obiente.cloud.vps.v1.GetFirewallRuleRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * Rule position (0-based index)
     *
     * @generated from field: int32 rule_pos = 3;
     */
    rulePos: number;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetFirewallRuleRequest.
 * Use `create(GetFirewallRuleRequestSchema)` to create a new message.
 */
export declare const GetFirewallRuleRequestSchema: GenMessage<GetFirewallRuleRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.GetFirewallRuleResponse
 */
export type GetFirewallRuleResponse = Message<"obiente.cloud.vps.v1.GetFirewallRuleResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallRule rule = 1;
     */
    rule?: FirewallRule;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetFirewallRuleResponse.
 * Use `create(GetFirewallRuleResponseSchema)` to create a new message.
 */
export declare const GetFirewallRuleResponseSchema: GenMessage<GetFirewallRuleResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.CreateFirewallRuleRequest
 */
export type CreateFirewallRuleRequest = Message<"obiente.cloud.vps.v1.CreateFirewallRuleRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallRule rule = 3;
     */
    rule?: FirewallRule;
    /**
     * Optional position to insert rule (defaults to end)
     *
     * @generated from field: optional int32 pos = 4;
     */
    pos?: number;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateFirewallRuleRequest.
 * Use `create(CreateFirewallRuleRequestSchema)` to create a new message.
 */
export declare const CreateFirewallRuleRequestSchema: GenMessage<CreateFirewallRuleRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.CreateFirewallRuleResponse
 */
export type CreateFirewallRuleResponse = Message<"obiente.cloud.vps.v1.CreateFirewallRuleResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallRule rule = 1;
     */
    rule?: FirewallRule;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateFirewallRuleResponse.
 * Use `create(CreateFirewallRuleResponseSchema)` to create a new message.
 */
export declare const CreateFirewallRuleResponseSchema: GenMessage<CreateFirewallRuleResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateFirewallRuleRequest
 */
export type UpdateFirewallRuleRequest = Message<"obiente.cloud.vps.v1.UpdateFirewallRuleRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * Rule position to update
     *
     * @generated from field: int32 rule_pos = 3;
     */
    rulePos: number;
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallRule rule = 4;
     */
    rule?: FirewallRule;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateFirewallRuleRequest.
 * Use `create(UpdateFirewallRuleRequestSchema)` to create a new message.
 */
export declare const UpdateFirewallRuleRequestSchema: GenMessage<UpdateFirewallRuleRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateFirewallRuleResponse
 */
export type UpdateFirewallRuleResponse = Message<"obiente.cloud.vps.v1.UpdateFirewallRuleResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallRule rule = 1;
     */
    rule?: FirewallRule;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateFirewallRuleResponse.
 * Use `create(UpdateFirewallRuleResponseSchema)` to create a new message.
 */
export declare const UpdateFirewallRuleResponseSchema: GenMessage<UpdateFirewallRuleResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.DeleteFirewallRuleRequest
 */
export type DeleteFirewallRuleRequest = Message<"obiente.cloud.vps.v1.DeleteFirewallRuleRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * Rule position to delete
     *
     * @generated from field: int32 rule_pos = 3;
     */
    rulePos: number;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteFirewallRuleRequest.
 * Use `create(DeleteFirewallRuleRequestSchema)` to create a new message.
 */
export declare const DeleteFirewallRuleRequestSchema: GenMessage<DeleteFirewallRuleRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.DeleteFirewallRuleResponse
 */
export type DeleteFirewallRuleResponse = Message<"obiente.cloud.vps.v1.DeleteFirewallRuleResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteFirewallRuleResponse.
 * Use `create(DeleteFirewallRuleResponseSchema)` to create a new message.
 */
export declare const DeleteFirewallRuleResponseSchema: GenMessage<DeleteFirewallRuleResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.GetFirewallOptionsRequest
 */
export type GetFirewallOptionsRequest = Message<"obiente.cloud.vps.v1.GetFirewallOptionsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetFirewallOptionsRequest.
 * Use `create(GetFirewallOptionsRequestSchema)` to create a new message.
 */
export declare const GetFirewallOptionsRequestSchema: GenMessage<GetFirewallOptionsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.GetFirewallOptionsResponse
 */
export type GetFirewallOptionsResponse = Message<"obiente.cloud.vps.v1.GetFirewallOptionsResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallOptions options = 1;
     */
    options?: FirewallOptions;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetFirewallOptionsResponse.
 * Use `create(GetFirewallOptionsResponseSchema)` to create a new message.
 */
export declare const GetFirewallOptionsResponseSchema: GenMessage<GetFirewallOptionsResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateFirewallOptionsRequest
 */
export type UpdateFirewallOptionsRequest = Message<"obiente.cloud.vps.v1.UpdateFirewallOptionsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallOptions options = 3;
     */
    options?: FirewallOptions;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateFirewallOptionsRequest.
 * Use `create(UpdateFirewallOptionsRequestSchema)` to create a new message.
 */
export declare const UpdateFirewallOptionsRequestSchema: GenMessage<UpdateFirewallOptionsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateFirewallOptionsResponse
 */
export type UpdateFirewallOptionsResponse = Message<"obiente.cloud.vps.v1.UpdateFirewallOptionsResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.FirewallOptions options = 1;
     */
    options?: FirewallOptions;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateFirewallOptionsResponse.
 * Use `create(UpdateFirewallOptionsResponseSchema)` to create a new message.
 */
export declare const UpdateFirewallOptionsResponseSchema: GenMessage<UpdateFirewallOptionsResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.FirewallRule
 */
export type FirewallRule = Message<"obiente.cloud.vps.v1.FirewallRule"> & {
    /**
     * Rule position (0-based index)
     *
     * @generated from field: int32 pos = 1;
     */
    pos: number;
    /**
     * Whether rule is enabled
     *
     * @generated from field: bool enable = 2;
     */
    enable: boolean;
    /**
     * ACCEPT, REJECT, DROP
     *
     * @generated from field: obiente.cloud.vps.v1.FirewallAction action = 3;
     */
    action: FirewallAction;
    /**
     * in, out
     *
     * @generated from field: obiente.cloud.vps.v1.FirewallDirection type = 4;
     */
    type: FirewallDirection;
    /**
     * Rule comment/description
     *
     * @generated from field: optional string comment = 5;
     */
    comment?: string;
    /**
     * Source/Destination
     *
     * Source IP/CIDR (e.g., "192.168.1.0/24")
     *
     * @generated from field: optional string source = 6;
     */
    source?: string;
    /**
     * Destination IP/CIDR
     *
     * @generated from field: optional string dest = 7;
     */
    dest?: string;
    /**
     * Network interface (e.g., "vmbr0")
     *
     * @generated from field: optional string iface = 8;
     */
    iface?: string;
    /**
     * Source MAC address
     *
     * @generated from field: optional string mac_source = 9;
     */
    macSource?: string;
    /**
     * Protocol and ports
     *
     * tcp, udp, icmp, etc.
     *
     * @generated from field: optional obiente.cloud.vps.v1.FirewallProtocol protocol = 10;
     */
    protocol?: FirewallProtocol;
    /**
     * Destination port(s) (e.g., "80", "80,443", "1000:2000")
     *
     * @generated from field: optional string dport = 11;
     */
    dport?: string;
    /**
     * Source port(s)
     *
     * @generated from field: optional string sport = 12;
     */
    sport?: string;
    /**
     * ICMP type (for ICMP protocol)
     *
     * @generated from field: optional int32 icmp_type = 13;
     */
    icmpType?: number;
    /**
     * Logging
     *
     * Enable logging for this rule
     *
     * @generated from field: optional bool log = 14;
     */
    log?: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.FirewallRule.
 * Use `create(FirewallRuleSchema)` to create a new message.
 */
export declare const FirewallRuleSchema: GenMessage<FirewallRule>;
/**
 * @generated from message obiente.cloud.vps.v1.FirewallOptions
 */
export type FirewallOptions = Message<"obiente.cloud.vps.v1.FirewallOptions"> & {
    /**
     * Enable firewall
     *
     * @generated from field: bool enable = 1;
     */
    enable: boolean;
    /**
     * Default policy for incoming traffic (ACCEPT, DROP, REJECT)
     *
     * @generated from field: optional string policy_in = 2;
     */
    policyIn?: string;
    /**
     * Default policy for outgoing traffic (ACCEPT, DROP, REJECT)
     *
     * @generated from field: optional string policy_out = 3;
     */
    policyOut?: string;
    /**
     * Log incoming traffic
     *
     * @generated from field: optional bool log_level_in = 4;
     */
    logLevelIn?: boolean;
    /**
     * Log outgoing traffic
     *
     * @generated from field: optional bool log_level_out = 5;
     */
    logLevelOut?: boolean;
    /**
     * Enable netfilter logging
     *
     * @generated from field: optional bool nf_log = 6;
     */
    nfLog?: boolean;
    /**
     * Allow DHCP
     *
     * @generated from field: optional bool dhcp = 7;
     */
    dhcp?: boolean;
    /**
     * Allow NDP (Neighbor Discovery Protocol)
     *
     * @generated from field: optional bool ndp = 8;
     */
    ndp?: boolean;
    /**
     * Allow Router Advertisement
     *
     * @generated from field: optional bool radv = 9;
     */
    radv?: boolean;
    /**
     * Enable IP filter
     *
     * @generated from field: optional bool ipfilter = 10;
     */
    ipfilter?: boolean;
    /**
     * Enable IP filter rules
     *
     * @generated from field: optional bool ipfilter_rules = 11;
     */
    ipfilterRules?: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.FirewallOptions.
 * Use `create(FirewallOptionsSchema)` to create a new message.
 */
export declare const FirewallOptionsSchema: GenMessage<FirewallOptions>;
/**
 * SSH key management messages
 *
 * @generated from message obiente.cloud.vps.v1.SSHKey
 */
export type SSHKey = Message<"obiente.cloud.vps.v1.SSHKey"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * SSH public key (e.g., "ssh-rsa AAAAB3NzaC1yc2E...")
     *
     * @generated from field: string public_key = 3;
     */
    publicKey: string;
    /**
     * SSH key fingerprint (MD5 or SHA256)
     *
     * @generated from field: string fingerprint = 4;
     */
    fingerprint: string;
    /**
     * If set, key is VPS-specific; if null, key is organization-wide
     *
     * @generated from field: optional string vps_id = 5;
     */
    vpsId?: string;
    /**
     * @generated from field: optional google.protobuf.Timestamp created_at = 6;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp updated_at = 7;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.SSHKey.
 * Use `create(SSHKeySchema)` to create a new message.
 */
export declare const SSHKeySchema: GenMessage<SSHKey>;
/**
 * @generated from message obiente.cloud.vps.v1.ListSSHKeysRequest
 */
export type ListSSHKeysRequest = Message<"obiente.cloud.vps.v1.ListSSHKeysRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * If provided, list keys for this VPS (includes org-wide keys); if null, list org-wide keys only
     *
     * @generated from field: optional string vps_id = 2;
     */
    vpsId?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListSSHKeysRequest.
 * Use `create(ListSSHKeysRequestSchema)` to create a new message.
 */
export declare const ListSSHKeysRequestSchema: GenMessage<ListSSHKeysRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ListSSHKeysResponse
 */
export type ListSSHKeysResponse = Message<"obiente.cloud.vps.v1.ListSSHKeysResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.SSHKey keys = 1;
     */
    keys: SSHKey[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ListSSHKeysResponse.
 * Use `create(ListSSHKeysResponseSchema)` to create a new message.
 */
export declare const ListSSHKeysResponseSchema: GenMessage<ListSSHKeysResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.AddSSHKeyRequest
 */
export type AddSSHKeyRequest = Message<"obiente.cloud.vps.v1.AddSSHKeyRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * User-friendly name for the key
     *
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * SSH public key content
     *
     * @generated from field: string public_key = 3;
     */
    publicKey: string;
    /**
     * If set, add key to this VPS; if null, add as organization-wide key
     *
     * @generated from field: optional string vps_id = 4;
     */
    vpsId?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.AddSSHKeyRequest.
 * Use `create(AddSSHKeyRequestSchema)` to create a new message.
 */
export declare const AddSSHKeyRequestSchema: GenMessage<AddSSHKeyRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.AddSSHKeyResponse
 */
export type AddSSHKeyResponse = Message<"obiente.cloud.vps.v1.AddSSHKeyResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.SSHKey key = 1;
     */
    key?: SSHKey;
};
/**
 * Describes the message obiente.cloud.vps.v1.AddSSHKeyResponse.
 * Use `create(AddSSHKeyResponseSchema)` to create a new message.
 */
export declare const AddSSHKeyResponseSchema: GenMessage<AddSSHKeyResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateSSHKeyRequest
 */
export type UpdateSSHKeyRequest = Message<"obiente.cloud.vps.v1.UpdateSSHKeyRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string key_id = 2;
     */
    keyId: string;
    /**
     * New name for the key
     *
     * @generated from field: string name = 3;
     */
    name: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateSSHKeyRequest.
 * Use `create(UpdateSSHKeyRequestSchema)` to create a new message.
 */
export declare const UpdateSSHKeyRequestSchema: GenMessage<UpdateSSHKeyRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.UpdateSSHKeyResponse
 */
export type UpdateSSHKeyResponse = Message<"obiente.cloud.vps.v1.UpdateSSHKeyResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.SSHKey key = 1;
     */
    key?: SSHKey;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateSSHKeyResponse.
 * Use `create(UpdateSSHKeyResponseSchema)` to create a new message.
 */
export declare const UpdateSSHKeyResponseSchema: GenMessage<UpdateSSHKeyResponse>;
/**
 * @generated from message obiente.cloud.vps.v1.RemoveSSHKeyRequest
 */
export type RemoveSSHKeyRequest = Message<"obiente.cloud.vps.v1.RemoveSSHKeyRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string key_id = 2;
     */
    keyId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RemoveSSHKeyRequest.
 * Use `create(RemoveSSHKeyRequestSchema)` to create a new message.
 */
export declare const RemoveSSHKeyRequestSchema: GenMessage<RemoveSSHKeyRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.RemoveSSHKeyResponse
 */
export type RemoveSSHKeyResponse = Message<"obiente.cloud.vps.v1.RemoveSSHKeyResponse"> & {
    /**
     * List of VPS instances that will be affected by removing this org-wide key
     * Only populated for org-wide keys (vps_id is null)
     *
     * @generated from field: repeated string affected_vps_ids = 1;
     */
    affectedVpsIds: string[];
    /**
     * Corresponding VPS names for the IDs
     *
     * @generated from field: repeated string affected_vps_names = 2;
     */
    affectedVpsNames: string[];
};
/**
 * Describes the message obiente.cloud.vps.v1.RemoveSSHKeyResponse.
 * Use `create(RemoveSSHKeyResponseSchema)` to create a new message.
 */
export declare const RemoveSSHKeyResponseSchema: GenMessage<RemoveSSHKeyResponse>;
/**
 * Password reset messages
 *
 * @generated from message obiente.cloud.vps.v1.ResetVPSPasswordRequest
 */
export type ResetVPSPasswordRequest = Message<"obiente.cloud.vps.v1.ResetVPSPasswordRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ResetVPSPasswordRequest.
 * Use `create(ResetVPSPasswordRequestSchema)` to create a new message.
 */
export declare const ResetVPSPasswordRequestSchema: GenMessage<ResetVPSPasswordRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ResetVPSPasswordResponse
 */
export type ResetVPSPasswordResponse = Message<"obiente.cloud.vps.v1.ResetVPSPasswordResponse"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * New root password (shown once, never stored)
     *
     * @generated from field: string root_password = 2;
     */
    rootPassword: string;
    /**
     * Note: Password will take effect after VM reboot or cloud-init re-run
     *
     * Instructions for applying the password change
     *
     * @generated from field: string message = 3;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ResetVPSPasswordResponse.
 * Use `create(ResetVPSPasswordResponseSchema)` to create a new message.
 */
export declare const ResetVPSPasswordResponseSchema: GenMessage<ResetVPSPasswordResponse>;
/**
 * Reinitialize VPS messages
 *
 * @generated from message obiente.cloud.vps.v1.ReinitializeVPSRequest
 */
export type ReinitializeVPSRequest = Message<"obiente.cloud.vps.v1.ReinitializeVPSRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ReinitializeVPSRequest.
 * Use `create(ReinitializeVPSRequestSchema)` to create a new message.
 */
export declare const ReinitializeVPSRequestSchema: GenMessage<ReinitializeVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ReinitializeVPSResponse
 */
export type ReinitializeVPSResponse = Message<"obiente.cloud.vps.v1.ReinitializeVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * Generated password (shown once, never stored)
     *
     * @generated from field: optional string root_password = 2;
     */
    rootPassword?: string;
    /**
     * Information message
     *
     * @generated from field: string message = 3;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ReinitializeVPSResponse.
 * Use `create(ReinitializeVPSResponseSchema)` to create a new message.
 */
export declare const ReinitializeVPSResponseSchema: GenMessage<ReinitializeVPSResponse>;
/**
 * VPS log streaming messages
 *
 * @generated from message obiente.cloud.vps.v1.StreamVPSLogsRequest
 */
export type StreamVPSLogsRequest = Message<"obiente.cloud.vps.v1.StreamVPSLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.StreamVPSLogsRequest.
 * Use `create(StreamVPSLogsRequestSchema)` to create a new message.
 */
export declare const StreamVPSLogsRequestSchema: GenMessage<StreamVPSLogsRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSLogLine
 */
export type VPSLogLine = Message<"obiente.cloud.vps.v1.VPSLogLine"> & {
    /**
     * Log line content
     *
     * @generated from field: string line = 1;
     */
    line: string;
    /**
     * Whether this is stderr output
     *
     * @generated from field: bool stderr = 2;
     */
    stderr: boolean;
    /**
     * Sequential line number
     *
     * @generated from field: int32 line_number = 3;
     */
    lineNumber: number;
    /**
     * Timestamp when log was written
     *
     * @generated from field: google.protobuf.Timestamp timestamp = 4;
     */
    timestamp?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSLogLine.
 * Use `create(VPSLogLineSchema)` to create a new message.
 */
export declare const VPSLogLineSchema: GenMessage<VPSLogLine>;
/**
 * Import VPS messages
 *
 * @generated from message obiente.cloud.vps.v1.ImportVPSRequest
 */
export type ImportVPSRequest = Message<"obiente.cloud.vps.v1.ImportVPSRequest"> & {
    /**
     * Organization to import VPS for (must match VPS ownership)
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ImportVPSRequest.
 * Use `create(ImportVPSRequestSchema)` to create a new message.
 */
export declare const ImportVPSRequestSchema: GenMessage<ImportVPSRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ImportVPSResponse
 */
export type ImportVPSResponse = Message<"obiente.cloud.vps.v1.ImportVPSResponse"> & {
    /**
     * Number of VPS instances imported
     *
     * @generated from field: int32 imported_count = 1;
     */
    importedCount: number;
    /**
     * List of imported VPS instances
     *
     * @generated from field: repeated obiente.cloud.vps.v1.VPSInstance imported_vps = 2;
     */
    importedVps: VPSInstance[];
    /**
     * Number of VPS instances skipped (already exist or don't belong to org)
     *
     * @generated from field: int32 skipped_count = 3;
     */
    skippedCount: number;
    /**
     * Any errors encountered during import
     *
     * @generated from field: repeated string errors = 4;
     */
    errors: string[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ImportVPSResponse.
 * Use `create(ImportVPSResponseSchema)` to create a new message.
 */
export declare const ImportVPSResponseSchema: GenMessage<ImportVPSResponse>;
/**
 * GetVPSLeases retrieves DHCP lease information for VPS instances
 *
 * @generated from message obiente.cloud.vps.v1.GetVPSLeasesRequest
 */
export type GetVPSLeasesRequest = Message<"obiente.cloud.vps.v1.GetVPSLeasesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: filter by specific VPS
     *
     * @generated from field: optional string vps_id = 2;
     */
    vpsId?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSLeasesRequest.
 * Use `create(GetVPSLeasesRequestSchema)` to create a new message.
 */
export declare const GetVPSLeasesRequestSchema: GenMessage<GetVPSLeasesRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.VPSLease
 */
export type VPSLease = Message<"obiente.cloud.vps.v1.VPSLease"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string mac_address = 3;
     */
    macAddress: string;
    /**
     * @generated from field: string ip_address = 4;
     */
    ipAddress: string;
    /**
     * @generated from field: google.protobuf.Timestamp expires_at = 5;
     */
    expiresAt?: Timestamp;
    /**
     * True if IP is public (outside DHCP pool)
     *
     * @generated from field: bool is_public = 6;
     */
    isPublic: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSLease.
 * Use `create(VPSLeaseSchema)` to create a new message.
 */
export declare const VPSLeaseSchema: GenMessage<VPSLease>;
/**
 * @generated from message obiente.cloud.vps.v1.GetVPSLeasesResponse
 */
export type GetVPSLeasesResponse = Message<"obiente.cloud.vps.v1.GetVPSLeasesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSLease leases = 1;
     */
    leases: VPSLease[];
};
/**
 * Describes the message obiente.cloud.vps.v1.GetVPSLeasesResponse.
 * Use `create(GetVPSLeasesResponseSchema)` to create a new message.
 */
export declare const GetVPSLeasesResponseSchema: GenMessage<GetVPSLeasesResponse>;
/**
 * RegisterLeaseRequest is called by gateway nodes to register a new DHCP lease
 *
 * @generated from message obiente.cloud.vps.v1.RegisterLeaseRequest
 */
export type RegisterLeaseRequest = Message<"obiente.cloud.vps.v1.RegisterLeaseRequest"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string mac_address = 3;
     */
    macAddress: string;
    /**
     * @generated from field: string ip_address = 4;
     */
    ipAddress: string;
    /**
     * @generated from field: google.protobuf.Timestamp expires_at = 5;
     */
    expiresAt?: Timestamp;
    /**
     * True if IP is public (outside DHCP pool)
     *
     * @generated from field: bool is_public = 6;
     */
    isPublic: boolean;
    /**
     * Gateway node name - identifies which gateway is registering this lease
     *
     * @generated from field: string gateway_node = 7;
     */
    gatewayNode: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RegisterLeaseRequest.
 * Use `create(RegisterLeaseRequestSchema)` to create a new message.
 */
export declare const RegisterLeaseRequestSchema: GenMessage<RegisterLeaseRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.RegisterLeaseResponse
 */
export type RegisterLeaseResponse = Message<"obiente.cloud.vps.v1.RegisterLeaseResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RegisterLeaseResponse.
 * Use `create(RegisterLeaseResponseSchema)` to create a new message.
 */
export declare const RegisterLeaseResponseSchema: GenMessage<RegisterLeaseResponse>;
/**
 * ReleaseLeaseRequest is called by gateway nodes to release a DHCP lease
 *
 * @generated from message obiente.cloud.vps.v1.ReleaseLeaseRequest
 */
export type ReleaseLeaseRequest = Message<"obiente.cloud.vps.v1.ReleaseLeaseRequest"> & {
    /**
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * Optional: for verification
     *
     * @generated from field: string mac_address = 2;
     */
    macAddress: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ReleaseLeaseRequest.
 * Use `create(ReleaseLeaseRequestSchema)` to create a new message.
 */
export declare const ReleaseLeaseRequestSchema: GenMessage<ReleaseLeaseRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.ReleaseLeaseResponse
 */
export type ReleaseLeaseResponse = Message<"obiente.cloud.vps.v1.ReleaseLeaseResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.ReleaseLeaseResponse.
 * Use `create(ReleaseLeaseResponseSchema)` to create a new message.
 */
export declare const ReleaseLeaseResponseSchema: GenMessage<ReleaseLeaseResponse>;
/**
 * AssignVPSPublicIP assigns a public IP to a VPS and triggers DHCP lease creation
 *
 * @generated from message obiente.cloud.vps.v1.AssignVPSPublicIPRequest
 */
export type AssignVPSPublicIPRequest = Message<"obiente.cloud.vps.v1.AssignVPSPublicIPRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: string public_ip = 3;
     */
    publicIp: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.AssignVPSPublicIPRequest.
 * Use `create(AssignVPSPublicIPRequestSchema)` to create a new message.
 */
export declare const AssignVPSPublicIPRequestSchema: GenMessage<AssignVPSPublicIPRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.AssignVPSPublicIPResponse
 */
export type AssignVPSPublicIPResponse = Message<"obiente.cloud.vps.v1.AssignVPSPublicIPResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.AssignVPSPublicIPResponse.
 * Use `create(AssignVPSPublicIPResponseSchema)` to create a new message.
 */
export declare const AssignVPSPublicIPResponseSchema: GenMessage<AssignVPSPublicIPResponse>;
/**
 * UnassignVPSPublicIP removes a public IP from a VPS and removes DHCP lease
 *
 * @generated from message obiente.cloud.vps.v1.UnassignVPSPublicIPRequest
 */
export type UnassignVPSPublicIPRequest = Message<"obiente.cloud.vps.v1.UnassignVPSPublicIPRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: string public_ip = 3;
     */
    publicIp: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UnassignVPSPublicIPRequest.
 * Use `create(UnassignVPSPublicIPRequestSchema)` to create a new message.
 */
export declare const UnassignVPSPublicIPRequestSchema: GenMessage<UnassignVPSPublicIPRequest>;
/**
 * @generated from message obiente.cloud.vps.v1.UnassignVPSPublicIPResponse
 */
export type UnassignVPSPublicIPResponse = Message<"obiente.cloud.vps.v1.UnassignVPSPublicIPResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UnassignVPSPublicIPResponse.
 * Use `create(UnassignVPSPublicIPResponseSchema)` to create a new message.
 */
export declare const UnassignVPSPublicIPResponseSchema: GenMessage<UnassignVPSPublicIPResponse>;
/**
 * VPS Public IP
 *
 * @generated from message obiente.cloud.vps.v1.VPSPublicIP
 */
export type VPSPublicIP = Message<"obiente.cloud.vps.v1.VPSPublicIP"> & {
    /**
     * IP record ID
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * IPv4 or IPv6 address
     *
     * @generated from field: string ip_address = 2;
     */
    ipAddress: string;
    /**
     * Assigned VPS instance ID (null if unassigned)
     *
     * @generated from field: optional string vps_id = 3;
     */
    vpsId?: string;
    /**
     * Organization that owns the VPS (null if unassigned)
     *
     * @generated from field: optional string organization_id = 4;
     */
    organizationId?: string;
    /**
     * VPS name (for display, null if unassigned)
     *
     * @generated from field: optional string vps_name = 5;
     */
    vpsName?: string;
    /**
     * Organization name (for display, null if unassigned)
     *
     * @generated from field: optional string organization_name = 6;
     */
    organizationName?: string;
    /**
     * Monthly cost in cents
     *
     * @generated from field: int64 monthly_cost_cents = 7;
     */
    monthlyCostCents: bigint;
    /**
     * Gateway IP for this public IP (null if not set)
     *
     * @generated from field: optional string gateway = 11;
     */
    gateway?: string;
    /**
     * Netmask/CIDR for this public IP (null if not set, e.g., "24" or "255.255.255.0")
     *
     * @generated from field: optional string netmask = 12;
     */
    netmask?: string;
    /**
     * When IP was assigned to VPS
     *
     * @generated from field: optional google.protobuf.Timestamp assigned_at = 8;
     */
    assignedAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 9;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 10;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSPublicIP.
 * Use `create(VPSPublicIPSchema)` to create a new message.
 */
export declare const VPSPublicIPSchema: GenMessage<VPSPublicIP>;
/**
 * List VPS Public IPs Request
 *
 * @generated from message obiente.cloud.vps.v1.ListVPSPublicIPsRequest
 */
export type ListVPSPublicIPsRequest = Message<"obiente.cloud.vps.v1.ListVPSPublicIPsRequest"> & {
    /**
     * Filter by VPS ID (optional)
     *
     * @generated from field: optional string vps_id = 1;
     */
    vpsId?: string;
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 2;
     */
    organizationId?: string;
    /**
     * Include unassigned IPs (default: true)
     *
     * @generated from field: optional bool include_unassigned = 3;
     */
    includeUnassigned?: boolean;
    /**
     * Page number (default: 1)
     *
     * @generated from field: int32 page = 4;
     */
    page: number;
    /**
     * Items per page (default: 50, max: 100)
     *
     * @generated from field: int32 per_page = 5;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSPublicIPsRequest.
 * Use `create(ListVPSPublicIPsRequestSchema)` to create a new message.
 */
export declare const ListVPSPublicIPsRequestSchema: GenMessage<ListVPSPublicIPsRequest>;
/**
 * List VPS Public IPs Response
 *
 * @generated from message obiente.cloud.vps.v1.ListVPSPublicIPsResponse
 */
export type ListVPSPublicIPsResponse = Message<"obiente.cloud.vps.v1.ListVPSPublicIPsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSPublicIP ips = 1;
     */
    ips: VPSPublicIP[];
    /**
     * Total number of IPs matching filters
     *
     * @generated from field: int64 total_count = 2;
     */
    totalCount: bigint;
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSPublicIPsResponse.
 * Use `create(ListVPSPublicIPsResponseSchema)` to create a new message.
 */
export declare const ListVPSPublicIPsResponseSchema: GenMessage<ListVPSPublicIPsResponse>;
/**
 * Create VPS Public IP Request
 *
 * @generated from message obiente.cloud.vps.v1.CreateVPSPublicIPRequest
 */
export type CreateVPSPublicIPRequest = Message<"obiente.cloud.vps.v1.CreateVPSPublicIPRequest"> & {
    /**
     * IPv4 or IPv6 address
     *
     * @generated from field: string ip_address = 1;
     */
    ipAddress: string;
    /**
     * Monthly cost in cents (set by superadmin)
     *
     * @generated from field: int64 monthly_cost_cents = 2;
     */
    monthlyCostCents: bigint;
    /**
     * Gateway IP for this public IP (set by superadmin)
     *
     * @generated from field: optional string gateway = 3;
     */
    gateway?: string;
    /**
     * Netmask/CIDR for this public IP (set by superadmin, e.g., "24" or "255.255.255.0")
     *
     * @generated from field: optional string netmask = 4;
     */
    netmask?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSPublicIPRequest.
 * Use `create(CreateVPSPublicIPRequestSchema)` to create a new message.
 */
export declare const CreateVPSPublicIPRequestSchema: GenMessage<CreateVPSPublicIPRequest>;
/**
 * Create VPS Public IP Response
 *
 * @generated from message obiente.cloud.vps.v1.CreateVPSPublicIPResponse
 */
export type CreateVPSPublicIPResponse = Message<"obiente.cloud.vps.v1.CreateVPSPublicIPResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSPublicIP ip = 1;
     */
    ip?: VPSPublicIP;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSPublicIPResponse.
 * Use `create(CreateVPSPublicIPResponseSchema)` to create a new message.
 */
export declare const CreateVPSPublicIPResponseSchema: GenMessage<CreateVPSPublicIPResponse>;
/**
 * Update VPS Public IP Request
 *
 * @generated from message obiente.cloud.vps.v1.UpdateVPSPublicIPRequest
 */
export type UpdateVPSPublicIPRequest = Message<"obiente.cloud.vps.v1.UpdateVPSPublicIPRequest"> & {
    /**
     * IP ID to update
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Update monthly cost in cents
     *
     * @generated from field: optional int64 monthly_cost_cents = 2;
     */
    monthlyCostCents?: bigint;
    /**
     * Update gateway IP for this public IP
     *
     * @generated from field: optional string gateway = 3;
     */
    gateway?: string;
    /**
     * Update netmask/CIDR for this public IP (e.g., "24" or "255.255.255.0")
     *
     * @generated from field: optional string netmask = 4;
     */
    netmask?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSPublicIPRequest.
 * Use `create(UpdateVPSPublicIPRequestSchema)` to create a new message.
 */
export declare const UpdateVPSPublicIPRequestSchema: GenMessage<UpdateVPSPublicIPRequest>;
/**
 * Update VPS Public IP Response
 *
 * @generated from message obiente.cloud.vps.v1.UpdateVPSPublicIPResponse
 */
export type UpdateVPSPublicIPResponse = Message<"obiente.cloud.vps.v1.UpdateVPSPublicIPResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSPublicIP ip = 1;
     */
    ip?: VPSPublicIP;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSPublicIPResponse.
 * Use `create(UpdateVPSPublicIPResponseSchema)` to create a new message.
 */
export declare const UpdateVPSPublicIPResponseSchema: GenMessage<UpdateVPSPublicIPResponse>;
/**
 * Delete VPS Public IP Request
 *
 * @generated from message obiente.cloud.vps.v1.DeleteVPSPublicIPRequest
 */
export type DeleteVPSPublicIPRequest = Message<"obiente.cloud.vps.v1.DeleteVPSPublicIPRequest"> & {
    /**
     * IP ID to delete
     *
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSPublicIPRequest.
 * Use `create(DeleteVPSPublicIPRequestSchema)` to create a new message.
 */
export declare const DeleteVPSPublicIPRequestSchema: GenMessage<DeleteVPSPublicIPRequest>;
/**
 * Delete VPS Public IP Response
 *
 * @generated from message obiente.cloud.vps.v1.DeleteVPSPublicIPResponse
 */
export type DeleteVPSPublicIPResponse = Message<"obiente.cloud.vps.v1.DeleteVPSPublicIPResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSPublicIPResponse.
 * Use `create(DeleteVPSPublicIPResponseSchema)` to create a new message.
 */
export declare const DeleteVPSPublicIPResponseSchema: GenMessage<DeleteVPSPublicIPResponse>;
/**
 * VPSStatus represents the current status of a VPS instance
 *
 * @generated from enum obiente.cloud.vps.v1.VPSStatus
 */
export declare enum VPSStatus {
    /**
     * @generated from enum value: VPS_STATUS_UNSPECIFIED = 0;
     */
    VPS_STATUS_UNSPECIFIED = 0,
    /**
     * VPS is being provisioned
     *
     * @generated from enum value: CREATING = 1;
     */
    CREATING = 1,
    /**
     * VPS is starting up
     *
     * @generated from enum value: STARTING = 2;
     */
    STARTING = 2,
    /**
     * VPS is running
     *
     * @generated from enum value: RUNNING = 3;
     */
    RUNNING = 3,
    /**
     * VPS is stopping
     *
     * @generated from enum value: STOPPING = 4;
     */
    STOPPING = 4,
    /**
     * VPS is stopped
     *
     * @generated from enum value: STOPPED = 5;
     */
    STOPPED = 5,
    /**
     * VPS is rebooting
     *
     * @generated from enum value: REBOOTING = 6;
     */
    REBOOTING = 6,
    /**
     * VPS provisioning or operation failed
     *
     * @generated from enum value: FAILED = 7;
     */
    FAILED = 7,
    /**
     * VPS is being deleted
     *
     * @generated from enum value: DELETING = 8;
     */
    DELETING = 8,
    /**
     * VPS has been deleted (soft delete)
     *
     * @generated from enum value: DELETED = 9;
     */
    DELETED = 9,
    /**
     * VPS is suspended (superadmin action, prevents normal operations)
     *
     * @generated from enum value: SUSPENDED = 10;
     */
    SUSPENDED = 10
}
/**
 * Describes the enum obiente.cloud.vps.v1.VPSStatus.
 */
export declare const VPSStatusSchema: GenEnum<VPSStatus>;
/**
 * VPSImage represents the OS image for the VPS
 *
 * @generated from enum obiente.cloud.vps.v1.VPSImage
 */
export declare enum VPSImage {
    /**
     * @generated from enum value: VPS_IMAGE_UNSPECIFIED = 0;
     */
    VPS_IMAGE_UNSPECIFIED = 0,
    /**
     * Ubuntu 22.04 LTS
     *
     * @generated from enum value: UBUNTU_22_04 = 1;
     */
    UBUNTU_22_04 = 1,
    /**
     * Ubuntu 24.04 LTS
     *
     * @generated from enum value: UBUNTU_24_04 = 2;
     */
    UBUNTU_24_04 = 2,
    /**
     * Debian 12
     *
     * @generated from enum value: DEBIAN_12 = 3;
     */
    DEBIAN_12 = 3,
    /**
     * Debian 13
     *
     * @generated from enum value: DEBIAN_13 = 4;
     */
    DEBIAN_13 = 4,
    /**
     * Rocky Linux 9
     *
     * @generated from enum value: ROCKY_LINUX_9 = 5;
     */
    ROCKY_LINUX_9 = 5,
    /**
     * AlmaLinux 9
     *
     * @generated from enum value: ALMA_LINUX_9 = 6;
     */
    ALMA_LINUX_9 = 6,
    /**
     * Custom image (specified via image_id)
     *
     * @generated from enum value: CUSTOM = 99;
     */
    CUSTOM = 99
}
/**
 * Describes the enum obiente.cloud.vps.v1.VPSImage.
 */
export declare const VPSImageSchema: GenEnum<VPSImage>;
/**
 * @generated from enum obiente.cloud.vps.v1.FirewallAction
 */
export declare enum FirewallAction {
    /**
     * @generated from enum value: FIREWALL_ACTION_UNSPECIFIED = 0;
     */
    FIREWALL_ACTION_UNSPECIFIED = 0,
    /**
     * @generated from enum value: ACCEPT = 1;
     */
    ACCEPT = 1,
    /**
     * @generated from enum value: REJECT = 2;
     */
    REJECT = 2,
    /**
     * @generated from enum value: DROP = 3;
     */
    DROP = 3
}
/**
 * Describes the enum obiente.cloud.vps.v1.FirewallAction.
 */
export declare const FirewallActionSchema: GenEnum<FirewallAction>;
/**
 * @generated from enum obiente.cloud.vps.v1.FirewallDirection
 */
export declare enum FirewallDirection {
    /**
     * @generated from enum value: FIREWALL_DIRECTION_UNSPECIFIED = 0;
     */
    FIREWALL_DIRECTION_UNSPECIFIED = 0,
    /**
     * Incoming traffic
     *
     * @generated from enum value: IN = 1;
     */
    IN = 1,
    /**
     * Outgoing traffic
     *
     * @generated from enum value: OUT = 2;
     */
    OUT = 2
}
/**
 * Describes the enum obiente.cloud.vps.v1.FirewallDirection.
 */
export declare const FirewallDirectionSchema: GenEnum<FirewallDirection>;
/**
 * @generated from enum obiente.cloud.vps.v1.FirewallProtocol
 */
export declare enum FirewallProtocol {
    /**
     * @generated from enum value: FIREWALL_PROTOCOL_UNSPECIFIED = 0;
     */
    FIREWALL_PROTOCOL_UNSPECIFIED = 0,
    /**
     * @generated from enum value: TCP = 1;
     */
    TCP = 1,
    /**
     * @generated from enum value: UDP = 2;
     */
    UDP = 2,
    /**
     * @generated from enum value: ICMP = 3;
     */
    ICMP = 3,
    /**
     * @generated from enum value: ICMPV6 = 4;
     */
    ICMPV6 = 4,
    /**
     * All protocols
     *
     * @generated from enum value: ALL = 5;
     */
    ALL = 5
}
/**
 * Describes the enum obiente.cloud.vps.v1.FirewallProtocol.
 */
export declare const FirewallProtocolSchema: GenEnum<FirewallProtocol>;
/**
 * @generated from service obiente.cloud.vps.v1.VPSService
 */
export declare const VPSService: GenService<{
    /**
     * List organization VPS instances
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ListVPS
     */
    listVPS: {
        methodKind: "unary";
        input: typeof ListVPSRequestSchema;
        output: typeof ListVPSResponseSchema;
    };
    /**
     * Create new VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.CreateVPS
     */
    createVPS: {
        methodKind: "unary";
        input: typeof CreateVPSRequestSchema;
        output: typeof CreateVPSResponseSchema;
    };
    /**
     * Get VPS instance details
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetVPS
     */
    getVPS: {
        methodKind: "unary";
        input: typeof GetVPSRequestSchema;
        output: typeof GetVPSResponseSchema;
    };
    /**
     * Update VPS instance configuration
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.UpdateVPS
     */
    updateVPS: {
        methodKind: "unary";
        input: typeof UpdateVPSRequestSchema;
        output: typeof UpdateVPSResponseSchema;
    };
    /**
     * Delete VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.DeleteVPS
     */
    deleteVPS: {
        methodKind: "unary";
        input: typeof DeleteVPSRequestSchema;
        output: typeof DeleteVPSResponseSchema;
    };
    /**
     * Start a stopped VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.StartVPS
     */
    startVPS: {
        methodKind: "unary";
        input: typeof StartVPSRequestSchema;
        output: typeof StartVPSResponseSchema;
    };
    /**
     * Stop a running VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.StopVPS
     */
    stopVPS: {
        methodKind: "unary";
        input: typeof StopVPSRequestSchema;
        output: typeof StopVPSResponseSchema;
    };
    /**
     * Reboot a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.RebootVPS
     */
    rebootVPS: {
        methodKind: "unary";
        input: typeof RebootVPSRequestSchema;
        output: typeof RebootVPSResponseSchema;
    };
    /**
     * Stream VPS status updates
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.StreamVPSStatus
     */
    streamVPSStatus: {
        methodKind: "server_streaming";
        input: typeof StreamVPSStatusRequestSchema;
        output: typeof VPSStatusUpdateSchema;
    };
    /**
     * Get VPS instance metrics (real-time or historical)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetVPSMetrics
     */
    getVPSMetrics: {
        methodKind: "unary";
        input: typeof GetVPSMetricsRequestSchema;
        output: typeof GetVPSMetricsResponseSchema;
    };
    /**
     * Stream real-time VPS metrics
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.StreamVPSMetrics
     */
    streamVPSMetrics: {
        methodKind: "server_streaming";
        input: typeof StreamVPSMetricsRequestSchema;
        output: typeof VPSMetricSchema;
    };
    /**
     * Get aggregated usage for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetVPSUsage
     */
    getVPSUsage: {
        methodKind: "unary";
        input: typeof GetVPSUsageRequestSchema;
        output: typeof GetVPSUsageResponseSchema;
    };
    /**
     * Get available VPS sizes/pricing
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ListVPSSizes
     */
    listVPSSizes: {
        methodKind: "unary";
        input: typeof ListAvailableVPSSizesRequestSchema;
        output: typeof ListAvailableVPSSizesResponseSchema;
    };
    /**
     * Get available VPS regions/locations
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ListVPSRegions
     */
    listVPSRegions: {
        methodKind: "unary";
        input: typeof ListVPSRegionsRequestSchema;
        output: typeof ListVPSRegionsResponseSchema;
    };
    /**
     * Get VPS proxy connection info (for accessing VPS without dedicated IP)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetVPSProxyInfo
     */
    getVPSProxyInfo: {
        methodKind: "unary";
        input: typeof GetVPSProxyInfoRequestSchema;
        output: typeof GetVPSProxyInfoResponseSchema;
    };
    /**
     * Firewall management
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ListFirewallRules
     */
    listFirewallRules: {
        methodKind: "unary";
        input: typeof ListFirewallRulesRequestSchema;
        output: typeof ListFirewallRulesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetFirewallRule
     */
    getFirewallRule: {
        methodKind: "unary";
        input: typeof GetFirewallRuleRequestSchema;
        output: typeof GetFirewallRuleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.CreateFirewallRule
     */
    createFirewallRule: {
        methodKind: "unary";
        input: typeof CreateFirewallRuleRequestSchema;
        output: typeof CreateFirewallRuleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.UpdateFirewallRule
     */
    updateFirewallRule: {
        methodKind: "unary";
        input: typeof UpdateFirewallRuleRequestSchema;
        output: typeof UpdateFirewallRuleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.DeleteFirewallRule
     */
    deleteFirewallRule: {
        methodKind: "unary";
        input: typeof DeleteFirewallRuleRequestSchema;
        output: typeof DeleteFirewallRuleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetFirewallOptions
     */
    getFirewallOptions: {
        methodKind: "unary";
        input: typeof GetFirewallOptionsRequestSchema;
        output: typeof GetFirewallOptionsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.UpdateFirewallOptions
     */
    updateFirewallOptions: {
        methodKind: "unary";
        input: typeof UpdateFirewallOptionsRequestSchema;
        output: typeof UpdateFirewallOptionsResponseSchema;
    };
    /**
     * SSH key management
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ListSSHKeys
     */
    listSSHKeys: {
        methodKind: "unary";
        input: typeof ListSSHKeysRequestSchema;
        output: typeof ListSSHKeysResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.AddSSHKey
     */
    addSSHKey: {
        methodKind: "unary";
        input: typeof AddSSHKeyRequestSchema;
        output: typeof AddSSHKeyResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.UpdateSSHKey
     */
    updateSSHKey: {
        methodKind: "unary";
        input: typeof UpdateSSHKeyRequestSchema;
        output: typeof UpdateSSHKeyResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.vps.v1.VPSService.RemoveSSHKey
     */
    removeSSHKey: {
        methodKind: "unary";
        input: typeof RemoveSSHKeyRequestSchema;
        output: typeof RemoveSSHKeyResponseSchema;
    };
    /**
     * Reset root password for a VPS instance
     * Password is generated and returned once, then discarded (never stored)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ResetVPSPassword
     */
    resetVPSPassword: {
        methodKind: "unary";
        input: typeof ResetVPSPasswordRequestSchema;
        output: typeof ResetVPSPasswordResponseSchema;
    };
    /**
     * Reinitialize a VPS instance
     * This will delete all data on the VPS and reinstall the operating system
     * The VPS will be reconfigured with the same cloud-init settings
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ReinitializeVPS
     */
    reinitializeVPS: {
        methodKind: "unary";
        input: typeof ReinitializeVPSRequestSchema;
        output: typeof ReinitializeVPSResponseSchema;
    };
    /**
     * Stream VPS provisioning logs
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.StreamVPSLogs
     */
    streamVPSLogs: {
        methodKind: "server_streaming";
        input: typeof StreamVPSLogsRequestSchema;
        output: typeof VPSLogLineSchema;
    };
    /**
     * Import missing VPS instances from Proxmox that belong to the organization
     * This will scan Proxmox for VMs with Obiente Cloud descriptions and import
     * any that are missing from the database, ensuring they belong to the requesting organization
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ImportVPS
     */
    importVPS: {
        methodKind: "unary";
        input: typeof ImportVPSRequestSchema;
        output: typeof ImportVPSResponseSchema;
    };
    /**
     * Get DHCP leases for VPS instances in an organization
     * Leases are stored in the database and synced from gateway nodes
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.GetVPSLeases
     */
    getVPSLeases: {
        methodKind: "unary";
        input: typeof GetVPSLeasesRequestSchema;
        output: typeof GetVPSLeasesResponseSchema;
    };
    /**
     * FindVPSByLease finds a VPS by DHCP lease information (IP or MAC). Returns
     * the VPS ID and organization if a matching DHCP lease exists in the
     * database. This is used by gateways to resolve a lease to a managed VPS
     * when the client-supplied hostname does not contain the VPS identifier.
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.FindVPSByLease
     */
    findVPSByLease: {
        methodKind: "unary";
        input: typeof FindVPSByLeaseRequestSchema;
        output: typeof FindVPSByLeaseResponseSchema;
    };
    /**
     * Register a new DHCP lease (called by gateway nodes via persistent connection)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.RegisterLease
     */
    registerLease: {
        methodKind: "unary";
        input: typeof RegisterLeaseRequestSchema;
        output: typeof RegisterLeaseResponseSchema;
    };
    /**
     * Release a DHCP lease (called by gateway nodes via persistent connection)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.ReleaseLease
     */
    releaseLease: {
        methodKind: "unary";
        input: typeof ReleaseLeaseRequestSchema;
        output: typeof ReleaseLeaseResponseSchema;
    };
    /**
     * Assign a public IP to a VPS (triggers DHCP static lease creation)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.AssignVPSPublicIP
     */
    assignVPSPublicIP: {
        methodKind: "unary";
        input: typeof AssignVPSPublicIPRequestSchema;
        output: typeof AssignVPSPublicIPResponseSchema;
    };
    /**
     * Unassign a public IP from a VPS (removes DHCP static lease)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSService.UnassignVPSPublicIP
     */
    unassignVPSPublicIP: {
        methodKind: "unary";
        input: typeof UnassignVPSPublicIPRequestSchema;
        output: typeof UnassignVPSPublicIPResponseSchema;
    };
}>;
