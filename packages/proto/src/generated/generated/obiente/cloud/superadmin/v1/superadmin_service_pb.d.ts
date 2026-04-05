import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { DeploymentStatus, Environment } from "../../deployments/v1/deployment_service_pb";
import type { Invoice } from "../../billing/v1/billing_service_pb";
import type { Pagination, VPSSize } from "../../common/v1/common_pb";
import type { AssignVPSPublicIPRequestSchema, AssignVPSPublicIPResponseSchema, CloudInitConfig, CreateVPSPublicIPRequestSchema, CreateVPSPublicIPResponseSchema, DeleteVPSPublicIPRequestSchema, DeleteVPSPublicIPResponseSchema, ListVPSPublicIPsRequestSchema, ListVPSPublicIPsResponseSchema, UnassignVPSPublicIPRequestSchema, UnassignVPSPublicIPResponseSchema, UpdateVPSPublicIPRequestSchema, UpdateVPSPublicIPResponseSchema, VPSInstance, VPSStatus } from "../../vps/v1/vps_service_pb";
import type { GetOrgLeasesRequestSchema, GetOrgLeasesResponseSchema } from "../../vpsgateway/v1/gateway_service_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/superadmin/v1/superadmin_service.proto.
 */
export declare const file_obiente_cloud_superadmin_v1_superadmin_service: GenFile;
/**
 * @generated from message obiente.cloud.superadmin.v1.GetOverviewRequest
 */
export type GetOverviewRequest = Message<"obiente.cloud.superadmin.v1.GetOverviewRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetOverviewRequest.
 * Use `create(GetOverviewRequestSchema)` to create a new message.
 */
export declare const GetOverviewRequestSchema: GenMessage<GetOverviewRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.GetOverviewResponse
 */
