import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { User } from "../../auth/v1/auth_service_pb";
import type { Pagination } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/organizations/v1/organization_service.proto.
 */
export declare const file_obiente_cloud_organizations_v1_organization_service: GenFile;
/**
 * @generated from message obiente.cloud.organizations.v1.GetUsageRequest
 */
export type GetUsageRequest = Message<"obiente.cloud.organizations.v1.GetUsageRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: specify a month (YYYY-MM format). Defaults to current month.
     *
     * @generated from field: optional string month = 2;
     */
    month?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetUsageRequest.
 * Use `create(GetUsageRequestSchema)` to create a new message.
 */
export declare const GetUsageRequestSchema: GenMessage<GetUsageRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetUsageResponse
 */
export type GetUsageResponse = Message<"obiente.cloud.organizations.v1.GetUsageResponse"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * YYYY-MM format
     *
     * @generated from field: string month = 2;
     */
    month: string;
    /**
     * @generated from field: obiente.cloud.organizations.v1.UsageMetrics current = 3;
     */
    current?: UsageMetrics;
    /**
     * @generated from field: obiente.cloud.organizations.v1.UsageMetrics estimated_monthly = 4;
     */
    estimatedMonthly?: UsageMetrics;
    /**
     * Quota limits (0 means unlimited)
     *
     * @generated from field: obiente.cloud.organizations.v1.UsageQuota quota = 5;
     */
    quota?: UsageQuota;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetUsageResponse.
 * Use `create(GetUsageResponseSchema)` to create a new message.
 */
export declare const GetUsageResponseSchema: GenMessage<GetUsageResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.UsageMetrics
 */
export type UsageMetrics = Message<"obiente.cloud.organizations.v1.UsageMetrics"> & {
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
     * @generated from field: int64 storage_bytes = 5;
     */
    storageBytes: bigint;
    /**
     * @generated from field: int32 deployments_active_peak = 6;
     */
    deploymentsActivePeak: number;
    /**
     * Estimated cost in cents (e.g., 2347 = $23.47)
     *
     * @generated from field: int64 estimated_cost_cents = 7;
     */
    estimatedCostCents: bigint;
    /**
     * Per-resource cost breakdown in cents (optional)
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
    /**
     * VPS public IP cost (monthly rate prorated)
     *
     * @generated from field: optional int64 public_ip_cost_cents = 12;
     */
    publicIpCostCents?: bigint;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UsageMetrics.
 * Use `create(UsageMetricsSchema)` to create a new message.
 */
export declare const UsageMetricsSchema: GenMessage<UsageMetrics>;
/**
 * @generated from message obiente.cloud.organizations.v1.UsageQuota
 */
export type UsageQuota = Message<"obiente.cloud.organizations.v1.UsageQuota"> & {
    /**
     * @generated from field: int64 cpu_core_seconds_monthly = 1;
     */
    cpuCoreSecondsMonthly: bigint;
    /**
     * @generated from field: int64 memory_byte_seconds_monthly = 2;
     */
    memoryByteSecondsMonthly: bigint;
    /**
     * @generated from field: int64 bandwidth_bytes_monthly = 3;
     */
    bandwidthBytesMonthly: bigint;
    /**
     * @generated from field: int64 storage_bytes = 4;
     */
    storageBytes: bigint;
    /**
     * @generated from field: int32 deployments_max = 5;
     */
    deploymentsMax: number;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UsageQuota.
 * Use `create(UsageQuotaSchema)` to create a new message.
 */
export declare const UsageQuotaSchema: GenMessage<UsageQuota>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListOrganizationsRequest
 */
export type ListOrganizationsRequest = Message<"obiente.cloud.organizations.v1.ListOrganizationsRequest"> & {
    /**
     * @generated from field: int32 page = 1;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 2;
     */
    perPage: number;
    /**
     * If true, only return organizations where the user is a member (even for superadmins)
     * If false or unset, superadmins get all organizations, regular users get their memberships
     *
     * @generated from field: optional bool only_mine = 3;
     */
    onlyMine?: boolean;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListOrganizationsRequest.
 * Use `create(ListOrganizationsRequestSchema)` to create a new message.
 */
export declare const ListOrganizationsRequestSchema: GenMessage<ListOrganizationsRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListOrganizationsResponse
 */
export type ListOrganizationsResponse = Message<"obiente.cloud.organizations.v1.ListOrganizationsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.organizations.v1.Organization organizations = 1;
     */
    organizations: Organization[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListOrganizationsResponse.
 * Use `create(ListOrganizationsResponseSchema)` to create a new message.
 */
export declare const ListOrganizationsResponseSchema: GenMessage<ListOrganizationsResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.CreateOrganizationRequest
 */
export type CreateOrganizationRequest = Message<"obiente.cloud.organizations.v1.CreateOrganizationRequest"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string slug = 2;
     */
    slug: string;
    /**
     * starter, pro, enterprise
     *
     * @generated from field: string plan = 3;
     */
    plan: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.CreateOrganizationRequest.
 * Use `create(CreateOrganizationRequestSchema)` to create a new message.
 */
export declare const CreateOrganizationRequestSchema: GenMessage<CreateOrganizationRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.CreateOrganizationResponse
 */
export type CreateOrganizationResponse = Message<"obiente.cloud.organizations.v1.CreateOrganizationResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
};
/**
 * Describes the message obiente.cloud.organizations.v1.CreateOrganizationResponse.
 * Use `create(CreateOrganizationResponseSchema)` to create a new message.
 */
export declare const CreateOrganizationResponseSchema: GenMessage<CreateOrganizationResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetOrganizationRequest
 */
export type GetOrganizationRequest = Message<"obiente.cloud.organizations.v1.GetOrganizationRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetOrganizationRequest.
 * Use `create(GetOrganizationRequestSchema)` to create a new message.
 */
export declare const GetOrganizationRequestSchema: GenMessage<GetOrganizationRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetOrganizationResponse
 */
export type GetOrganizationResponse = Message<"obiente.cloud.organizations.v1.GetOrganizationResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetOrganizationResponse.
 * Use `create(GetOrganizationResponseSchema)` to create a new message.
 */
export declare const GetOrganizationResponseSchema: GenMessage<GetOrganizationResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.UpdateOrganizationRequest
 */
export type UpdateOrganizationRequest = Message<"obiente.cloud.organizations.v1.UpdateOrganizationRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: optional string name = 2;
     */
    name?: string;
    /**
     * @generated from field: optional string domain = 3;
     */
    domain?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UpdateOrganizationRequest.
 * Use `create(UpdateOrganizationRequestSchema)` to create a new message.
 */
export declare const UpdateOrganizationRequestSchema: GenMessage<UpdateOrganizationRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.UpdateOrganizationResponse
 */
export type UpdateOrganizationResponse = Message<"obiente.cloud.organizations.v1.UpdateOrganizationResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UpdateOrganizationResponse.
 * Use `create(UpdateOrganizationResponseSchema)` to create a new message.
 */
export declare const UpdateOrganizationResponseSchema: GenMessage<UpdateOrganizationResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListMembersRequest
 */
export type ListMembersRequest = Message<"obiente.cloud.organizations.v1.ListMembersRequest"> & {
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
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListMembersRequest.
 * Use `create(ListMembersRequestSchema)` to create a new message.
 */
export declare const ListMembersRequestSchema: GenMessage<ListMembersRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListMembersResponse
 */
export type ListMembersResponse = Message<"obiente.cloud.organizations.v1.ListMembersResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.organizations.v1.OrganizationMember members = 1;
     */
    members: OrganizationMember[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListMembersResponse.
 * Use `create(ListMembersResponseSchema)` to create a new message.
 */
export declare const ListMembersResponseSchema: GenMessage<ListMembersResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.InviteMemberRequest
 */
export type InviteMemberRequest = Message<"obiente.cloud.organizations.v1.InviteMemberRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string email = 2;
     */
    email: string;
    /**
     * admin, member, viewer
     *
     * @generated from field: string role = 3;
     */
    role: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.InviteMemberRequest.
 * Use `create(InviteMemberRequestSchema)` to create a new message.
 */
export declare const InviteMemberRequestSchema: GenMessage<InviteMemberRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.InviteMemberResponse
 */
export type InviteMemberResponse = Message<"obiente.cloud.organizations.v1.InviteMemberResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.OrganizationMember member = 1;
     */
    member?: OrganizationMember;
};
/**
 * Describes the message obiente.cloud.organizations.v1.InviteMemberResponse.
 * Use `create(InviteMemberResponseSchema)` to create a new message.
 */
export declare const InviteMemberResponseSchema: GenMessage<InviteMemberResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.ResendInviteRequest
 */
export type ResendInviteRequest = Message<"obiente.cloud.organizations.v1.ResendInviteRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string member_id = 2;
     */
    memberId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ResendInviteRequest.
 * Use `create(ResendInviteRequestSchema)` to create a new message.
 */
export declare const ResendInviteRequestSchema: GenMessage<ResendInviteRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.ResendInviteResponse
 */
export type ResendInviteResponse = Message<"obiente.cloud.organizations.v1.ResendInviteResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ResendInviteResponse.
 * Use `create(ResendInviteResponseSchema)` to create a new message.
 */
export declare const ResendInviteResponseSchema: GenMessage<ResendInviteResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListMyInvitesRequest
 */
export type ListMyInvitesRequest = Message<"obiente.cloud.organizations.v1.ListMyInvitesRequest"> & {
    /**
     * Optional pagination
     *
     * @generated from field: int32 page = 1;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 2;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListMyInvitesRequest.
 * Use `create(ListMyInvitesRequestSchema)` to create a new message.
 */
export declare const ListMyInvitesRequestSchema: GenMessage<ListMyInvitesRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.ListMyInvitesResponse
 */
export type ListMyInvitesResponse = Message<"obiente.cloud.organizations.v1.ListMyInvitesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.organizations.v1.PendingInvite invites = 1;
     */
    invites: PendingInvite[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.organizations.v1.ListMyInvitesResponse.
 * Use `create(ListMyInvitesResponseSchema)` to create a new message.
 */
export declare const ListMyInvitesResponseSchema: GenMessage<ListMyInvitesResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.PendingInvite
 */
export type PendingInvite = Message<"obiente.cloud.organizations.v1.PendingInvite"> & {
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
     * @generated from field: string role = 4;
     */
    role: string;
    /**
     * @generated from field: google.protobuf.Timestamp invited_at = 5;
     */
    invitedAt?: Timestamp;
    /**
     * Email of the person who sent the invite
     *
     * @generated from field: string inviter_email = 6;
     */
    inviterEmail: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.PendingInvite.
 * Use `create(PendingInviteSchema)` to create a new message.
 */
export declare const PendingInviteSchema: GenMessage<PendingInvite>;
/**
 * @generated from message obiente.cloud.organizations.v1.AcceptInviteRequest
 */
export type AcceptInviteRequest = Message<"obiente.cloud.organizations.v1.AcceptInviteRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string member_id = 2;
     */
    memberId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AcceptInviteRequest.
 * Use `create(AcceptInviteRequestSchema)` to create a new message.
 */
export declare const AcceptInviteRequestSchema: GenMessage<AcceptInviteRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.AcceptInviteResponse
 */
export type AcceptInviteResponse = Message<"obiente.cloud.organizations.v1.AcceptInviteResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.OrganizationMember member = 1;
     */
    member?: OrganizationMember;
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 2;
     */
    organization?: Organization;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AcceptInviteResponse.
 * Use `create(AcceptInviteResponseSchema)` to create a new message.
 */
export declare const AcceptInviteResponseSchema: GenMessage<AcceptInviteResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.DeclineInviteRequest
 */
export type DeclineInviteRequest = Message<"obiente.cloud.organizations.v1.DeclineInviteRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string member_id = 2;
     */
    memberId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.DeclineInviteRequest.
 * Use `create(DeclineInviteRequestSchema)` to create a new message.
 */
export declare const DeclineInviteRequestSchema: GenMessage<DeclineInviteRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.DeclineInviteResponse
 */
export type DeclineInviteResponse = Message<"obiente.cloud.organizations.v1.DeclineInviteResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.organizations.v1.DeclineInviteResponse.
 * Use `create(DeclineInviteResponseSchema)` to create a new message.
 */
export declare const DeclineInviteResponseSchema: GenMessage<DeclineInviteResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.UpdateMemberRequest
 */
export type UpdateMemberRequest = Message<"obiente.cloud.organizations.v1.UpdateMemberRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string member_id = 2;
     */
    memberId: string;
    /**
     * @generated from field: optional string role = 3;
     */
    role?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UpdateMemberRequest.
 * Use `create(UpdateMemberRequestSchema)` to create a new message.
 */
export declare const UpdateMemberRequestSchema: GenMessage<UpdateMemberRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.UpdateMemberResponse
 */
export type UpdateMemberResponse = Message<"obiente.cloud.organizations.v1.UpdateMemberResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.OrganizationMember member = 1;
     */
    member?: OrganizationMember;
};
/**
 * Describes the message obiente.cloud.organizations.v1.UpdateMemberResponse.
 * Use `create(UpdateMemberResponseSchema)` to create a new message.
 */
export declare const UpdateMemberResponseSchema: GenMessage<UpdateMemberResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.RemoveMemberRequest
 */
export type RemoveMemberRequest = Message<"obiente.cloud.organizations.v1.RemoveMemberRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string member_id = 2;
     */
    memberId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.RemoveMemberRequest.
 * Use `create(RemoveMemberRequestSchema)` to create a new message.
 */
export declare const RemoveMemberRequestSchema: GenMessage<RemoveMemberRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.RemoveMemberResponse
 */
export type RemoveMemberResponse = Message<"obiente.cloud.organizations.v1.RemoveMemberResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.organizations.v1.RemoveMemberResponse.
 * Use `create(RemoveMemberResponseSchema)` to create a new message.
 */
export declare const RemoveMemberResponseSchema: GenMessage<RemoveMemberResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.TransferOwnershipRequest
 */
export type TransferOwnershipRequest = Message<"obiente.cloud.organizations.v1.TransferOwnershipRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string new_owner_member_id = 2;
     */
    newOwnerMemberId: string;
    /**
     * Optional role for the previous owner after transfer (defaults to "admin")
     *
     * @generated from field: string fallback_role = 3;
     */
    fallbackRole: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.TransferOwnershipRequest.
 * Use `create(TransferOwnershipRequestSchema)` to create a new message.
 */
export declare const TransferOwnershipRequestSchema: GenMessage<TransferOwnershipRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.TransferOwnershipResponse
 */
export type TransferOwnershipResponse = Message<"obiente.cloud.organizations.v1.TransferOwnershipResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string previous_owner_member_id = 2;
     */
    previousOwnerMemberId: string;
    /**
     * @generated from field: string new_owner_member_id = 3;
     */
    newOwnerMemberId: string;
    /**
     * @generated from field: string fallback_role = 4;
     */
    fallbackRole: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.TransferOwnershipResponse.
 * Use `create(TransferOwnershipResponseSchema)` to create a new message.
 */
export declare const TransferOwnershipResponseSchema: GenMessage<TransferOwnershipResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.Organization
 */
export type Organization = Message<"obiente.cloud.organizations.v1.Organization"> & {
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
     * @generated from field: int32 max_deployments = 8;
     */
    maxDeployments: number;
    /**
     * @generated from field: int32 max_vps_instances = 9;
     */
    maxVpsInstances: number;
    /**
     * @generated from field: int32 max_team_members = 10;
     */
    maxTeamMembers: number;
    /**
     * Credits balance in cents ($0.01 units)
     *
     * @generated from field: int64 credits = 11;
     */
    credits: bigint;
    /**
     * Current plan information (if assigned via plan management)
     *
     * @generated from field: optional obiente.cloud.organizations.v1.PlanInfo plan_info = 12;
     */
    planInfo?: PlanInfo;
    /**
     * Total amount paid in cents (for safety check/auto-upgrade)
     *
     * @generated from field: int64 total_paid_cents = 13;
     */
    totalPaidCents: bigint;
};
/**
 * Describes the message obiente.cloud.organizations.v1.Organization.
 * Use `create(OrganizationSchema)` to create a new message.
 */
export declare const OrganizationSchema: GenMessage<Organization>;
/**
 * @generated from message obiente.cloud.organizations.v1.PlanInfo
 */
export type PlanInfo = Message<"obiente.cloud.organizations.v1.PlanInfo"> & {
    /**
     * @generated from field: string plan_id = 1;
     */
    planId: string;
    /**
     * @generated from field: string plan_name = 2;
     */
    planName: string;
    /**
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * Resource limits from the plan
     *
     * @generated from field: int32 cpu_cores = 4;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 5;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int32 deployments_max = 6;
     */
    deploymentsMax: number;
    /**
     * Maximum VPS instances (0 = unlimited)
     *
     * @generated from field: int32 max_vps_instances = 11;
     */
    maxVpsInstances: number;
    /**
     * @generated from field: int64 bandwidth_bytes_month = 7;
     */
    bandwidthBytesMonth: bigint;
    /**
     * @generated from field: int64 storage_bytes = 8;
     */
    storageBytes: bigint;
    /**
     * Minimum payment required to qualify for this plan (in cents)
     *
     * @generated from field: int64 minimum_payment_cents = 9;
     */
    minimumPaymentCents: bigint;
    /**
     * Monthly free credits in cents granted to organizations on this plan
     *
     * @generated from field: int64 monthly_free_credits_cents = 10;
     */
    monthlyFreeCreditsCents: bigint;
    /**
     * Number of trial days for Stripe subscriptions (0 = no trial)
     *
     * @generated from field: int32 trial_days = 12;
     */
    trialDays: number;
};
/**
 * Describes the message obiente.cloud.organizations.v1.PlanInfo.
 * Use `create(PlanInfoSchema)` to create a new message.
 */
export declare const PlanInfoSchema: GenMessage<PlanInfo>;
/**
 * @generated from message obiente.cloud.organizations.v1.OrganizationMember
 */
export type OrganizationMember = Message<"obiente.cloud.organizations.v1.OrganizationMember"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: obiente.cloud.auth.v1.User user = 2;
     */
    user?: User;
    /**
     * @generated from field: string role = 3;
     */
    role: string;
    /**
     * @generated from field: string status = 4;
     */
    status: string;
    /**
     * @generated from field: google.protobuf.Timestamp joined_at = 5;
     */
    joinedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.organizations.v1.OrganizationMember.
 * Use `create(OrganizationMemberSchema)` to create a new message.
 */
export declare const OrganizationMemberSchema: GenMessage<OrganizationMember>;
/**
 * @generated from message obiente.cloud.organizations.v1.AddCreditsRequest
 */
export type AddCreditsRequest = Message<"obiente.cloud.organizations.v1.AddCreditsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Amount in cents ($0.01 units). Must be positive.
     *
     * @generated from field: int64 amount_cents = 2;
     */
    amountCents: bigint;
    /**
     * Optional note/reason for adding credits
     *
     * @generated from field: optional string note = 3;
     */
    note?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AddCreditsRequest.
 * Use `create(AddCreditsRequestSchema)` to create a new message.
 */
export declare const AddCreditsRequestSchema: GenMessage<AddCreditsRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.AddCreditsResponse
 */
export type AddCreditsResponse = Message<"obiente.cloud.organizations.v1.AddCreditsResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
    /**
     * New credits balance after adding
     *
     * @generated from field: int64 new_balance_cents = 2;
     */
    newBalanceCents: bigint;
    /**
     * Amount added
     *
     * @generated from field: int64 amount_added_cents = 3;
     */
    amountAddedCents: bigint;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AddCreditsResponse.
 * Use `create(AddCreditsResponseSchema)` to create a new message.
 */
export declare const AddCreditsResponseSchema: GenMessage<AddCreditsResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.AdminAddCreditsRequest
 */
export type AdminAddCreditsRequest = Message<"obiente.cloud.organizations.v1.AdminAddCreditsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Amount in cents ($0.01 units). Must be positive.
     *
     * @generated from field: int64 amount_cents = 2;
     */
    amountCents: bigint;
    /**
     * Optional note/reason for adding credits
     *
     * @generated from field: optional string note = 3;
     */
    note?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminAddCreditsRequest.
 * Use `create(AdminAddCreditsRequestSchema)` to create a new message.
 */
export declare const AdminAddCreditsRequestSchema: GenMessage<AdminAddCreditsRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.AdminAddCreditsResponse
 */
export type AdminAddCreditsResponse = Message<"obiente.cloud.organizations.v1.AdminAddCreditsResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
    /**
     * New credits balance after adding
     *
     * @generated from field: int64 new_balance_cents = 2;
     */
    newBalanceCents: bigint;
    /**
     * Amount added
     *
     * @generated from field: int64 amount_added_cents = 3;
     */
    amountAddedCents: bigint;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminAddCreditsResponse.
 * Use `create(AdminAddCreditsResponseSchema)` to create a new message.
 */
export declare const AdminAddCreditsResponseSchema: GenMessage<AdminAddCreditsResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.AdminRemoveCreditsRequest
 */
export type AdminRemoveCreditsRequest = Message<"obiente.cloud.organizations.v1.AdminRemoveCreditsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Amount in cents ($0.01 units). Must be positive.
     *
     * @generated from field: int64 amount_cents = 2;
     */
    amountCents: bigint;
    /**
     * Optional note/reason for removing credits
     *
     * @generated from field: optional string note = 3;
     */
    note?: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminRemoveCreditsRequest.
 * Use `create(AdminRemoveCreditsRequestSchema)` to create a new message.
 */
export declare const AdminRemoveCreditsRequestSchema: GenMessage<AdminRemoveCreditsRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.AdminRemoveCreditsResponse
 */
export type AdminRemoveCreditsResponse = Message<"obiente.cloud.organizations.v1.AdminRemoveCreditsResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
    /**
     * New credits balance after removing
     *
     * @generated from field: int64 new_balance_cents = 2;
     */
    newBalanceCents: bigint;
    /**
     * Amount removed
     *
     * @generated from field: int64 amount_removed_cents = 3;
     */
    amountRemovedCents: bigint;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminRemoveCreditsResponse.
 * Use `create(AdminRemoveCreditsResponseSchema)` to create a new message.
 */
export declare const AdminRemoveCreditsResponseSchema: GenMessage<AdminRemoveCreditsResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetCreditLogRequest
 */
export type GetCreditLogRequest = Message<"obiente.cloud.organizations.v1.GetCreditLogRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Pagination
     *
     * @generated from field: int32 page = 2;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 3;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetCreditLogRequest.
 * Use `create(GetCreditLogRequestSchema)` to create a new message.
 */
export declare const GetCreditLogRequestSchema: GenMessage<GetCreditLogRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetCreditLogResponse
 */
export type GetCreditLogResponse = Message<"obiente.cloud.organizations.v1.GetCreditLogResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.organizations.v1.CreditTransaction transactions = 1;
     */
    transactions: CreditTransaction[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetCreditLogResponse.
 * Use `create(GetCreditLogResponseSchema)` to create a new message.
 */
export declare const GetCreditLogResponseSchema: GenMessage<GetCreditLogResponse>;
/**
 * @generated from message obiente.cloud.organizations.v1.CreditTransaction
 */
export type CreditTransaction = Message<"obiente.cloud.organizations.v1.CreditTransaction"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Positive for additions, negative for removals (in cents)
     *
     * @generated from field: int64 amount_cents = 3;
     */
    amountCents: bigint;
    /**
     * Credit balance after this transaction
     *
     * @generated from field: int64 balance_after = 4;
     */
    balanceAfter: bigint;
    /**
     * Transaction type: "payment", "admin_add", "admin_remove", "usage", "refund", etc.
     *
     * @generated from field: string type = 5;
     */
    type: string;
    /**
     * Source: "stripe", "admin", "system", etc.
     *
     * @generated from field: string source = 6;
     */
    source: string;
    /**
     * Optional note/reason
     *
     * @generated from field: optional string note = 7;
     */
    note?: string;
    /**
     * User ID who initiated (null for system/automatic)
     *
     * @generated from field: optional string created_by = 8;
     */
    createdBy?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 9;
     */
    createdAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.organizations.v1.CreditTransaction.
 * Use `create(CreditTransactionSchema)` to create a new message.
 */
export declare const CreditTransactionSchema: GenMessage<CreditTransaction>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetMyPermissionsRequest
 */
export type GetMyPermissionsRequest = Message<"obiente.cloud.organizations.v1.GetMyPermissionsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetMyPermissionsRequest.
 * Use `create(GetMyPermissionsRequestSchema)` to create a new message.
 */
export declare const GetMyPermissionsRequestSchema: GenMessage<GetMyPermissionsRequest>;
/**
 * @generated from message obiente.cloud.organizations.v1.GetMyPermissionsResponse
 */
export type GetMyPermissionsResponse = Message<"obiente.cloud.organizations.v1.GetMyPermissionsResponse"> & {
    /**
     * @generated from field: repeated string permissions = 1;
     */
    permissions: string[];
};
/**
 * Describes the message obiente.cloud.organizations.v1.GetMyPermissionsResponse.
 * Use `create(GetMyPermissionsResponseSchema)` to create a new message.
 */
export declare const GetMyPermissionsResponseSchema: GenMessage<GetMyPermissionsResponse>;
/**
 * Request to set the active plan for an organization (superadmin only)
 *
 * @generated from message obiente.cloud.organizations.v1.AdminSetPlanRequest
 */
export type AdminSetPlanRequest = Message<"obiente.cloud.organizations.v1.AdminSetPlanRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * The plan to assign (must match OrganizationPlan.ID)
     *
     * @generated from field: string plan_id = 2;
     */
    planId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminSetPlanRequest.
 * Use `create(AdminSetPlanRequestSchema)` to create a new message.
 */
export declare const AdminSetPlanRequestSchema: GenMessage<AdminSetPlanRequest>;
/**
 * Response for AdminSetPlan
 *
 * @generated from message obiente.cloud.organizations.v1.AdminSetPlanResponse
 */
export type AdminSetPlanResponse = Message<"obiente.cloud.organizations.v1.AdminSetPlanResponse"> & {
    /**
     * @generated from field: obiente.cloud.organizations.v1.Organization organization = 1;
     */
    organization?: Organization;
    /**
     * @generated from field: string plan_id = 2;
     */
    planId: string;
};
/**
 * Describes the message obiente.cloud.organizations.v1.AdminSetPlanResponse.
 * Use `create(AdminSetPlanResponseSchema)` to create a new message.
 */
export declare const AdminSetPlanResponseSchema: GenMessage<AdminSetPlanResponse>;
/**
 * @generated from service obiente.cloud.organizations.v1.OrganizationService
 */
export declare const OrganizationService: GenService<{
    /**
     * Admin: Set the active plan for an organization (superadmin only)
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.AdminSetPlan
     */
    adminSetPlan: {
        methodKind: "unary";
        input: typeof AdminSetPlanRequestSchema;
        output: typeof AdminSetPlanResponseSchema;
    };
    /**
     * List user's organizations
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.ListOrganizations
     */
    listOrganizations: {
        methodKind: "unary";
        input: typeof ListOrganizationsRequestSchema;
        output: typeof ListOrganizationsResponseSchema;
    };
    /**
     * Create new organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.CreateOrganization
     */
    createOrganization: {
        methodKind: "unary";
        input: typeof CreateOrganizationRequestSchema;
        output: typeof CreateOrganizationResponseSchema;
    };
    /**
     * Get organization details
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.GetOrganization
     */
    getOrganization: {
        methodKind: "unary";
        input: typeof GetOrganizationRequestSchema;
        output: typeof GetOrganizationResponseSchema;
    };
    /**
     * Update organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.UpdateOrganization
     */
    updateOrganization: {
        methodKind: "unary";
        input: typeof UpdateOrganizationRequestSchema;
        output: typeof UpdateOrganizationResponseSchema;
    };
    /**
     * List organization members
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.ListMembers
     */
    listMembers: {
        methodKind: "unary";
        input: typeof ListMembersRequestSchema;
        output: typeof ListMembersResponseSchema;
    };
    /**
     * Invite user to organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.InviteMember
     */
    inviteMember: {
        methodKind: "unary";
        input: typeof InviteMemberRequestSchema;
        output: typeof InviteMemberResponseSchema;
    };
    /**
     * Resend invitation email to a pending member
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.ResendInvite
     */
    resendInvite: {
        methodKind: "unary";
        input: typeof ResendInviteRequestSchema;
        output: typeof ResendInviteResponseSchema;
    };
    /**
     * List invites sent to the current user
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.ListMyInvites
     */
    listMyInvites: {
        methodKind: "unary";
        input: typeof ListMyInvitesRequestSchema;
        output: typeof ListMyInvitesResponseSchema;
    };
    /**
     * Accept an invitation to join an organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.AcceptInvite
     */
    acceptInvite: {
        methodKind: "unary";
        input: typeof AcceptInviteRequestSchema;
        output: typeof AcceptInviteResponseSchema;
    };
    /**
     * Decline an invitation to join an organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.DeclineInvite
     */
    declineInvite: {
        methodKind: "unary";
        input: typeof DeclineInviteRequestSchema;
        output: typeof DeclineInviteResponseSchema;
    };
    /**
     * Update member role/permissions
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.UpdateMember
     */
    updateMember: {
        methodKind: "unary";
        input: typeof UpdateMemberRequestSchema;
        output: typeof UpdateMemberResponseSchema;
    };
    /**
     * Remove member from organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.RemoveMember
     */
    removeMember: {
        methodKind: "unary";
        input: typeof RemoveMemberRequestSchema;
        output: typeof RemoveMemberResponseSchema;
    };
    /**
     * Transfer organization ownership to another member
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.TransferOwnership
     */
    transferOwnership: {
        methodKind: "unary";
        input: typeof TransferOwnershipRequestSchema;
        output: typeof TransferOwnershipResponseSchema;
    };
    /**
     * Get current usage and billing information for the organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.GetUsage
     */
    getUsage: {
        methodKind: "unary";
        input: typeof GetUsageRequestSchema;
        output: typeof GetUsageResponseSchema;
    };
    /**
     * Add credits to organization (for regular users/owners)
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.AddCredits
     */
    addCredits: {
        methodKind: "unary";
        input: typeof AddCreditsRequestSchema;
        output: typeof AddCreditsResponseSchema;
    };
    /**
     * Admin: Add credits to any organization (superadmin only)
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.AdminAddCredits
     */
    adminAddCredits: {
        methodKind: "unary";
        input: typeof AdminAddCreditsRequestSchema;
        output: typeof AdminAddCreditsResponseSchema;
    };
    /**
     * Admin: Remove credits from any organization (superadmin only)
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.AdminRemoveCredits
     */
    adminRemoveCredits: {
        methodKind: "unary";
        input: typeof AdminRemoveCreditsRequestSchema;
        output: typeof AdminRemoveCreditsResponseSchema;
    };
    /**
     * Get credit transaction history for an organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.GetCreditLog
     */
    getCreditLog: {
        methodKind: "unary";
        input: typeof GetCreditLogRequestSchema;
        output: typeof GetCreditLogResponseSchema;
    };
    /**
     * Get current user's permissions for an organization
     *
     * @generated from rpc obiente.cloud.organizations.v1.OrganizationService.GetMyPermissions
     */
    getMyPermissions: {
        methodKind: "unary";
        input: typeof GetMyPermissionsRequestSchema;
        output: typeof GetMyPermissionsResponseSchema;
    };
}>;