export type GetOverviewResponse = Message<"obiente.cloud.superadmin.v1.GetOverviewResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.OverviewCounts counts = 1;
     */
    counts?: OverviewCounts;
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.OrganizationOverview organizations = 2;
     */
    organizations: OrganizationOverview[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuperadminPendingInvite pending_invites = 3;
     */
    pendingInvites: SuperadminPendingInvite[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.DeploymentOverview deployments = 4;
     */
    deployments: DeploymentOverview[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.OrganizationUsage usages = 5;
     */
    usages: OrganizationUsage[];
    /**
     * Git commit hash of the running API
     *
     * @generated from field: optional string api_commit = 6;
     */
    apiCommit?: string;
    /**
     * Git commit hash of the running dashboard
     *
     * @generated from field: optional string dashboard_commit = 7;
     */
    dashboardCommit?: string;
    /**
     * Git commit message of the running API
     *
     * @generated from field: optional string api_commit_message = 8;
     */
    apiCommitMessage?: string;
    /**
     * Git commit message of the running dashboard
     *
     * @generated from field: optional string dashboard_commit_message = 9;
     */
    dashboardCommitMessage?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetOverviewResponse.
 * Use `create(GetOverviewResponseSchema)` to create a new message.
 */
export declare const GetOverviewResponseSchema: GenMessage<GetOverviewResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.OverviewCounts
 */
export type OverviewCounts = Message<"obiente.cloud.superadmin.v1.OverviewCounts"> & {
    /**
     * @generated from field: int64 total_organizations = 1;
     */
    totalOrganizations: bigint;
    /**
     * @generated from field: int64 active_members = 2;
     */
    activeMembers: bigint;
    /**
     * @generated from field: int64 pending_invites = 3;
     */
    pendingInvites: bigint;
    /**
     * @generated from field: int64 total_deployments = 4;
     */
    totalDeployments: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.OverviewCounts.
 * Use `create(OverviewCountsSchema)` to create a new message.
 */
export declare const OverviewCountsSchema: GenMessage<OverviewCounts>;
/**
 * @generated from message obiente.cloud.superadmin.v1.OrganizationOverview
 */
export type OrganizationOverview = Message<"obiente.cloud.superadmin.v1.OrganizationOverview"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string slug = 3;
     */
    slug: string;
    /**
     * @generated from field: optional string domain = 4;
     */
    domain?: string;
    /**
     * @generated from field: string plan = 5;
     */
    plan: string;
    /**
     * @generated from field: string status = 6;
     */
    status: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 7;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: int64 member_count = 8;
     */
    memberCount: bigint;
    /**
     * @generated from field: int64 invite_count = 9;
     */
    inviteCount: bigint;
    /**
     * @generated from field: int64 deployment_count = 10;
     */
    deploymentCount: bigint;
    /**
     * Owner user ID
     *
     * @generated from field: optional string owner_id = 11;
     */
    ownerId?: string;
    /**
     * Owner name or email
     *
     * @generated from field: optional string owner_name = 12;
     */
    ownerName?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.OrganizationOverview.
 * Use `create(OrganizationOverviewSchema)` to create a new message.
 */
export declare const OrganizationOverviewSchema: GenMessage<OrganizationOverview>;
/**
 * @generated from message obiente.cloud.superadmin.v1.SuperadminPendingInvite
 */
export type SuperadminPendingInvite = Message<"obiente.cloud.superadmin.v1.SuperadminPendingInvite"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string email = 3;
     */
    email: string;
    /**
     * @generated from field: string role = 4;
     */
    role: string;
    /**
     * @generated from field: google.protobuf.Timestamp invited_at = 5;
     */
    invitedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminPendingInvite.
 * Use `create(SuperadminPendingInviteSchema)` to create a new message.
 */
export declare const SuperadminPendingInviteSchema: GenMessage<SuperadminPendingInvite>;
/**
 * @generated from message obiente.cloud.superadmin.v1.DeploymentOverview
 */
export type DeploymentOverview = Message<"obiente.cloud.superadmin.v1.DeploymentOverview"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Organization name for display
     *
     * @generated from field: optional string organization_name = 3;
     */
    organizationName?: string;
    /**
     * @generated from field: string name = 4;
     */
    name: string;
    /**
     * @generated from field: obiente.cloud.deployments.v1.Environment environment = 5;
     */
    environment: Environment;
    /**
     * @generated from field: obiente.cloud.deployments.v1.DeploymentStatus status = 6;
     */
    status: DeploymentStatus;
    /**
     * @generated from field: optional string domain = 7;
     */
    domain?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 8;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp last_deployed_at = 9;
     */
    lastDeployedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeploymentOverview.
 * Use `create(DeploymentOverviewSchema)` to create a new message.
 */
export declare const DeploymentOverviewSchema: GenMessage<DeploymentOverview>;
/**
 * @generated from message obiente.cloud.superadmin.v1.OrganizationUsage
 */
export type OrganizationUsage = Message<"obiente.cloud.superadmin.v1.OrganizationUsage"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
    /**
     * @generated from field: string month = 3;
     */
    month: string;
    /**
     * @generated from field: int64 cpu_core_seconds = 4;
     */
    cpuCoreSeconds: bigint;
    /**
     * @generated from field: int64 memory_byte_seconds = 5;
     */
    memoryByteSeconds: bigint;
    /**
     * @generated from field: int64 bandwidth_rx_bytes = 6;
     */
    bandwidthRxBytes: bigint;
    /**
     * @generated from field: int64 bandwidth_tx_bytes = 7;
     */
    bandwidthTxBytes: bigint;
    /**
     * @generated from field: int64 storage_bytes = 8;
     */
    storageBytes: bigint;
    /**
     * @generated from field: int32 deployments_active_peak = 9;
     */
    deploymentsActivePeak: number;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.OrganizationUsage.
 * Use `create(OrganizationUsageSchema)` to create a new message.
 */
export declare const OrganizationUsageSchema: GenMessage<OrganizationUsage>;
/**
 * DNS Query Request
 *
 * @generated from message obiente.cloud.superadmin.v1.QueryDNSRequest
 */
export type QueryDNSRequest = Message<"obiente.cloud.superadmin.v1.QueryDNSRequest"> & {
    /**
     * Domain to query (e.g., deploy-123.my.obiente.cloud)
     *
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * DNS record type (A, AAAA, TXT, etc.) - defaults to A
     *
     * @generated from field: string record_type = 2;
     */
    recordType: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.QueryDNSRequest.
 * Use `create(QueryDNSRequestSchema)` to create a new message.
 */
export declare const QueryDNSRequestSchema: GenMessage<QueryDNSRequest>;
/**
 * DNS Query Response
 *
 * @generated from message obiente.cloud.superadmin.v1.QueryDNSResponse
 */
export type QueryDNSResponse = Message<"obiente.cloud.superadmin.v1.QueryDNSResponse"> & {
    /**
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * @generated from field: string record_type = 2;
     */
    recordType: string;
    /**
     * IP addresses or record values
     *
     * @generated from field: repeated string records = 3;
     */
    records: string[];
    /**
     * Error message if query failed
     *
     * @generated from field: string error = 4;
     */
    error: string;
    /**
     * Time to live in seconds
     *
     * @generated from field: int64 ttl = 5;
     */
    ttl: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.QueryDNSResponse.
 * Use `create(QueryDNSResponseSchema)` to create a new message.
 */
export declare const QueryDNSResponseSchema: GenMessage<QueryDNSResponse>;
/**
 * List DNS Records Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDNSRecordsRequest
 */
export type ListDNSRecordsRequest = Message<"obiente.cloud.superadmin.v1.ListDNSRecordsRequest"> & {
    /**
     * Filter by deployment ID
     *
     * @generated from field: optional string deployment_id = 1;
     */
    deploymentId?: string;
    /**
     * Filter by organization ID
     *
     * @generated from field: optional string organization_id = 2;
     */
    organizationId?: string;
    /**
     * Filter by record type (A, SRV) - empty means all
     *
     * @generated from field: optional string record_type = 3;
     */
    recordType?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDNSRecordsRequest.
 * Use `create(ListDNSRecordsRequestSchema)` to create a new message.
 */
export declare const ListDNSRecordsRequestSchema: GenMessage<ListDNSRecordsRequest>;
/**
 * DNS Record Information
 *
 * @generated from message obiente.cloud.superadmin.v1.DNSRecord
 */
export type DNSRecord = Message<"obiente.cloud.superadmin.v1.DNSRecord"> & {
    /**
     * A or SRV
     *
     * @generated from field: string record_type = 1;
     */
    recordType: string;
    /**
     * Deployment ID (for A records)
     *
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Game Server ID (for SRV records)
     *
     * @generated from field: string game_server_id = 3;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 4;
     */
    organizationId: string;
    /**
     * Deployment name (for A records)
     *
     * @generated from field: string deployment_name = 5;
     */
    deploymentName: string;
    /**
     * Game server name (for SRV records)
     *
     * @generated from field: string game_server_name = 6;
     */
    gameServerName: string;
    /**
     * Full domain (e.g., deploy-123.my.obiente.cloud or _minecraft._tcp.gameserver-123.my.obiente.cloud)
     *
     * @generated from field: string domain = 7;
     */
    domain: string;
    /**
     * Resolved IP addresses (for A records)
     *
     * @generated from field: repeated string ip_addresses = 8;
     */
    ipAddresses: string[];
    /**
     * SRV target hostname/IP (for SRV records)
     *
     * @generated from field: string target = 9;
     */
    target: string;
    /**
     * Port (for SRV records)
     *
     * @generated from field: int32 port = 10;
     */
    port: number;
    /**
     * Region where resource is running
     *
     * @generated from field: string region = 11;
     */
    region: string;
    /**
     * Resource status
     *
     * @generated from field: string status = 12;
     */
    status: string;
    /**
     * @generated from field: google.protobuf.Timestamp last_resolved = 13;
     */
    lastResolved?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DNSRecord.
 * Use `create(DNSRecordSchema)` to create a new message.
 */
export declare const DNSRecordSchema: GenMessage<DNSRecord>;
/**
 * List DNS Records Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDNSRecordsResponse
 */
export type ListDNSRecordsResponse = Message<"obiente.cloud.superadmin.v1.ListDNSRecordsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.DNSRecord records = 1;
     */
    records: DNSRecord[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDNSRecordsResponse.
 * Use `create(ListDNSRecordsResponseSchema)` to create a new message.
 */
export declare const ListDNSRecordsResponseSchema: GenMessage<ListDNSRecordsResponse>;
/**
 * Get DNS Config Request
 *
 * @generated from message obiente.cloud.superadmin.v1.GetDNSConfigRequest
 */
export type GetDNSConfigRequest = Message<"obiente.cloud.superadmin.v1.GetDNSConfigRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetDNSConfigRequest.
 * Use `create(GetDNSConfigRequestSchema)` to create a new message.
 */
export declare const GetDNSConfigRequestSchema: GenMessage<GetDNSConfigRequest>;
/**
 * DNS Configuration
 *
 * @generated from message obiente.cloud.superadmin.v1.DNSConfig
 */
export type DNSConfig = Message<"obiente.cloud.superadmin.v1.DNSConfig"> & {
    /**
     * All configured Traefik IPs
     *
     * @generated from field: repeated string traefik_ips = 1;
     */
    traefikIps: string[];
    /**
     * Traefik IPs grouped by region
     *
     * @generated from field: map<string, obiente.cloud.superadmin.v1.TraefikIPs> traefik_ips_by_region = 2;
     */
    traefikIpsByRegion: {
        [key: string]: TraefikIPs;
    };
    /**
     * DNS server IPs
     *
     * @generated from field: repeated string dns_server_ips = 3;
     */
    dnsServerIps: string[];
    /**
     * DNS server port
     *
     * @generated from field: string dns_port = 4;
     */
    dnsPort: string;
    /**
     * Cache TTL in seconds
     *
     * @generated from field: int64 cache_ttl_seconds = 5;
     */
    cacheTtlSeconds: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DNSConfig.
 * Use `create(DNSConfigSchema)` to create a new message.
 */
export declare const DNSConfigSchema: GenMessage<DNSConfig>;
/**
 * @generated from message obiente.cloud.superadmin.v1.TraefikIPs
 */
export type TraefikIPs = Message<"obiente.cloud.superadmin.v1.TraefikIPs"> & {
    /**
     * @generated from field: string region = 1;
     */
    region: string;
    /**
     * @generated from field: repeated string ips = 2;
     */
    ips: string[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.TraefikIPs.
 * Use `create(TraefikIPsSchema)` to create a new message.
 */
export declare const TraefikIPsSchema: GenMessage<TraefikIPs>;
/**
 * Get DNS Config Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetDNSConfigResponse
 */
export type GetDNSConfigResponse = Message<"obiente.cloud.superadmin.v1.GetDNSConfigResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.DNSConfig config = 1;
     */
    config?: DNSConfig;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetDNSConfigResponse.
 * Use `create(GetDNSConfigResponseSchema)` to create a new message.
 */
export declare const GetDNSConfigResponseSchema: GenMessage<GetDNSConfigResponse>;
/**
 * List Delegated DNS Records Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsRequest
 */
export type ListDelegatedDNSRecordsRequest = Message<"obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsRequest"> & {
    /**
     * Filter by organization ID
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
    /**
     * Filter by API key ID
     *
     * @generated from field: optional string api_key_id = 2;
     */
    apiKeyId?: string;
    /**
     * Filter by record type (A, SRV) - empty means all
     *
     * @generated from field: optional string record_type = 3;
     */
    recordType?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsRequest.
 * Use `create(ListDelegatedDNSRecordsRequestSchema)` to create a new message.
 */
export declare const ListDelegatedDNSRecordsRequestSchema: GenMessage<ListDelegatedDNSRecordsRequest>;
/**
 * Delegated DNS Record Information
 *
 * @generated from message obiente.cloud.superadmin.v1.DelegatedDNSRecord
 */
export type DelegatedDNSRecord = Message<"obiente.cloud.superadmin.v1.DelegatedDNSRecord"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Full domain (e.g., deploy-123.my.obiente.cloud)
     *
     * @generated from field: string domain = 2;
     */
    domain: string;
    /**
     * A or SRV
     *
     * @generated from field: string record_type = 3;
     */
    recordType: string;
    /**
     * Record values (IP addresses for A, SRV strings for SRV)
     *
     * @generated from field: repeated string records = 4;
     */
    records: string[];
    /**
     * URL of the API that pushed this record
     *
     * @generated from field: string source_api = 5;
     */
    sourceApi: string;
    /**
     * ID of the API key that delegated this record
     *
     * @generated from field: string api_key_id = 6;
     */
    apiKeyId: string;
    /**
     * Organization that owns the API key
     *
     * @generated from field: string organization_id = 7;
     */
    organizationId: string;
    /**
     * TTL in seconds
     *
     * @generated from field: int64 ttl = 8;
     */
    ttl: bigint;
    /**
     * When this record expires
     *
     * @generated from field: google.protobuf.Timestamp expires_at = 9;
     */
    expiresAt?: Timestamp;
    /**
     * Last time this record was updated
     *
     * @generated from field: google.protobuf.Timestamp last_updated = 10;
     */
    lastUpdated?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 11;
     */
    createdAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DelegatedDNSRecord.
 * Use `create(DelegatedDNSRecordSchema)` to create a new message.
 */
export declare const DelegatedDNSRecordSchema: GenMessage<DelegatedDNSRecord>;
/**
 * List Delegated DNS Records Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsResponse
 */
export type ListDelegatedDNSRecordsResponse = Message<"obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.DelegatedDNSRecord records = 1;
     */
    records: DelegatedDNSRecord[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDelegatedDNSRecordsResponse.
 * Use `create(ListDelegatedDNSRecordsResponseSchema)` to create a new message.
 */
export declare const ListDelegatedDNSRecordsResponseSchema: GenMessage<ListDelegatedDNSRecordsResponse>;
/**
 * Has Delegated DNS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.HasDelegatedDNSRequest
 */
export type HasDelegatedDNSRequest = Message<"obiente.cloud.superadmin.v1.HasDelegatedDNSRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.HasDelegatedDNSRequest.
 * Use `create(HasDelegatedDNSRequestSchema)` to create a new message.
 */
export declare const HasDelegatedDNSRequestSchema: GenMessage<HasDelegatedDNSRequest>;
/**
 * Has Delegated DNS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.HasDelegatedDNSResponse
 */
export type HasDelegatedDNSResponse = Message<"obiente.cloud.superadmin.v1.HasDelegatedDNSResponse"> & {
    /**
     * Whether the user has delegated DNS
     *
     * @generated from field: bool has_delegated_dns = 1;
     */
    hasDelegatedDns: boolean;
    /**
     * Organization ID that has delegated DNS (if any)
     *
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * API key ID (if any)
     *
     * @generated from field: string api_key_id = 3;
     */
    apiKeyId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.HasDelegatedDNSResponse.
 * Use `create(HasDelegatedDNSResponseSchema)` to create a new message.
 */
export declare const HasDelegatedDNSResponseSchema: GenMessage<HasDelegatedDNSResponse>;
/**
 * Get Pricing Request - public endpoint
 *
 * @generated from message obiente.cloud.superadmin.v1.GetPricingRequest
 */
export type GetPricingRequest = Message<"obiente.cloud.superadmin.v1.GetPricingRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetPricingRequest.
 * Use `create(GetPricingRequestSchema)` to create a new message.
 */
export declare const GetPricingRequestSchema: GenMessage<GetPricingRequest>;
/**
 * Get Pricing Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetPricingResponse
 */
export type GetPricingResponse = Message<"obiente.cloud.superadmin.v1.GetPricingResponse"> & {
    /**
     * @generated from field: double cpu_cost_per_core_second = 1;
     */
    cpuCostPerCoreSecond: number;
    /**
     * @generated from field: double memory_cost_per_byte_second = 2;
     */
    memoryCostPerByteSecond: number;
    /**
     * @generated from field: double bandwidth_cost_per_byte = 3;
     */
    bandwidthCostPerByte: number;
    /**
     * @generated from field: double storage_cost_per_byte_month = 4;
     */
    storageCostPerByteMonth: number;
    /**
     * Human-readable description
     *
     * @generated from field: string pricing_info = 5;
     */
    pricingInfo: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetPricingResponse.
 * Use `create(GetPricingResponseSchema)` to create a new message.
 */
export declare const GetPricingResponseSchema: GenMessage<GetPricingResponse>;
/**
 * Create DNS Delegation API Key Request
 *
 * @generated from message obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyRequest
 */
export type CreateDNSDelegationAPIKeyRequest = Message<"obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyRequest"> & {
    /**
     * Description of who/what this key is for
     *
     * @generated from field: string description = 1;
     */
    description: string;
    /**
     * URL of the API that will use this key (optional)
     *
     * @generated from field: optional string source_api = 2;
     */
    sourceApi?: string;
    /**
     * Organization ID (optional, required for non-superadmins, inferred from user's memberships if not provided)
     *
     * @generated from field: optional string organization_id = 3;
     */
    organizationId?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyRequest.
 * Use `create(CreateDNSDelegationAPIKeyRequestSchema)` to create a new message.
 */
export declare const CreateDNSDelegationAPIKeyRequestSchema: GenMessage<CreateDNSDelegationAPIKeyRequest>;
/**
 * Create DNS Delegation API Key Response
 *
 * @generated from message obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyResponse
 */
export type CreateDNSDelegationAPIKeyResponse = Message<"obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyResponse"> & {
    /**
     * The generated API key (shown only once)
     *
     * @generated from field: string api_key = 1;
     */
    apiKey: string;
    /**
     * Success message
     *
     * @generated from field: string message = 2;
     */
    message: string;
    /**
     * Description of the API key
     *
     * @generated from field: string description = 3;
     */
    description: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateDNSDelegationAPIKeyResponse.
 * Use `create(CreateDNSDelegationAPIKeyResponseSchema)` to create a new message.
 */
export declare const CreateDNSDelegationAPIKeyResponseSchema: GenMessage<CreateDNSDelegationAPIKeyResponse>;
/**
 * Revoke DNS Delegation API Key Request
 *
 * @generated from message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyRequest
 */
export type RevokeDNSDelegationAPIKeyRequest = Message<"obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyRequest"> & {
    /**
     * The API key to revoke
     *
     * @generated from field: string api_key = 1;
     */
    apiKey: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyRequest.
 * Use `create(RevokeDNSDelegationAPIKeyRequestSchema)` to create a new message.
 */
export declare const RevokeDNSDelegationAPIKeyRequestSchema: GenMessage<RevokeDNSDelegationAPIKeyRequest>;
/**
 * Revoke DNS Delegation API Key Response
 *
 * @generated from message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyResponse
 */
export type RevokeDNSDelegationAPIKeyResponse = Message<"obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Success message
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyResponse.
 * Use `create(RevokeDNSDelegationAPIKeyResponseSchema)` to create a new message.
 */
export declare const RevokeDNSDelegationAPIKeyResponseSchema: GenMessage<RevokeDNSDelegationAPIKeyResponse>;
/**
 * Revoke DNS Delegation API Key For Organization Request
 *
 * @generated from message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationRequest
 */
export type RevokeDNSDelegationAPIKeyForOrganizationRequest = Message<"obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationRequest"> & {
    /**
     * Organization ID whose API key to revoke
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationRequest.
 * Use `create(RevokeDNSDelegationAPIKeyForOrganizationRequestSchema)` to create a new message.
 */
export declare const RevokeDNSDelegationAPIKeyForOrganizationRequestSchema: GenMessage<RevokeDNSDelegationAPIKeyForOrganizationRequest>;
/**
 * Revoke DNS Delegation API Key For Organization Response
 *
 * @generated from message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationResponse
 */
export type RevokeDNSDelegationAPIKeyForOrganizationResponse = Message<"obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * Success message
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.RevokeDNSDelegationAPIKeyForOrganizationResponse.
 * Use `create(RevokeDNSDelegationAPIKeyForOrganizationResponseSchema)` to create a new message.
 */
export declare const RevokeDNSDelegationAPIKeyForOrganizationResponseSchema: GenMessage<RevokeDNSDelegationAPIKeyForOrganizationResponse>;
/**
 * List DNS Delegation API Keys Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysRequest
 */
export type ListDNSDelegationAPIKeysRequest = Message<"obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysRequest"> & {
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysRequest.
 * Use `create(ListDNSDelegationAPIKeysRequestSchema)` to create a new message.
 */
export declare const ListDNSDelegationAPIKeysRequestSchema: GenMessage<ListDNSDelegationAPIKeysRequest>;
/**
 * DNS Delegation API Key Info
 *
 * @generated from message obiente.cloud.superadmin.v1.DNSDelegationAPIKeyInfo
 */
export type DNSDelegationAPIKeyInfo = Message<"obiente.cloud.superadmin.v1.DNSDelegationAPIKeyInfo"> & {
    /**
     * API key ID
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Description
     *
     * @generated from field: string description = 2;
     */
    description: string;
    /**
     * Source API URL (if set)
     *
     * @generated from field: string source_api = 3;
     */
    sourceApi: string;
    /**
     * Organization ID (if set)
     *
     * @generated from field: string organization_id = 4;
     */
    organizationId: string;
    /**
     * Whether the key is active
     *
     * @generated from field: bool is_active = 5;
     */
    isActive: boolean;
    /**
     * When the key was created
     *
     * @generated from field: google.protobuf.Timestamp created_at = 6;
     */
    createdAt?: Timestamp;
    /**
     * When the key was revoked (if revoked)
     *
     * @generated from field: google.protobuf.Timestamp revoked_at = 7;
     */
    revokedAt?: Timestamp;
    /**
     * Stripe subscription ID (if linked to subscription)
     *
     * @generated from field: string stripe_subscription_id = 8;
     */
    stripeSubscriptionId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DNSDelegationAPIKeyInfo.
 * Use `create(DNSDelegationAPIKeyInfoSchema)` to create a new message.
 */
export declare const DNSDelegationAPIKeyInfoSchema: GenMessage<DNSDelegationAPIKeyInfo>;
/**
 * List DNS Delegation API Keys Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysResponse
 */
export type ListDNSDelegationAPIKeysResponse = Message<"obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.DNSDelegationAPIKeyInfo api_keys = 1;
     */
    apiKeys: DNSDelegationAPIKeyInfo[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListDNSDelegationAPIKeysResponse.
 * Use `create(ListDNSDelegationAPIKeysResponseSchema)` to create a new message.
 */
export declare const ListDNSDelegationAPIKeysResponseSchema: GenMessage<ListDNSDelegationAPIKeysResponse>;
/**
 * Get Abuse Detection Request
 *
 * @generated from message obiente.cloud.superadmin.v1.GetAbuseDetectionRequest
 */
export type GetAbuseDetectionRequest = Message<"obiente.cloud.superadmin.v1.GetAbuseDetectionRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetAbuseDetectionRequest.
 * Use `create(GetAbuseDetectionRequestSchema)` to create a new message.
 */
export declare const GetAbuseDetectionRequestSchema: GenMessage<GetAbuseDetectionRequest>;
/**
 * Abuse Detection Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetAbuseDetectionResponse
 */
export type GetAbuseDetectionResponse = Message<"obiente.cloud.superadmin.v1.GetAbuseDetectionResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuspiciousOrganization suspicious_organizations = 1;
     */
    suspiciousOrganizations: SuspiciousOrganization[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuspiciousActivity suspicious_activities = 2;
     */
    suspiciousActivities: SuspiciousActivity[];
    /**
     * @generated from field: obiente.cloud.superadmin.v1.AbuseMetrics metrics = 3;
     */
    metrics?: AbuseMetrics;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetAbuseDetectionResponse.
 * Use `create(GetAbuseDetectionResponseSchema)` to create a new message.
 */
export declare const GetAbuseDetectionResponseSchema: GenMessage<GetAbuseDetectionResponse>;
/**
 * Suspicious Organization
 *
 * @generated from message obiente.cloud.superadmin.v1.SuspiciousOrganization
 */
export type SuspiciousOrganization = Message<"obiente.cloud.superadmin.v1.SuspiciousOrganization"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
    /**
     * Why it's flagged
     *
     * @generated from field: string reason = 3;
     */
    reason: string;
    /**
     * 0-100, higher is more suspicious
     *
     * @generated from field: int64 risk_score = 4;
     */
    riskScore: bigint;
    /**
     * Resources created in last 24h
     *
     * @generated from field: int64 created_count_24h = 5;
     */
    createdCount24h: bigint;
    /**
     * Failed deployments in last 24h
     *
     * @generated from field: int64 failed_deployments_24h = 6;
     */
    failedDeployments24h: bigint;
    /**
     * Total credits spent
     *
     * @generated from field: int64 total_credits_spent = 7;
     */
    totalCreditsSpent: bigint;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 8;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp last_activity = 9;
     */
    lastActivity?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuspiciousOrganization.
 * Use `create(SuspiciousOrganizationSchema)` to create a new message.
 */
export declare const SuspiciousOrganizationSchema: GenMessage<SuspiciousOrganization>;
/**
 * Suspicious Activity
 *
 * @generated from message obiente.cloud.superadmin.v1.SuspiciousActivity
 */
export type SuspiciousActivity = Message<"obiente.cloud.superadmin.v1.SuspiciousActivity"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Organization name for display
     *
     * @generated from field: string organization_name = 7;
     */
    organizationName: string;
    /**
     * "rapid_creation", "failed_payments", "unusual_usage", etc.
     *
     * @generated from field: string activity_type = 3;
     */
    activityType: string;
    /**
     * @generated from field: string description = 4;
     */
    description: string;
    /**
     * 0-100
     *
     * @generated from field: int64 severity = 5;
     */
    severity: bigint;
    /**
     * @generated from field: google.protobuf.Timestamp occurred_at = 6;
     */
    occurredAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuspiciousActivity.
 * Use `create(SuspiciousActivitySchema)` to create a new message.
 */
export declare const SuspiciousActivitySchema: GenMessage<SuspiciousActivity>;
/**
 * Abuse Metrics
 *
 * @generated from message obiente.cloud.superadmin.v1.AbuseMetrics
 */
export type AbuseMetrics = Message<"obiente.cloud.superadmin.v1.AbuseMetrics"> & {
    /**
     * @generated from field: int64 total_suspicious_orgs = 1;
     */
    totalSuspiciousOrgs: bigint;
    /**
     * Risk score > 70
     *
     * @generated from field: int64 high_risk_orgs = 2;
     */
    highRiskOrgs: bigint;
    /**
     * Organizations with >10 resources created in 24h
     *
     * @generated from field: int64 rapid_creations_24h = 3;
     */
    rapidCreations24h: bigint;
    /**
     * @generated from field: int64 failed_payment_attempts_24h = 4;
     */
    failedPaymentAttempts24h: bigint;
    /**
     * @generated from field: int64 unusual_usage_spikes_24h = 5;
     */
    unusualUsageSpikes24h: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.AbuseMetrics.
 * Use `create(AbuseMetricsSchema)` to create a new message.
 */
export declare const AbuseMetricsSchema: GenMessage<AbuseMetrics>;
/**
 * Get Income Overview Request
 *
 * @generated from message obiente.cloud.superadmin.v1.GetIncomeOverviewRequest
 */
export type GetIncomeOverviewRequest = Message<"obiente.cloud.superadmin.v1.GetIncomeOverviewRequest"> & {
    /**
     * ISO 8601 date string (optional, defaults to 30 days ago)
     *
     * @generated from field: optional string start_date = 1;
     */
    startDate?: string;
    /**
     * ISO 8601 date string (optional, defaults to now)
     *
     * @generated from field: optional string end_date = 2;
     */
    endDate?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetIncomeOverviewRequest.
 * Use `create(GetIncomeOverviewRequestSchema)` to create a new message.
 */
export declare const GetIncomeOverviewRequestSchema: GenMessage<GetIncomeOverviewRequest>;
/**
 * Income Overview Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetIncomeOverviewResponse
 */
export type GetIncomeOverviewResponse = Message<"obiente.cloud.superadmin.v1.GetIncomeOverviewResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.IncomeSummary summary = 1;
     */
    summary?: IncomeSummary;
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.MonthlyIncome monthly_income = 2;
     */
    monthlyIncome: MonthlyIncome[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.TopCustomer top_customers = 3;
     */
    topCustomers: TopCustomer[];
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.BillingTransaction transactions = 4;
     */
    transactions: BillingTransaction[];
    /**
     * @generated from field: obiente.cloud.superadmin.v1.PaymentMetrics payment_metrics = 5;
     */
    paymentMetrics?: PaymentMetrics;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetIncomeOverviewResponse.
 * Use `create(GetIncomeOverviewResponseSchema)` to create a new message.
 */
export declare const GetIncomeOverviewResponseSchema: GenMessage<GetIncomeOverviewResponse>;
/**
 * Income Summary
 *
 * @generated from message obiente.cloud.superadmin.v1.IncomeSummary
 */
export type IncomeSummary = Message<"obiente.cloud.superadmin.v1.IncomeSummary"> & {
    /**
     * Total revenue in period
     *
     * @generated from field: double total_revenue = 1;
     */
    totalRevenue: number;
    /**
     * Estimated MRR
     *
     * @generated from field: double monthly_recurring_revenue = 2;
     */
    monthlyRecurringRevenue: number;
    /**
     * Average monthly revenue
     *
     * @generated from field: double average_monthly_revenue = 3;
     */
    averageMonthlyRevenue: number;
    /**
     * @generated from field: int64 total_transactions = 4;
     */
    totalTransactions: bigint;
    /**
     * @generated from field: double total_refunds = 5;
     */
    totalRefunds: number;
    /**
     * Revenue minus refunds
     *
     * @generated from field: double net_revenue = 6;
     */
    netRevenue: number;
    /**
     * Estimated monthly income from all orgs based on usage
     *
     * @generated from field: double estimated_monthly_income = 7;
     */
    estimatedMonthlyIncome: number;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.IncomeSummary.
 * Use `create(IncomeSummarySchema)` to create a new message.
 */
export declare const IncomeSummarySchema: GenMessage<IncomeSummary>;
/**
 * Monthly Income
 *
 * @generated from message obiente.cloud.superadmin.v1.MonthlyIncome
 */
export type MonthlyIncome = Message<"obiente.cloud.superadmin.v1.MonthlyIncome"> & {
    /**
     * YYYY-MM format
     *
     * @generated from field: string month = 1;
     */
    month: string;
    /**
     * @generated from field: double revenue = 2;
     */
    revenue: number;
    /**
     * @generated from field: int64 transaction_count = 3;
     */
    transactionCount: bigint;
    /**
     * @generated from field: double refunds = 4;
     */
    refunds: number;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.MonthlyIncome.
 * Use `create(MonthlyIncomeSchema)` to create a new message.
 */
export declare const MonthlyIncomeSchema: GenMessage<MonthlyIncome>;
/**
 * Top Customer
 *
 * @generated from message obiente.cloud.superadmin.v1.TopCustomer
 */
export type TopCustomer = Message<"obiente.cloud.superadmin.v1.TopCustomer"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
    /**
     * @generated from field: double total_revenue = 3;
     */
    totalRevenue: number;
    /**
     * @generated from field: int64 transaction_count = 4;
     */
    transactionCount: bigint;
    /**
     * @generated from field: google.protobuf.Timestamp first_payment = 5;
     */
    firstPayment?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp last_payment = 6;
     */
    lastPayment?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.TopCustomer.
 * Use `create(TopCustomerSchema)` to create a new message.
 */
export declare const TopCustomerSchema: GenMessage<TopCustomer>;
/**
 * Billing Transaction
 *
 * @generated from message obiente.cloud.superadmin.v1.BillingTransaction
 */
export type BillingTransaction = Message<"obiente.cloud.superadmin.v1.BillingTransaction"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 3;
     */
    organizationName: string;
    /**
     * "payment", "refund", "credit_add", "credit_remove"
     *
     * @generated from field: string type = 4;
     */
    type: string;
    /**
     * @generated from field: double amount_cents = 5;
     */
    amountCents: number;
    /**
     * @generated from field: string currency = 6;
     */
    currency: string;
    /**
     * "succeeded", "failed", "pending"
     *
     * @generated from field: string status = 7;
     */
    status: string;
    /**
     * @generated from field: optional string stripe_invoice_id = 8;
     */
    stripeInvoiceId?: string;
    /**
     * @generated from field: optional string stripe_payment_intent_id = 9;
     */
    stripePaymentIntentId?: string;
    /**
     * @generated from field: optional string note = 10;
     */
    note?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 11;
     */
    createdAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.BillingTransaction.
 * Use `create(BillingTransactionSchema)` to create a new message.
 */
export declare const BillingTransactionSchema: GenMessage<BillingTransaction>;
/**
 * Payment Metrics
 *
 * @generated from message obiente.cloud.superadmin.v1.PaymentMetrics
 */
export type PaymentMetrics = Message<"obiente.cloud.superadmin.v1.PaymentMetrics"> & {
    /**
     * Percentage of successful payments
     *
     * @generated from field: double success_rate = 1;
     */
    successRate: number;
    /**
     * @generated from field: int64 successful_payments = 2;
     */
    successfulPayments: bigint;
    /**
     * @generated from field: int64 failed_payments = 3;
     */
    failedPayments: bigint;
    /**
     * @generated from field: int64 pending_payments = 4;
     */
    pendingPayments: bigint;
    /**
     * @generated from field: double average_payment_amount = 5;
     */
    averagePaymentAmount: number;
    /**
     * @generated from field: double largest_payment = 6;
     */
    largestPayment: number;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.PaymentMetrics.
 * Use `create(PaymentMetricsSchema)` to create a new message.
 */
export declare const PaymentMetricsSchema: GenMessage<PaymentMetrics>;
/**
 * List All Invoices Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListAllInvoicesRequest
 */
export type ListAllInvoicesRequest = Message<"obiente.cloud.superadmin.v1.ListAllInvoicesRequest"> & {
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
    /**
     * Filter by status: "draft", "open", "paid", "uncollectible", "void" (optional)
     *
     * @generated from field: optional string status = 2;
     */
    status?: string;
    /**
     * Limit number of results (default: 50, max: 500)
     *
     * @generated from field: optional int32 limit = 3;
     */
    limit?: number;
    /**
     * ISO 8601 date string (optional)
     *
     * @generated from field: optional string start_date = 4;
     */
    startDate?: string;
    /**
     * ISO 8601 date string (optional)
     *
     * @generated from field: optional string end_date = 5;
     */
    endDate?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListAllInvoicesRequest.
 * Use `create(ListAllInvoicesRequestSchema)` to create a new message.
 */
export declare const ListAllInvoicesRequestSchema: GenMessage<ListAllInvoicesRequest>;
/**
 * List All Invoices Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListAllInvoicesResponse
 */
export type ListAllInvoicesResponse = Message<"obiente.cloud.superadmin.v1.ListAllInvoicesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.InvoiceWithOrganization invoices = 1;
     */
    invoices: InvoiceWithOrganization[];
    /**
     * @generated from field: bool has_more = 2;
     */
    hasMore: boolean;
    /**
     * @generated from field: int64 total_count = 3;
     */
    totalCount: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListAllInvoicesResponse.
 * Use `create(ListAllInvoicesResponseSchema)` to create a new message.
 */
export declare const ListAllInvoicesResponseSchema: GenMessage<ListAllInvoicesResponse>;
/**
 * Invoice with Organization Info
 *
 * @generated from message obiente.cloud.superadmin.v1.InvoiceWithOrganization
 */
export type InvoiceWithOrganization = Message<"obiente.cloud.superadmin.v1.InvoiceWithOrganization"> & {
    /**
     * @generated from field: obiente.cloud.billing.v1.Invoice invoice = 1;
     */
    invoice?: Invoice;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 3;
     */
    organizationName: string;
    /**
     * @generated from field: string customer_email = 4;
     */
    customerEmail: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.InvoiceWithOrganization.
 * Use `create(InvoiceWithOrganizationSchema)` to create a new message.
 */
export declare const InvoiceWithOrganizationSchema: GenMessage<InvoiceWithOrganization>;
/**
 * Send Invoice Reminder Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SendInvoiceReminderRequest
 */
export type SendInvoiceReminderRequest = Message<"obiente.cloud.superadmin.v1.SendInvoiceReminderRequest"> & {
    /**
     * Stripe invoice ID
     *
     * @generated from field: string invoice_id = 1;
     */
    invoiceId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SendInvoiceReminderRequest.
 * Use `create(SendInvoiceReminderRequestSchema)` to create a new message.
 */
export declare const SendInvoiceReminderRequestSchema: GenMessage<SendInvoiceReminderRequest>;
/**
 * Send Invoice Reminder Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SendInvoiceReminderResponse
 */
export type SendInvoiceReminderResponse = Message<"obiente.cloud.superadmin.v1.SendInvoiceReminderResponse"> & {
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
 * Describes the message obiente.cloud.superadmin.v1.SendInvoiceReminderResponse.
 * Use `create(SendInvoiceReminderResponseSchema)` to create a new message.
 */
export declare const SendInvoiceReminderResponseSchema: GenMessage<SendInvoiceReminderResponse>;
/**
 * List Plans Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListPlansRequest
 */
export type ListPlansRequest = Message<"obiente.cloud.superadmin.v1.ListPlansRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListPlansRequest.
 * Use `create(ListPlansRequestSchema)` to create a new message.
 */
export declare const ListPlansRequestSchema: GenMessage<ListPlansRequest>;
/**
 * List Plans Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListPlansResponse
 */
export type ListPlansResponse = Message<"obiente.cloud.superadmin.v1.ListPlansResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.Plan plans = 1;
     */
    plans: Plan[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListPlansResponse.
 * Use `create(ListPlansResponseSchema)` to create a new message.
 */
export declare const ListPlansResponseSchema: GenMessage<ListPlansResponse>;
/**
 * Create Plan Request
 *
 * @generated from message obiente.cloud.superadmin.v1.CreatePlanRequest
 */
export type CreatePlanRequest = Message<"obiente.cloud.superadmin.v1.CreatePlanRequest"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: int32 cpu_cores = 2;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 3;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int32 deployments_max = 4;
     */
    deploymentsMax: number;
    /**
     * Maximum VPS instances (0 = unlimited)
     *
     * @generated from field: int32 max_vps_instances = 10;
     */
    maxVpsInstances: number;
    /**
     * @generated from field: int64 bandwidth_bytes_month = 5;
     */
    bandwidthBytesMonth: bigint;
    /**
     * @generated from field: int64 storage_bytes = 6;
     */
    storageBytes: bigint;
    /**
     * Minimum payment in cents to automatically upgrade to this plan
     *
     * @generated from field: int64 minimum_payment_cents = 7;
     */
    minimumPaymentCents: bigint;
    /**
     * Monthly free credits in cents granted to organizations on this plan
     *
     * @generated from field: int64 monthly_free_credits_cents = 8;
     */
    monthlyFreeCreditsCents: bigint;
    /**
     * Number of trial days for Stripe subscriptions (0 = no trial)
     *
     * @generated from field: int32 trial_days = 11;
     */
    trialDays: number;
    /**
     * Optional description of the plan
     *
     * @generated from field: string description = 9;
     */
    description: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreatePlanRequest.
 * Use `create(CreatePlanRequestSchema)` to create a new message.
 */
export declare const CreatePlanRequestSchema: GenMessage<CreatePlanRequest>;
/**
 * Create Plan Response
 *
 * @generated from message obiente.cloud.superadmin.v1.CreatePlanResponse
 */
export type CreatePlanResponse = Message<"obiente.cloud.superadmin.v1.CreatePlanResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.Plan plan = 1;
     */
    plan?: Plan;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreatePlanResponse.
 * Use `create(CreatePlanResponseSchema)` to create a new message.
 */
export declare const CreatePlanResponseSchema: GenMessage<CreatePlanResponse>;
/**
 * Update Plan Request
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdatePlanRequest
 */
export type UpdatePlanRequest = Message<"obiente.cloud.superadmin.v1.UpdatePlanRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: optional string name = 2;
     */
    name?: string;
    /**
     * @generated from field: optional int32 cpu_cores = 3;
     */
    cpuCores?: number;
    /**
     * @generated from field: optional int64 memory_bytes = 4;
     */
    memoryBytes?: bigint;
    /**
     * @generated from field: optional int32 deployments_max = 5;
     */
    deploymentsMax?: number;
    /**
     * Maximum VPS instances (0 = unlimited)
     *
     * @generated from field: optional int32 max_vps_instances = 11;
     */
    maxVpsInstances?: number;
    /**
     * @generated from field: optional int64 bandwidth_bytes_month = 6;
     */
    bandwidthBytesMonth?: bigint;
    /**
     * @generated from field: optional int64 storage_bytes = 7;
     */
    storageBytes?: bigint;
    /**
     * @generated from field: optional int64 minimum_payment_cents = 8;
     */
    minimumPaymentCents?: bigint;
    /**
     * @generated from field: optional int64 monthly_free_credits_cents = 9;
     */
    monthlyFreeCreditsCents?: bigint;
    /**
     * Number of trial days for Stripe subscriptions (0 = no trial)
     *
     * @generated from field: optional int32 trial_days = 12;
     */
    trialDays?: number;
    /**
     * @generated from field: optional string description = 10;
     */
    description?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdatePlanRequest.
 * Use `create(UpdatePlanRequestSchema)` to create a new message.
 */
export declare const UpdatePlanRequestSchema: GenMessage<UpdatePlanRequest>;
/**
 * Update Plan Response
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdatePlanResponse
 */
export type UpdatePlanResponse = Message<"obiente.cloud.superadmin.v1.UpdatePlanResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.Plan plan = 1;
     */
    plan?: Plan;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdatePlanResponse.
 * Use `create(UpdatePlanResponseSchema)` to create a new message.
 */
export declare const UpdatePlanResponseSchema: GenMessage<UpdatePlanResponse>;
/**
 * Delete Plan Request
 *
 * @generated from message obiente.cloud.superadmin.v1.DeletePlanRequest
 */
export type DeletePlanRequest = Message<"obiente.cloud.superadmin.v1.DeletePlanRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeletePlanRequest.
 * Use `create(DeletePlanRequestSchema)` to create a new message.
 */
export declare const DeletePlanRequestSchema: GenMessage<DeletePlanRequest>;
/**
 * Delete Plan Response
 *
 * @generated from message obiente.cloud.superadmin.v1.DeletePlanResponse
 */
export type DeletePlanResponse = Message<"obiente.cloud.superadmin.v1.DeletePlanResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeletePlanResponse.
 * Use `create(DeletePlanResponseSchema)` to create a new message.
 */
export declare const DeletePlanResponseSchema: GenMessage<DeletePlanResponse>;
/**
 * Plan
 *
 * @generated from message obiente.cloud.superadmin.v1.Plan
 */
export type Plan = Message<"obiente.cloud.superadmin.v1.Plan"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: int32 cpu_cores = 3;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 4;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int32 deployments_max = 5;
     */
    deploymentsMax: number;
    /**
     * Maximum VPS instances (0 = unlimited)
     *
     * @generated from field: int32 max_vps_instances = 11;
     */
    maxVpsInstances: number;
    /**
     * @generated from field: int64 bandwidth_bytes_month = 6;
     */
    bandwidthBytesMonth: bigint;
    /**
     * @generated from field: int64 storage_bytes = 7;
     */
    storageBytes: bigint;
    /**
     * Minimum payment in cents to automatically upgrade to this plan
     *
     * @generated from field: int64 minimum_payment_cents = 8;
     */
    minimumPaymentCents: bigint;
    /**
     * Monthly free credits in cents granted to organizations on this plan
     *
     * @generated from field: int64 monthly_free_credits_cents = 9;
     */
    monthlyFreeCreditsCents: bigint;
    /**
     * Number of trial days for Stripe subscriptions (0 = no trial)
     *
     * @generated from field: int32 trial_days = 12;
     */
    trialDays: number;
    /**
     * @generated from field: string description = 10;
     */
    description: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.Plan.
 * Use `create(PlanSchema)` to create a new message.
 */
export declare const PlanSchema: GenMessage<Plan>;
/**
 * Assign Plan To Organization Request
 *
 * @generated from message obiente.cloud.superadmin.v1.AssignPlanToOrganizationRequest
 */
export type AssignPlanToOrganizationRequest = Message<"obiente.cloud.superadmin.v1.AssignPlanToOrganizationRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string plan_id = 2;
     */
    planId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.AssignPlanToOrganizationRequest.
 * Use `create(AssignPlanToOrganizationRequestSchema)` to create a new message.
 */
export declare const AssignPlanToOrganizationRequestSchema: GenMessage<AssignPlanToOrganizationRequest>;
/**
 * Assign Plan To Organization Response
 *
 * @generated from message obiente.cloud.superadmin.v1.AssignPlanToOrganizationResponse
 */
export type AssignPlanToOrganizationResponse = Message<"obiente.cloud.superadmin.v1.AssignPlanToOrganizationResponse"> & {
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
 * Describes the message obiente.cloud.superadmin.v1.AssignPlanToOrganizationResponse.
 * Use `create(AssignPlanToOrganizationResponseSchema)` to create a new message.
 */
export declare const AssignPlanToOrganizationResponseSchema: GenMessage<AssignPlanToOrganizationResponse>;
/**
 * List Users Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListUsersRequest
 */
export type ListUsersRequest = Message<"obiente.cloud.superadmin.v1.ListUsersRequest"> & {
    /**
     * Page number (default: 1)
     *
     * @generated from field: optional int32 page = 1;
     */
    page?: number;
    /**
     * Results per page (default: 50, max: 100)
     *
     * @generated from field: optional int32 per_page = 2;
     */
    perPage?: number;
    /**
     * Search by email, name, or user ID
     *
     * @generated from field: optional string search = 3;
     */
    search?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListUsersRequest.
 * Use `create(ListUsersRequestSchema)` to create a new message.
 */
export declare const ListUsersRequestSchema: GenMessage<ListUsersRequest>;
/**
 * List Users Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListUsersResponse
 */
export type ListUsersResponse = Message<"obiente.cloud.superadmin.v1.ListUsersResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.UserInfo users = 1;
     */
    users: UserInfo[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListUsersResponse.
 * Use `create(ListUsersResponseSchema)` to create a new message.
 */
export declare const ListUsersResponseSchema: GenMessage<ListUsersResponse>;
/**
 * Get User Request
 *
 * @generated from message obiente.cloud.superadmin.v1.GetUserRequest
 */
export type GetUserRequest = Message<"obiente.cloud.superadmin.v1.GetUserRequest"> & {
    /**
     * User ID to fetch
     *
     * @generated from field: string user_id = 1;
     */
    userId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetUserRequest.
 * Use `create(GetUserRequestSchema)` to create a new message.
 */
export declare const GetUserRequestSchema: GenMessage<GetUserRequest>;
/**
 * Get User Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetUserResponse
 */
export type GetUserResponse = Message<"obiente.cloud.superadmin.v1.GetUserResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.UserInfo user = 1;
     */
    user?: UserInfo;
    /**
     * Organizations this user belongs to
     *
     * @generated from field: repeated obiente.cloud.superadmin.v1.UserOrganization organizations = 2;
     */
    organizations: UserOrganization[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetUserResponse.
 * Use `create(GetUserResponseSchema)` to create a new message.
 */
export declare const GetUserResponseSchema: GenMessage<GetUserResponse>;
/**
 * User Info
 *
 * @generated from message obiente.cloud.superadmin.v1.UserInfo
 */
export type UserInfo = Message<"obiente.cloud.superadmin.v1.UserInfo"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string email = 2;
     */
    email: string;
    /**
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * @generated from field: string preferred_username = 4;
     */
    preferredUsername: string;
    /**
     * @generated from field: string locale = 5;
     */
    locale: string;
    /**
     * @generated from field: bool email_verified = 6;
     */
    emailVerified: boolean;
    /**
     * @generated from field: optional string avatar_url = 7;
     */
    avatarUrl?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 8;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 9;
     */
    updatedAt?: Timestamp;
    /**
     * User roles (e.g., "superadmin")
     *
     * @generated from field: repeated string roles = 10;
     */
    roles: string[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UserInfo.
 * Use `create(UserInfoSchema)` to create a new message.
 */
export declare const UserInfoSchema: GenMessage<UserInfo>;
/**
 * User Organization
 *
 * @generated from message obiente.cloud.superadmin.v1.UserOrganization
 */
export type UserOrganization = Message<"obiente.cloud.superadmin.v1.UserOrganization"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
    /**
     * Member role in this organization
     *
     * @generated from field: string role = 3;
     */
    role: string;
    /**
     * "active" or "invited"
     *
     * @generated from field: string status = 4;
     */
    status: string;
    /**
     * @generated from field: google.protobuf.Timestamp joined_at = 5;
     */
    joinedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UserOrganization.
 * Use `create(UserOrganizationSchema)` to create a new message.
 */
export declare const UserOrganizationSchema: GenMessage<UserOrganization>;
/**
 * List All VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListAllVPSRequest
 */
export type ListAllVPSRequest = Message<"obiente.cloud.superadmin.v1.ListAllVPSRequest"> & {
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
    /**
     * Filter by status (optional)
     *
     * @generated from field: optional obiente.cloud.vps.v1.VPSStatus status = 2;
     */
    status?: VPSStatus;
    /**
     * Page number (default: 1)
     *
     * @generated from field: optional int32 page = 3;
     */
    page?: number;
    /**
     * Results per page (default: 50, max: 100)
     *
     * @generated from field: optional int32 per_page = 4;
     */
    perPage?: number;
    /**
     * Search by name, ID, organization ID, region, size
     *
     * @generated from field: optional string search = 5;
     */
    search?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListAllVPSRequest.
 * Use `create(ListAllVPSRequestSchema)` to create a new message.
 */
export declare const ListAllVPSRequestSchema: GenMessage<ListAllVPSRequest>;
/**
 * VPS Overview
 *
 * @generated from message obiente.cloud.superadmin.v1.VPSOverview
 */
export type VPSOverview = Message<"obiente.cloud.superadmin.v1.VPSOverview"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * Organization name (for display)
     *
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.VPSOverview.
 * Use `create(VPSOverviewSchema)` to create a new message.
 */
export declare const VPSOverviewSchema: GenMessage<VPSOverview>;
/**
 * List All VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListAllVPSResponse
 */
export type ListAllVPSResponse = Message<"obiente.cloud.superadmin.v1.ListAllVPSResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.VPSOverview vps_instances = 1;
     */
    vpsInstances: VPSOverview[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListAllVPSResponse.
 * Use `create(ListAllVPSResponseSchema)` to create a new message.
 */
export declare const ListAllVPSResponseSchema: GenMessage<ListAllVPSResponse>;
/**
 * List VPS Sizes Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListVPSSizesRequest
 */
export type ListVPSSizesRequest = Message<"obiente.cloud.superadmin.v1.ListVPSSizesRequest"> & {
    /**
     * Filter by region (optional)
     *
     * @generated from field: optional string region = 1;
     */
    region?: string;
    /**
     * Include unavailable sizes (default: false)
     *
     * @generated from field: optional bool include_unavailable = 2;
     */
    includeUnavailable?: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListVPSSizesRequest.
 * Use `create(ListVPSSizesRequestSchema)` to create a new message.
 */
export declare const ListVPSSizesRequestSchema: GenMessage<ListVPSSizesRequest>;
/**
 * List VPS Sizes Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListVPSSizesResponse
 */
export type ListVPSSizesResponse = Message<"obiente.cloud.superadmin.v1.ListVPSSizesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.common.v1.VPSSize sizes = 1;
     */
    sizes: VPSSize[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListVPSSizesResponse.
 * Use `create(ListVPSSizesResponseSchema)` to create a new message.
 */
export declare const ListVPSSizesResponseSchema: GenMessage<ListVPSSizesResponse>;
/**
 * Create VPS Size Request
 *
 * @generated from message obiente.cloud.superadmin.v1.CreateVPSSizeRequest
 */
export type CreateVPSSizeRequest = Message<"obiente.cloud.superadmin.v1.CreateVPSSizeRequest"> & {
    /**
     * Size ID (e.g., "small", "medium", "custom-1")
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Display name
     *
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * Description
     *
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * Number of CPU cores
     *
     * @generated from field: int32 cpu_cores = 4;
     */
    cpuCores: number;
    /**
     * Memory in bytes
     *
     * @generated from field: int64 memory_bytes = 5;
     */
    memoryBytes: bigint;
    /**
     * Disk space in bytes
     *
     * @generated from field: int64 disk_bytes = 6;
     */
    diskBytes: bigint;
    /**
     * Monthly bandwidth limit (0 = unlimited)
     *
     * @generated from field: int64 bandwidth_bytes_month = 7;
     */
    bandwidthBytesMonth: bigint;
    /**
     * Minimum payment in cents required to create this VPS size (0 = no requirement)
     *
     * @generated from field: int64 minimum_payment_cents = 8;
     */
    minimumPaymentCents: bigint;
    /**
     * Whether this size is available
     *
     * @generated from field: bool available = 9;
     */
    available: boolean;
    /**
     * Region (empty = all regions)
     *
     * @generated from field: string region = 10;
     */
    region: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateVPSSizeRequest.
 * Use `create(CreateVPSSizeRequestSchema)` to create a new message.
 */
export declare const CreateVPSSizeRequestSchema: GenMessage<CreateVPSSizeRequest>;
/**
 * Create VPS Size Response
 *
 * @generated from message obiente.cloud.superadmin.v1.CreateVPSSizeResponse
 */
export type CreateVPSSizeResponse = Message<"obiente.cloud.superadmin.v1.CreateVPSSizeResponse"> & {
    /**
     * @generated from field: obiente.cloud.common.v1.VPSSize size = 1;
     */
    size?: VPSSize;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateVPSSizeResponse.
 * Use `create(CreateVPSSizeResponseSchema)` to create a new message.
 */
export declare const CreateVPSSizeResponseSchema: GenMessage<CreateVPSSizeResponse>;
/**
 * Update VPS Size Request
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdateVPSSizeRequest
 */
export type UpdateVPSSizeRequest = Message<"obiente.cloud.superadmin.v1.UpdateVPSSizeRequest"> & {
    /**
     * Size ID to update
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: optional string name = 2;
     */
    name?: string;
    /**
     * @generated from field: optional string description = 3;
     */
    description?: string;
    /**
     * @generated from field: optional int32 cpu_cores = 4;
     */
    cpuCores?: number;
    /**
     * @generated from field: optional int64 memory_bytes = 5;
     */
    memoryBytes?: bigint;
    /**
     * @generated from field: optional int64 disk_bytes = 6;
     */
    diskBytes?: bigint;
    /**
     * @generated from field: optional int64 bandwidth_bytes_month = 7;
     */
    bandwidthBytesMonth?: bigint;
    /**
     * Minimum payment in cents required to create this VPS size (0 = no requirement)
     *
     * @generated from field: optional int64 minimum_payment_cents = 8;
     */
    minimumPaymentCents?: bigint;
    /**
     * @generated from field: optional bool available = 9;
     */
    available?: boolean;
    /**
     * @generated from field: optional string region = 10;
     */
    region?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateVPSSizeRequest.
 * Use `create(UpdateVPSSizeRequestSchema)` to create a new message.
 */
export declare const UpdateVPSSizeRequestSchema: GenMessage<UpdateVPSSizeRequest>;
/**
 * Update VPS Size Response
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdateVPSSizeResponse
 */
export type UpdateVPSSizeResponse = Message<"obiente.cloud.superadmin.v1.UpdateVPSSizeResponse"> & {
    /**
     * @generated from field: obiente.cloud.common.v1.VPSSize size = 1;
     */
    size?: VPSSize;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateVPSSizeResponse.
 * Use `create(UpdateVPSSizeResponseSchema)` to create a new message.
 */
export declare const UpdateVPSSizeResponseSchema: GenMessage<UpdateVPSSizeResponse>;
/**
 * Delete VPS Size Request
 *
 * @generated from message obiente.cloud.superadmin.v1.DeleteVPSSizeRequest
 */
export type DeleteVPSSizeRequest = Message<"obiente.cloud.superadmin.v1.DeleteVPSSizeRequest"> & {
    /**
     * Size ID to delete
     *
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteVPSSizeRequest.
 * Use `create(DeleteVPSSizeRequestSchema)` to create a new message.
 */
export declare const DeleteVPSSizeRequestSchema: GenMessage<DeleteVPSSizeRequest>;
/**
 * Delete VPS Size Response
 *
 * @generated from message obiente.cloud.superadmin.v1.DeleteVPSSizeResponse
 */
export type DeleteVPSSizeResponse = Message<"obiente.cloud.superadmin.v1.DeleteVPSSizeResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteVPSSizeResponse.
 * Use `create(DeleteVPSSizeResponseSchema)` to create a new message.
 */
export declare const DeleteVPSSizeResponseSchema: GenMessage<DeleteVPSSizeResponse>;
/**
 * Superadmin Get VPS Request (superadmin - bypasses organization checks)
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminGetVPSRequest
 */
export type SuperadminGetVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminGetVPSRequest"> & {
    /**
     * VPS ID to fetch
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminGetVPSRequest.
 * Use `create(SuperadminGetVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminGetVPSRequestSchema: GenMessage<SuperadminGetVPSRequest>;
/**
 * Superadmin Get VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminGetVPSResponse
 */
export type SuperadminGetVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminGetVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * Organization name (for display)
     *
     * @generated from field: string organization_name = 2;
     */
    organizationName: string;
    /**
     * User who created the VPS (if available)
     *
     * @generated from field: optional obiente.cloud.superadmin.v1.UserInfo created_by = 3;
     */
    createdBy?: UserInfo;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminGetVPSResponse.
 * Use `create(SuperadminGetVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminGetVPSResponseSchema: GenMessage<SuperadminGetVPSResponse>;
/**
 * Superadmin Resize VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminResizeVPSRequest
 */
export type SuperadminResizeVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminResizeVPSRequest"> & {
    /**
     * VPS ID to resize
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * New VPS size ID (e.g., "small", "medium", "large"). Use "custom" for custom sizing.
     *
     * @generated from field: string new_size = 2;
     */
    newSize: string;
    /**
     * Whether to grow the disk (default: true)
     *
     * @generated from field: bool grow_disk = 3;
     */
    growDisk: boolean;
    /**
     * Whether to apply cloud-init for disk growth (default: true)
     *
     * @generated from field: bool apply_cloudinit = 4;
     */
    applyCloudinit: boolean;
    /**
     * Custom size options (used when new_size is "custom" or empty)
     *
     * Custom CPU cores (required if using custom size)
     *
     * @generated from field: optional int32 custom_cpu_cores = 5;
     */
    customCpuCores?: number;
    /**
     * Custom memory in bytes (required if using custom size)
     *
     * @generated from field: optional int64 custom_memory_bytes = 6;
     */
    customMemoryBytes?: bigint;
    /**
     * Custom disk in bytes (required if using custom size)
     *
     * @generated from field: optional int64 custom_disk_bytes = 7;
     */
    customDiskBytes?: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminResizeVPSRequest.
 * Use `create(SuperadminResizeVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminResizeVPSRequestSchema: GenMessage<SuperadminResizeVPSRequest>;
/**
 * Superadmin Resize VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminResizeVPSResponse
 */
export type SuperadminResizeVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminResizeVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * Information message about the resize operation
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminResizeVPSResponse.
 * Use `create(SuperadminResizeVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminResizeVPSResponseSchema: GenMessage<SuperadminResizeVPSResponse>;
/**
 * Superadmin Suspend VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminSuspendVPSRequest
 */
export type SuperadminSuspendVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminSuspendVPSRequest"> & {
    /**
     * VPS ID to suspend
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * Optional reason for suspension
     *
     * @generated from field: optional string reason = 2;
     */
    reason?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminSuspendVPSRequest.
 * Use `create(SuperadminSuspendVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminSuspendVPSRequestSchema: GenMessage<SuperadminSuspendVPSRequest>;
/**
 * Superadmin Suspend VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminSuspendVPSResponse
 */
export type SuperadminSuspendVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminSuspendVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminSuspendVPSResponse.
 * Use `create(SuperadminSuspendVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminSuspendVPSResponseSchema: GenMessage<SuperadminSuspendVPSResponse>;
/**
 * Superadmin Unsuspend VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSRequest
 */
export type SuperadminUnsuspendVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSRequest"> & {
    /**
     * VPS ID to unsuspend
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSRequest.
 * Use `create(SuperadminUnsuspendVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminUnsuspendVPSRequestSchema: GenMessage<SuperadminUnsuspendVPSRequest>;
/**
 * Superadmin Unsuspend VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSResponse
 */
export type SuperadminUnsuspendVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminUnsuspendVPSResponse.
 * Use `create(SuperadminUnsuspendVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminUnsuspendVPSResponseSchema: GenMessage<SuperadminUnsuspendVPSResponse>;
/**
 * Superadmin Update VPS CloudInit Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitRequest
 */
export type SuperadminUpdateVPSCloudInitRequest = Message<"obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitRequest"> & {
    /**
     * VPS ID
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * New cloud-init configuration
     *
     * @generated from field: obiente.cloud.vps.v1.CloudInitConfig cloud_init = 2;
     */
    cloudInit?: CloudInitConfig;
    /**
     * Whether to grow disk if cloud-init requires it (default: true)
     *
     * @generated from field: bool grow_disk_if_needed = 3;
     */
    growDiskIfNeeded: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitRequest.
 * Use `create(SuperadminUpdateVPSCloudInitRequestSchema)` to create a new message.
 */
export declare const SuperadminUpdateVPSCloudInitRequestSchema: GenMessage<SuperadminUpdateVPSCloudInitRequest>;
/**
 * Superadmin Update VPS CloudInit Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitResponse
 */
export type SuperadminUpdateVPSCloudInitResponse = Message<"obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminUpdateVPSCloudInitResponse.
 * Use `create(SuperadminUpdateVPSCloudInitResponseSchema)` to create a new message.
 */
export declare const SuperadminUpdateVPSCloudInitResponseSchema: GenMessage<SuperadminUpdateVPSCloudInitResponse>;
/**
 * Superadmin Force Stop VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminForceStopVPSRequest
 */
export type SuperadminForceStopVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminForceStopVPSRequest"> & {
    /**
     * VPS ID to force stop
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminForceStopVPSRequest.
 * Use `create(SuperadminForceStopVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminForceStopVPSRequestSchema: GenMessage<SuperadminForceStopVPSRequest>;
/**
 * Superadmin Force Stop VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminForceStopVPSResponse
 */
export type SuperadminForceStopVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminForceStopVPSResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSInstance vps = 1;
     */
    vps?: VPSInstance;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminForceStopVPSResponse.
 * Use `create(SuperadminForceStopVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminForceStopVPSResponseSchema: GenMessage<SuperadminForceStopVPSResponse>;
/**
 * Superadmin Force Delete VPS Request
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSRequest
 */
export type SuperadminForceDeleteVPSRequest = Message<"obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSRequest"> & {
    /**
     * VPS ID to force delete
     *
     * @generated from field: string vps_id = 1;
     */
    vpsId: string;
    /**
     * If true, perform hard delete (default: false, soft delete)
     *
     * @generated from field: bool hard_delete = 2;
     */
    hardDelete: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSRequest.
 * Use `create(SuperadminForceDeleteVPSRequestSchema)` to create a new message.
 */
export declare const SuperadminForceDeleteVPSRequestSchema: GenMessage<SuperadminForceDeleteVPSRequest>;
/**
 * Superadmin Force Delete VPS Response
 *
 * @generated from message obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSResponse
 */
export type SuperadminForceDeleteVPSResponse = Message<"obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSResponse"> & {
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
 * Describes the message obiente.cloud.superadmin.v1.SuperadminForceDeleteVPSResponse.
 * Use `create(SuperadminForceDeleteVPSResponseSchema)` to create a new message.
 */
export declare const SuperadminForceDeleteVPSResponseSchema: GenMessage<SuperadminForceDeleteVPSResponse>;
/**
 * List Stripe Webhook Events Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListStripeWebhookEventsRequest
 */
export type ListStripeWebhookEventsRequest = Message<"obiente.cloud.superadmin.v1.ListStripeWebhookEventsRequest"> & {
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
    /**
     * Filter by event type (optional, e.g., "invoice.paid")
     *
     * @generated from field: optional string event_type = 2;
     */
    eventType?: string;
    /**
     * Filter by Stripe customer ID (optional)
     *
     * @generated from field: optional string customer_id = 3;
     */
    customerId?: string;
    /**
     * Filter by Stripe subscription ID (optional)
     *
     * @generated from field: optional string subscription_id = 4;
     */
    subscriptionId?: string;
    /**
     * Filter by Stripe invoice ID (optional)
     *
     * @generated from field: optional string invoice_id = 5;
     */
    invoiceId?: string;
    /**
     * Limit number of results (default: 50, max: 500)
     *
     * @generated from field: optional int32 limit = 6;
     */
    limit?: number;
    /**
     * Offset for pagination (default: 0)
     *
     * @generated from field: optional int32 offset = 7;
     */
    offset?: number;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListStripeWebhookEventsRequest.
 * Use `create(ListStripeWebhookEventsRequestSchema)` to create a new message.
 */
export declare const ListStripeWebhookEventsRequestSchema: GenMessage<ListStripeWebhookEventsRequest>;
/**
 * Stripe Webhook Event with organization info
 *
 * @generated from message obiente.cloud.superadmin.v1.StripeWebhookEvent
 */
export type StripeWebhookEvent = Message<"obiente.cloud.superadmin.v1.StripeWebhookEvent"> & {
    /**
     * Stripe event ID (evt_*)
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * Event type (e.g., "invoice.paid")
     *
     * @generated from field: string event_type = 2;
     */
    eventType: string;
    /**
     * When the event was processed
     *
     * @generated from field: google.protobuf.Timestamp processed_at = 3;
     */
    processedAt?: Timestamp;
    /**
     * When the event was created
     *
     * @generated from field: google.protobuf.Timestamp created_at = 4;
     */
    createdAt?: Timestamp;
    /**
     * Organization ID if available
     *
     * @generated from field: optional string organization_id = 5;
     */
    organizationId?: string;
    /**
     * Organization name if available
     *
     * @generated from field: optional string organization_name = 6;
     */
    organizationName?: string;
    /**
     * Stripe customer ID if available
     *
     * @generated from field: optional string customer_id = 7;
     */
    customerId?: string;
    /**
     * Stripe subscription ID if available
     *
     * @generated from field: optional string subscription_id = 8;
     */
    subscriptionId?: string;
    /**
     * Stripe invoice ID if available
     *
     * @generated from field: optional string invoice_id = 9;
     */
    invoiceId?: string;
    /**
     * Stripe checkout session ID if available
     *
     * @generated from field: optional string checkout_session_id = 10;
     */
    checkoutSessionId?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.StripeWebhookEvent.
 * Use `create(StripeWebhookEventSchema)` to create a new message.
 */
export declare const StripeWebhookEventSchema: GenMessage<StripeWebhookEvent>;
/**
 * List Stripe Webhook Events Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListStripeWebhookEventsResponse
 */
export type ListStripeWebhookEventsResponse = Message<"obiente.cloud.superadmin.v1.ListStripeWebhookEventsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.StripeWebhookEvent events = 1;
     */
    events: StripeWebhookEvent[];
    /**
     * Total number of events matching filters
     *
     * @generated from field: int64 total_count = 2;
     */
    totalCount: bigint;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListStripeWebhookEventsResponse.
 * Use `create(ListStripeWebhookEventsResponseSchema)` to create a new message.
 */
export declare const ListStripeWebhookEventsResponseSchema: GenMessage<ListStripeWebhookEventsResponse>;
/**
 * List Nodes Request
 *
 * @generated from message obiente.cloud.superadmin.v1.ListNodesRequest
 */
export type ListNodesRequest = Message<"obiente.cloud.superadmin.v1.ListNodesRequest"> & {
    /**
     * Filter by role: "manager", "worker" (optional)
     *
     * @generated from field: optional string role = 1;
     */
    role?: string;
    /**
     * Filter by availability: "active", "pause", "drain" (optional)
     *
     * @generated from field: optional string availability = 2;
     */
    availability?: string;
    /**
     * Filter by status: "ready", "down" (optional)
     *
     * @generated from field: optional string status = 3;
     */
    status?: string;
    /**
     * Filter by region (optional)
     *
     * @generated from field: optional string region = 4;
     */
    region?: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListNodesRequest.
 * Use `create(ListNodesRequestSchema)` to create a new message.
 */
export declare const ListNodesRequestSchema: GenMessage<ListNodesRequest>;
/**
 * List Nodes Response
 *
 * @generated from message obiente.cloud.superadmin.v1.ListNodesResponse
 */
export type ListNodesResponse = Message<"obiente.cloud.superadmin.v1.ListNodesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.NodeInfo nodes = 1;
     */
    nodes: NodeInfo[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListNodesResponse.
 * Use `create(ListNodesResponseSchema)` to create a new message.
 */
export declare const ListNodesResponseSchema: GenMessage<ListNodesResponse>;
/**
 * Get Node Request
 *
 * @generated from message obiente.cloud.superadmin.v1.GetNodeRequest
 */
export type GetNodeRequest = Message<"obiente.cloud.superadmin.v1.GetNodeRequest"> & {
    /**
     * Node ID to fetch
     *
     * @generated from field: string node_id = 1;
     */
    nodeId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetNodeRequest.
 * Use `create(GetNodeRequestSchema)` to create a new message.
 */
export declare const GetNodeRequestSchema: GenMessage<GetNodeRequest>;
/**
 * Get Node Response
 *
 * @generated from message obiente.cloud.superadmin.v1.GetNodeResponse
 */
export type GetNodeResponse = Message<"obiente.cloud.superadmin.v1.GetNodeResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.NodeInfo node = 1;
     */
    node?: NodeInfo;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetNodeResponse.
 * Use `create(GetNodeResponseSchema)` to create a new message.
 */
export declare const GetNodeResponseSchema: GenMessage<GetNodeResponse>;
/**
 * Update Node Config Request
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdateNodeConfigRequest
 */
export type UpdateNodeConfigRequest = Message<"obiente.cloud.superadmin.v1.UpdateNodeConfigRequest"> & {
    /**
     * Node ID to update
     *
     * @generated from field: string node_id = 1;
     */
    nodeId: string;
    /**
     * Node subdomain identifier (e.g., "node1", "us-east-1")
     *
     * @generated from field: optional string subdomain = 2;
     */
    subdomain?: string;
    /**
     * Enable node-specific domains for this node's microservices
     *
     * @generated from field: optional bool use_node_specific_domains = 3;
     */
    useNodeSpecificDomains?: boolean;
    /**
     * Service domain pattern: "node-service" or "service-node"
     *
     * @generated from field: optional string service_domain_pattern = 4;
     */
    serviceDomainPattern?: string;
    /**
     * Region identifier
     *
     * @generated from field: optional string region = 5;
     */
    region?: string;
    /**
     * Maximum deployments allowed on this node
     *
     * @generated from field: optional int32 max_deployments = 6;
     */
    maxDeployments?: number;
    /**
     * Custom labels (key-value pairs)
     *
     * @generated from field: map<string, string> custom_labels = 7;
     */
    customLabels: {
        [key: string]: string;
    };
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateNodeConfigRequest.
 * Use `create(UpdateNodeConfigRequestSchema)` to create a new message.
 */
export declare const UpdateNodeConfigRequestSchema: GenMessage<UpdateNodeConfigRequest>;
/**
 * Update Node Config Response
 *
 * @generated from message obiente.cloud.superadmin.v1.UpdateNodeConfigResponse
 */
export type UpdateNodeConfigResponse = Message<"obiente.cloud.superadmin.v1.UpdateNodeConfigResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.NodeInfo node = 1;
     */
    node?: NodeInfo;
    /**
     * Success message
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateNodeConfigResponse.
 * Use `create(UpdateNodeConfigResponseSchema)` to create a new message.
 */
export declare const UpdateNodeConfigResponseSchema: GenMessage<UpdateNodeConfigResponse>;
/**
 * Node Info
 *
 * @generated from message obiente.cloud.superadmin.v1.NodeInfo
 */
export type NodeInfo = Message<"obiente.cloud.superadmin.v1.NodeInfo"> & {
    /**
     * Swarm node ID
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string hostname = 2;
     */
    hostname: string;
    /**
     * @generated from field: string ip = 3;
     */
    ip: string;
    /**
     * "manager" or "worker"
     *
     * @generated from field: string role = 4;
     */
    role: string;
    /**
     * "active", "pause", "drain"
     *
     * @generated from field: string availability = 5;
     */
    availability: string;
    /**
     * "ready", "down"
     *
     * @generated from field: string status = 6;
     */
    status: string;
    /**
     * Total CPU cores
     *
     * @generated from field: int32 total_cpu = 7;
     */
    totalCpu: number;
    /**
     * Total memory in bytes
     *
     * @generated from field: int64 total_memory = 8;
     */
    totalMemory: bigint;
    /**
     * Used CPU percentage
     *
     * @generated from field: double used_cpu = 9;
     */
    usedCpu: number;
    /**
     * Used memory in bytes
     *
     * @generated from field: int64 used_memory = 10;
     */
    usedMemory: bigint;
    /**
     * Number of deployments on this node
     *
     * @generated from field: int32 deployment_count = 11;
     */
    deploymentCount: number;
    /**
     * Max deployments allowed
     *
     * @generated from field: int32 max_deployments = 12;
     */
    maxDeployments: number;
    /**
     * Region identifier
     *
     * @generated from field: optional string region = 13;
     */
    region?: string;
    /**
     * Node-specific configuration
     *
     * @generated from field: obiente.cloud.superadmin.v1.NodeConfig config = 14;
     */
    config?: NodeConfig;
    /**
     * @generated from field: google.protobuf.Timestamp last_heartbeat = 15;
     */
    lastHeartbeat?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 16;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 17;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.NodeInfo.
 * Use `create(NodeInfoSchema)` to create a new message.
 */
export declare const NodeInfoSchema: GenMessage<NodeInfo>;
/**
 * Node Configuration (stored in Labels JSON field)
 *
 * @generated from message obiente.cloud.superadmin.v1.NodeConfig
 */
export type NodeConfig = Message<"obiente.cloud.superadmin.v1.NodeConfig"> & {
    /**
     * Node subdomain identifier
     *
     * @generated from field: optional string subdomain = 1;
     */
    subdomain?: string;
    /**
     * Enable node-specific domains
     *
     * @generated from field: optional bool use_node_specific_domains = 2;
     */
    useNodeSpecificDomains?: boolean;
    /**
     * "node-service" or "service-node"
     *
     * @generated from field: optional string service_domain_pattern = 3;
     */
    serviceDomainPattern?: string;
    /**
     * Custom labels
     *
     * @generated from field: map<string, string> custom_labels = 4;
     */
    customLabels: {
        [key: string]: string;
    };
};
/**
 * Describes the message obiente.cloud.superadmin.v1.NodeConfig.
 * Use `create(NodeConfigSchema)` to create a new message.
 */
export declare const NodeConfigSchema: GenMessage<NodeConfig>;
/**
 * Superadmin permissions catalog
 *
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminPermissionsRequest
 */
export type ListSuperadminPermissionsRequest = Message<"obiente.cloud.superadmin.v1.ListSuperadminPermissionsRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminPermissionsRequest.
 * Use `create(ListSuperadminPermissionsRequestSchema)` to create a new message.
 */
export declare const ListSuperadminPermissionsRequestSchema: GenMessage<ListSuperadminPermissionsRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.SuperadminPermissionDefinition
 */
export type SuperadminPermissionDefinition = Message<"obiente.cloud.superadmin.v1.SuperadminPermissionDefinition"> & {
    /**
     * e.g., admin.roles.create
     *
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * human friendly
     *
     * @generated from field: string description = 2;
     */
    description: string;
    /**
     * admin | organization
     *
     * @generated from field: string resource_type = 3;
     */
    resourceType: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminPermissionDefinition.
 * Use `create(SuperadminPermissionDefinitionSchema)` to create a new message.
 */
export declare const SuperadminPermissionDefinitionSchema: GenMessage<SuperadminPermissionDefinition>;
/**
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminPermissionsResponse
 */
export type ListSuperadminPermissionsResponse = Message<"obiente.cloud.superadmin.v1.ListSuperadminPermissionsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuperadminPermissionDefinition permissions = 1;
     */
    permissions: SuperadminPermissionDefinition[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminPermissionsResponse.
 * Use `create(ListSuperadminPermissionsResponseSchema)` to create a new message.
 */
export declare const ListSuperadminPermissionsResponseSchema: GenMessage<ListSuperadminPermissionsResponse>;
/**
 * Get current user's superadmin permissions
 *
 * @generated from message obiente.cloud.superadmin.v1.GetMySuperadminPermissionsRequest
 */
export type GetMySuperadminPermissionsRequest = Message<"obiente.cloud.superadmin.v1.GetMySuperadminPermissionsRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetMySuperadminPermissionsRequest.
 * Use `create(GetMySuperadminPermissionsRequestSchema)` to create a new message.
 */
export declare const GetMySuperadminPermissionsRequestSchema: GenMessage<GetMySuperadminPermissionsRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.GetMySuperadminPermissionsResponse
 */
export type GetMySuperadminPermissionsResponse = Message<"obiente.cloud.superadmin.v1.GetMySuperadminPermissionsResponse"> & {
    /**
     * List of permission IDs the user has (e.g., ["superadmin.overview.read", "superadmin.vps.read"])
     *
     * @generated from field: repeated string permissions = 1;
     */
    permissions: string[];
    /**
     * true if user is a full superadmin (email-based), false if role-based
     *
     * @generated from field: bool is_full_superadmin = 2;
     */
    isFullSuperadmin: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.GetMySuperadminPermissionsResponse.
 * Use `create(GetMySuperadminPermissionsResponseSchema)` to create a new message.
 */
export declare const GetMySuperadminPermissionsResponseSchema: GenMessage<GetMySuperadminPermissionsResponse>;
/**
 * Superadmin role management messages
 *
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminRolesRequest
 */
export type ListSuperadminRolesRequest = Message<"obiente.cloud.superadmin.v1.ListSuperadminRolesRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminRolesRequest.
 * Use `create(ListSuperadminRolesRequestSchema)` to create a new message.
 */
export declare const ListSuperadminRolesRequestSchema: GenMessage<ListSuperadminRolesRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.SuperadminRole
 */
export type SuperadminRole = Message<"obiente.cloud.superadmin.v1.SuperadminRole"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * @generated from field: string permissions_json = 4;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminRole.
 * Use `create(SuperadminRoleSchema)` to create a new message.
 */
export declare const SuperadminRoleSchema: GenMessage<SuperadminRole>;
/**
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminRolesResponse
 */
export type ListSuperadminRolesResponse = Message<"obiente.cloud.superadmin.v1.ListSuperadminRolesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuperadminRole roles = 1;
     */
    roles: SuperadminRole[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminRolesResponse.
 * Use `create(ListSuperadminRolesResponseSchema)` to create a new message.
 */
export declare const ListSuperadminRolesResponseSchema: GenMessage<ListSuperadminRolesResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.CreateSuperadminRoleRequest
 */
export type CreateSuperadminRoleRequest = Message<"obiente.cloud.superadmin.v1.CreateSuperadminRoleRequest"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string description = 2;
     */
    description: string;
    /**
     * @generated from field: string permissions_json = 3;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateSuperadminRoleRequest.
 * Use `create(CreateSuperadminRoleRequestSchema)` to create a new message.
 */
export declare const CreateSuperadminRoleRequestSchema: GenMessage<CreateSuperadminRoleRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.CreateSuperadminRoleResponse
 */
export type CreateSuperadminRoleResponse = Message<"obiente.cloud.superadmin.v1.CreateSuperadminRoleResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.SuperadminRole role = 1;
     */
    role?: SuperadminRole;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateSuperadminRoleResponse.
 * Use `create(CreateSuperadminRoleResponseSchema)` to create a new message.
 */
export declare const CreateSuperadminRoleResponseSchema: GenMessage<CreateSuperadminRoleResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.UpdateSuperadminRoleRequest
 */
export type UpdateSuperadminRoleRequest = Message<"obiente.cloud.superadmin.v1.UpdateSuperadminRoleRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * @generated from field: string permissions_json = 4;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateSuperadminRoleRequest.
 * Use `create(UpdateSuperadminRoleRequestSchema)` to create a new message.
 */
export declare const UpdateSuperadminRoleRequestSchema: GenMessage<UpdateSuperadminRoleRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.UpdateSuperadminRoleResponse
 */
export type UpdateSuperadminRoleResponse = Message<"obiente.cloud.superadmin.v1.UpdateSuperadminRoleResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.SuperadminRole role = 1;
     */
    role?: SuperadminRole;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.UpdateSuperadminRoleResponse.
 * Use `create(UpdateSuperadminRoleResponseSchema)` to create a new message.
 */
export declare const UpdateSuperadminRoleResponseSchema: GenMessage<UpdateSuperadminRoleResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.DeleteSuperadminRoleRequest
 */
export type DeleteSuperadminRoleRequest = Message<"obiente.cloud.superadmin.v1.DeleteSuperadminRoleRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteSuperadminRoleRequest.
 * Use `create(DeleteSuperadminRoleRequestSchema)` to create a new message.
 */
export declare const DeleteSuperadminRoleRequestSchema: GenMessage<DeleteSuperadminRoleRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.DeleteSuperadminRoleResponse
 */
export type DeleteSuperadminRoleResponse = Message<"obiente.cloud.superadmin.v1.DeleteSuperadminRoleResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteSuperadminRoleResponse.
 * Use `create(DeleteSuperadminRoleResponseSchema)` to create a new message.
 */
export declare const DeleteSuperadminRoleResponseSchema: GenMessage<DeleteSuperadminRoleResponse>;
/**
 * Superadmin role binding messages
 *
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsRequest
 */
export type ListSuperadminRoleBindingsRequest = Message<"obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsRequest"> & {};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsRequest.
 * Use `create(ListSuperadminRoleBindingsRequestSchema)` to create a new message.
 */
export declare const ListSuperadminRoleBindingsRequestSchema: GenMessage<ListSuperadminRoleBindingsRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.SuperadminRoleBinding
 */
export type SuperadminRoleBinding = Message<"obiente.cloud.superadmin.v1.SuperadminRoleBinding"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string user_id = 2;
     */
    userId: string;
    /**
     * @generated from field: string role_id = 3;
     */
    roleId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.SuperadminRoleBinding.
 * Use `create(SuperadminRoleBindingSchema)` to create a new message.
 */
export declare const SuperadminRoleBindingSchema: GenMessage<SuperadminRoleBinding>;
/**
 * @generated from message obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsResponse
 */
export type ListSuperadminRoleBindingsResponse = Message<"obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.superadmin.v1.SuperadminRoleBinding bindings = 1;
     */
    bindings: SuperadminRoleBinding[];
};
/**
 * Describes the message obiente.cloud.superadmin.v1.ListSuperadminRoleBindingsResponse.
 * Use `create(ListSuperadminRoleBindingsResponseSchema)` to create a new message.
 */
export declare const ListSuperadminRoleBindingsResponseSchema: GenMessage<ListSuperadminRoleBindingsResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingRequest
 */
export type CreateSuperadminRoleBindingRequest = Message<"obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingRequest"> & {
    /**
     * @generated from field: string user_id = 1;
     */
    userId: string;
    /**
     * @generated from field: string role_id = 2;
     */
    roleId: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingRequest.
 * Use `create(CreateSuperadminRoleBindingRequestSchema)` to create a new message.
 */
export declare const CreateSuperadminRoleBindingRequestSchema: GenMessage<CreateSuperadminRoleBindingRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingResponse
 */
export type CreateSuperadminRoleBindingResponse = Message<"obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingResponse"> & {
    /**
     * @generated from field: obiente.cloud.superadmin.v1.SuperadminRoleBinding binding = 1;
     */
    binding?: SuperadminRoleBinding;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.CreateSuperadminRoleBindingResponse.
 * Use `create(CreateSuperadminRoleBindingResponseSchema)` to create a new message.
 */
export declare const CreateSuperadminRoleBindingResponseSchema: GenMessage<CreateSuperadminRoleBindingResponse>;
/**
 * @generated from message obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingRequest
 */
export type DeleteSuperadminRoleBindingRequest = Message<"obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingRequest.
 * Use `create(DeleteSuperadminRoleBindingRequestSchema)` to create a new message.
 */
export declare const DeleteSuperadminRoleBindingRequestSchema: GenMessage<DeleteSuperadminRoleBindingRequest>;
/**
 * @generated from message obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingResponse
 */
export type DeleteSuperadminRoleBindingResponse = Message<"obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.superadmin.v1.DeleteSuperadminRoleBindingResponse.
 * Use `create(DeleteSuperadminRoleBindingResponseSchema)` to create a new message.
 */
export declare const DeleteSuperadminRoleBindingResponseSchema: GenMessage<DeleteSuperadminRoleBindingResponse>;
/**
 * @generated from service obiente.cloud.superadmin.v1.SuperadminService
 */
export declare const SuperadminService: GenService<{
    /**
     * Returns a system-wide overview for superadmin operators.
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetOverview
     */
    getOverview: {
        methodKind: "unary";
        input: typeof GetOverviewRequestSchema;
        output: typeof GetOverviewResponseSchema;
    };
    /**
     * DNS management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.QueryDNS
     */
    queryDNS: {
        methodKind: "unary";
        input: typeof QueryDNSRequestSchema;
        output: typeof QueryDNSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListDNSRecords
     */
    listDNSRecords: {
        methodKind: "unary";
        input: typeof ListDNSRecordsRequestSchema;
        output: typeof ListDNSRecordsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetDNSConfig
     */
    getDNSConfig: {
        methodKind: "unary";
        input: typeof GetDNSConfigRequestSchema;
        output: typeof GetDNSConfigResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListDelegatedDNSRecords
     */
    listDelegatedDNSRecords: {
        methodKind: "unary";
        input: typeof ListDelegatedDNSRecordsRequestSchema;
        output: typeof ListDelegatedDNSRecordsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.HasDelegatedDNS
     */
    hasDelegatedDNS: {
        methodKind: "unary";
        input: typeof HasDelegatedDNSRequestSchema;
        output: typeof HasDelegatedDNSResponseSchema;
    };
    /**
     * DNS Delegation API Key management
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreateDNSDelegationAPIKey
     */
    createDNSDelegationAPIKey: {
        methodKind: "unary";
        input: typeof CreateDNSDelegationAPIKeyRequestSchema;
        output: typeof CreateDNSDelegationAPIKeyResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListDNSDelegationAPIKeys
     */
    listDNSDelegationAPIKeys: {
        methodKind: "unary";
        input: typeof ListDNSDelegationAPIKeysRequestSchema;
        output: typeof ListDNSDelegationAPIKeysResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.RevokeDNSDelegationAPIKey
     */
    revokeDNSDelegationAPIKey: {
        methodKind: "unary";
        input: typeof RevokeDNSDelegationAPIKeyRequestSchema;
        output: typeof RevokeDNSDelegationAPIKeyResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.RevokeDNSDelegationAPIKeyForOrganization
     */
    revokeDNSDelegationAPIKeyForOrganization: {
        methodKind: "unary";
        input: typeof RevokeDNSDelegationAPIKeyForOrganizationRequestSchema;
        output: typeof RevokeDNSDelegationAPIKeyForOrganizationResponseSchema;
    };
    /**
     * Public pricing endpoint - no authentication required
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetPricing
     */
    getPricing: {
        methodKind: "unary";
        input: typeof GetPricingRequestSchema;
        output: typeof GetPricingResponseSchema;
    };
    /**
     * Abuse detection endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetAbuseDetection
     */
    getAbuseDetection: {
        methodKind: "unary";
        input: typeof GetAbuseDetectionRequestSchema;
        output: typeof GetAbuseDetectionResponseSchema;
    };
    /**
     * Income and billing overview endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetIncomeOverview
     */
    getIncomeOverview: {
        methodKind: "unary";
        input: typeof GetIncomeOverviewRequestSchema;
        output: typeof GetIncomeOverviewResponseSchema;
    };
    /**
     * Invoice management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListAllInvoices
     */
    listAllInvoices: {
        methodKind: "unary";
        input: typeof ListAllInvoicesRequestSchema;
        output: typeof ListAllInvoicesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SendInvoiceReminder
     */
    sendInvoiceReminder: {
        methodKind: "unary";
        input: typeof SendInvoiceReminderRequestSchema;
        output: typeof SendInvoiceReminderResponseSchema;
    };
    /**
     * Plan management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListPlans
     */
    listPlans: {
        methodKind: "unary";
        input: typeof ListPlansRequestSchema;
        output: typeof ListPlansResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreatePlan
     */
    createPlan: {
        methodKind: "unary";
        input: typeof CreatePlanRequestSchema;
        output: typeof CreatePlanResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UpdatePlan
     */
    updatePlan: {
        methodKind: "unary";
        input: typeof UpdatePlanRequestSchema;
        output: typeof UpdatePlanResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.DeletePlan
     */
    deletePlan: {
        methodKind: "unary";
        input: typeof DeletePlanRequestSchema;
        output: typeof DeletePlanResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.AssignPlanToOrganization
     */
    assignPlanToOrganization: {
        methodKind: "unary";
        input: typeof AssignPlanToOrganizationRequestSchema;
        output: typeof AssignPlanToOrganizationResponseSchema;
    };
    /**
     * User management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListUsers
     */
    listUsers: {
        methodKind: "unary";
        input: typeof ListUsersRequestSchema;
        output: typeof ListUsersResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetUser
     */
    getUser: {
        methodKind: "unary";
        input: typeof GetUserRequestSchema;
        output: typeof GetUserResponseSchema;
    };
    /**
     * VPS management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListAllVPS
     */
    listAllVPS: {
        methodKind: "unary";
        input: typeof ListAllVPSRequestSchema;
        output: typeof ListAllVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminGetVPS
     */
    superadminGetVPS: {
        methodKind: "unary";
        input: typeof SuperadminGetVPSRequestSchema;
        output: typeof SuperadminGetVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminResizeVPS
     */
    superadminResizeVPS: {
        methodKind: "unary";
        input: typeof SuperadminResizeVPSRequestSchema;
        output: typeof SuperadminResizeVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminSuspendVPS
     */
    superadminSuspendVPS: {
        methodKind: "unary";
        input: typeof SuperadminSuspendVPSRequestSchema;
        output: typeof SuperadminSuspendVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminUnsuspendVPS
     */
    superadminUnsuspendVPS: {
        methodKind: "unary";
        input: typeof SuperadminUnsuspendVPSRequestSchema;
        output: typeof SuperadminUnsuspendVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminUpdateVPSCloudInit
     */
    superadminUpdateVPSCloudInit: {
        methodKind: "unary";
        input: typeof SuperadminUpdateVPSCloudInitRequestSchema;
        output: typeof SuperadminUpdateVPSCloudInitResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminForceStopVPS
     */
    superadminForceStopVPS: {
        methodKind: "unary";
        input: typeof SuperadminForceStopVPSRequestSchema;
        output: typeof SuperadminForceStopVPSResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.SuperadminForceDeleteVPS
     */
    superadminForceDeleteVPS: {
        methodKind: "unary";
        input: typeof SuperadminForceDeleteVPSRequestSchema;
        output: typeof SuperadminForceDeleteVPSResponseSchema;
    };
    /**
     * VPS size catalog management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListVPSSizes
     */
    listVPSSizes: {
        methodKind: "unary";
        input: typeof ListVPSSizesRequestSchema;
        output: typeof ListVPSSizesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreateVPSSize
     */
    createVPSSize: {
        methodKind: "unary";
        input: typeof CreateVPSSizeRequestSchema;
        output: typeof CreateVPSSizeResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UpdateVPSSize
     */
    updateVPSSize: {
        methodKind: "unary";
        input: typeof UpdateVPSSizeRequestSchema;
        output: typeof UpdateVPSSizeResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.DeleteVPSSize
     */
    deleteVPSSize: {
        methodKind: "unary";
        input: typeof DeleteVPSSizeRequestSchema;
        output: typeof DeleteVPSSizeResponseSchema;
    };
    /**
     * VPS public IP management endpoints
     * VPS public IP management uses the canonical types from the vps package
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListVPSPublicIPs
     */
    listVPSPublicIPs: {
        methodKind: "unary";
        input: typeof ListVPSPublicIPsRequestSchema;
        output: typeof ListVPSPublicIPsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreateVPSPublicIP
     */
    createVPSPublicIP: {
        methodKind: "unary";
        input: typeof CreateVPSPublicIPRequestSchema;
        output: typeof CreateVPSPublicIPResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UpdateVPSPublicIP
     */
    updateVPSPublicIP: {
        methodKind: "unary";
        input: typeof UpdateVPSPublicIPRequestSchema;
        output: typeof UpdateVPSPublicIPResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.DeleteVPSPublicIP
     */
    deleteVPSPublicIP: {
        methodKind: "unary";
        input: typeof DeleteVPSPublicIPRequestSchema;
        output: typeof DeleteVPSPublicIPResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.AssignVPSPublicIP
     */
    assignVPSPublicIP: {
        methodKind: "unary";
        input: typeof AssignVPSPublicIPRequestSchema;
        output: typeof AssignVPSPublicIPResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UnassignVPSPublicIP
     */
    unassignVPSPublicIP: {
        methodKind: "unary";
        input: typeof UnassignVPSPublicIPRequestSchema;
        output: typeof UnassignVPSPublicIPResponseSchema;
    };
    /**
     * DHCP lease management endpoints
     * Get all DHCP leases for an organization (from VPSGatewayService)
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetOrgLeases
     */
    getOrgLeases: {
        methodKind: "unary";
        input: typeof GetOrgLeasesRequestSchema;
        output: typeof GetOrgLeasesResponseSchema;
    };
    /**
     * Stripe webhook events management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListStripeWebhookEvents
     */
    listStripeWebhookEvents: {
        methodKind: "unary";
        input: typeof ListStripeWebhookEventsRequestSchema;
        output: typeof ListStripeWebhookEventsResponseSchema;
    };
    /**
     * Node management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListNodes
     */
    listNodes: {
        methodKind: "unary";
        input: typeof ListNodesRequestSchema;
        output: typeof ListNodesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetNode
     */
    getNode: {
        methodKind: "unary";
        input: typeof GetNodeRequestSchema;
        output: typeof GetNodeResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UpdateNodeConfig
     */
    updateNodeConfig: {
        methodKind: "unary";
        input: typeof UpdateNodeConfigRequestSchema;
        output: typeof UpdateNodeConfigResponseSchema;
    };
    /**
     * Superadmin permissions catalog (only superadmin-only permissions)
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListSuperadminPermissions
     */
    listSuperadminPermissions: {
        methodKind: "unary";
        input: typeof ListSuperadminPermissionsRequestSchema;
        output: typeof ListSuperadminPermissionsResponseSchema;
    };
    /**
     * Get current user's superadmin permissions (from their role bindings)
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.GetMySuperadminPermissions
     */
    getMySuperadminPermissions: {
        methodKind: "unary";
        input: typeof GetMySuperadminPermissionsRequestSchema;
        output: typeof GetMySuperadminPermissionsResponseSchema;
    };
    /**
     * Superadmin role management endpoints (global roles, not organization-scoped)
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListSuperadminRoles
     */
    listSuperadminRoles: {
        methodKind: "unary";
        input: typeof ListSuperadminRolesRequestSchema;
        output: typeof ListSuperadminRolesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreateSuperadminRole
     */
    createSuperadminRole: {
        methodKind: "unary";
        input: typeof CreateSuperadminRoleRequestSchema;
        output: typeof CreateSuperadminRoleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.UpdateSuperadminRole
     */
    updateSuperadminRole: {
        methodKind: "unary";
        input: typeof UpdateSuperadminRoleRequestSchema;
        output: typeof UpdateSuperadminRoleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.DeleteSuperadminRole
     */
    deleteSuperadminRole: {
        methodKind: "unary";
        input: typeof DeleteSuperadminRoleRequestSchema;
        output: typeof DeleteSuperadminRoleResponseSchema;
    };
    /**
     * Superadmin role binding management endpoints
     *
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.ListSuperadminRoleBindings
     */
    listSuperadminRoleBindings: {
        methodKind: "unary";
        input: typeof ListSuperadminRoleBindingsRequestSchema;
        output: typeof ListSuperadminRoleBindingsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.CreateSuperadminRoleBinding
     */
    createSuperadminRoleBinding: {
        methodKind: "unary";
        input: typeof CreateSuperadminRoleBindingRequestSchema;
        output: typeof CreateSuperadminRoleBindingResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.superadmin.v1.SuperadminService.DeleteSuperadminRoleBinding
     */
    deleteSuperadminRoleBinding: {
        methodKind: "unary";
        input: typeof DeleteSuperadminRoleBindingRequestSchema;
        output: typeof DeleteSuperadminRoleBindingResponseSchema;
    };
}>;
