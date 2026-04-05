import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/admin/v1/admin_service.proto.
 */
export declare const file_obiente_cloud_admin_v1_admin_service: GenFile;
/**
 * @generated from message obiente.cloud.admin.v1.UpsertOrgQuotaRequest
 */
export type UpsertOrgQuotaRequest = Message<"obiente.cloud.admin.v1.UpsertOrgQuotaRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * 0 = unlimited
     *
     * @generated from field: int32 deployments_max_override = 2;
     */
    deploymentsMaxOverride: number;
    /**
     * 0 = unlimited
     *
     * @generated from field: int32 cpu_cores_override = 3;
     */
    cpuCoresOverride: number;
    /**
     * 0 = unlimited
     *
     * @generated from field: int64 memory_bytes_override = 4;
     */
    memoryBytesOverride: bigint;
    /**
     * 0 = unlimited
     *
     * @generated from field: int64 bandwidth_bytes_month_override = 5;
     */
    bandwidthBytesMonthOverride: bigint;
    /**
     * 0 = unlimited
     *
     * @generated from field: int64 storage_bytes_override = 6;
     */
    storageBytesOverride: bigint;
};
/**
 * Describes the message obiente.cloud.admin.v1.UpsertOrgQuotaRequest.
 * Use `create(UpsertOrgQuotaRequestSchema)` to create a new message.
 */
export declare const UpsertOrgQuotaRequestSchema: GenMessage<UpsertOrgQuotaRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.UpsertOrgQuotaResponse
 */
export type UpsertOrgQuotaResponse = Message<"obiente.cloud.admin.v1.UpsertOrgQuotaResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.admin.v1.UpsertOrgQuotaResponse.
 * Use `create(UpsertOrgQuotaResponseSchema)` to create a new message.
 */
export declare const UpsertOrgQuotaResponseSchema: GenMessage<UpsertOrgQuotaResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.ListPermissionsRequest
 */
export type ListPermissionsRequest = Message<"obiente.cloud.admin.v1.ListPermissionsRequest"> & {};
/**
 * Describes the message obiente.cloud.admin.v1.ListPermissionsRequest.
 * Use `create(ListPermissionsRequestSchema)` to create a new message.
 */
export declare const ListPermissionsRequestSchema: GenMessage<ListPermissionsRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.PermissionDefinition
 */
export type PermissionDefinition = Message<"obiente.cloud.admin.v1.PermissionDefinition"> & {
    /**
     * e.g., deployments.read
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
     * deployment | environment | admin
     *
     * @generated from field: string resource_type = 3;
     */
    resourceType: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.PermissionDefinition.
 * Use `create(PermissionDefinitionSchema)` to create a new message.
 */
export declare const PermissionDefinitionSchema: GenMessage<PermissionDefinition>;
/**
 * @generated from message obiente.cloud.admin.v1.ListPermissionsResponse
 */
export type ListPermissionsResponse = Message<"obiente.cloud.admin.v1.ListPermissionsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.admin.v1.PermissionDefinition permissions = 1;
     */
    permissions: PermissionDefinition[];
};
/**
 * Describes the message obiente.cloud.admin.v1.ListPermissionsResponse.
 * Use `create(ListPermissionsResponseSchema)` to create a new message.
 */
export declare const ListPermissionsResponseSchema: GenMessage<ListPermissionsResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.ListRolesRequest
 */
export type ListRolesRequest = Message<"obiente.cloud.admin.v1.ListRolesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.ListRolesRequest.
 * Use `create(ListRolesRequestSchema)` to create a new message.
 */
export declare const ListRolesRequestSchema: GenMessage<ListRolesRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.Role
 */
export type Role = Message<"obiente.cloud.admin.v1.Role"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string permissions_json = 3;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.Role.
 * Use `create(RoleSchema)` to create a new message.
 */
export declare const RoleSchema: GenMessage<Role>;
/**
 * @generated from message obiente.cloud.admin.v1.ListRolesResponse
 */
export type ListRolesResponse = Message<"obiente.cloud.admin.v1.ListRolesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.admin.v1.Role roles = 1;
     */
    roles: Role[];
};
/**
 * Describes the message obiente.cloud.admin.v1.ListRolesResponse.
 * Use `create(ListRolesResponseSchema)` to create a new message.
 */
export declare const ListRolesResponseSchema: GenMessage<ListRolesResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.CreateRoleRequest
 */
export type CreateRoleRequest = Message<"obiente.cloud.admin.v1.CreateRoleRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string permissions_json = 3;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.CreateRoleRequest.
 * Use `create(CreateRoleRequestSchema)` to create a new message.
 */
export declare const CreateRoleRequestSchema: GenMessage<CreateRoleRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.CreateRoleResponse
 */
export type CreateRoleResponse = Message<"obiente.cloud.admin.v1.CreateRoleResponse"> & {
    /**
     * @generated from field: obiente.cloud.admin.v1.Role role = 1;
     */
    role?: Role;
};
/**
 * Describes the message obiente.cloud.admin.v1.CreateRoleResponse.
 * Use `create(CreateRoleResponseSchema)` to create a new message.
 */
export declare const CreateRoleResponseSchema: GenMessage<CreateRoleResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.UpdateRoleRequest
 */
export type UpdateRoleRequest = Message<"obiente.cloud.admin.v1.UpdateRoleRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * @generated from field: string permissions_json = 4;
     */
    permissionsJson: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.UpdateRoleRequest.
 * Use `create(UpdateRoleRequestSchema)` to create a new message.
 */
export declare const UpdateRoleRequestSchema: GenMessage<UpdateRoleRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.UpdateRoleResponse
 */
export type UpdateRoleResponse = Message<"obiente.cloud.admin.v1.UpdateRoleResponse"> & {
    /**
     * @generated from field: obiente.cloud.admin.v1.Role role = 1;
     */
    role?: Role;
};
/**
 * Describes the message obiente.cloud.admin.v1.UpdateRoleResponse.
 * Use `create(UpdateRoleResponseSchema)` to create a new message.
 */
export declare const UpdateRoleResponseSchema: GenMessage<UpdateRoleResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.DeleteRoleRequest
 */
export type DeleteRoleRequest = Message<"obiente.cloud.admin.v1.DeleteRoleRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.DeleteRoleRequest.
 * Use `create(DeleteRoleRequestSchema)` to create a new message.
 */
export declare const DeleteRoleRequestSchema: GenMessage<DeleteRoleRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.DeleteRoleResponse
 */
export type DeleteRoleResponse = Message<"obiente.cloud.admin.v1.DeleteRoleResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.admin.v1.DeleteRoleResponse.
 * Use `create(DeleteRoleResponseSchema)` to create a new message.
 */
export declare const DeleteRoleResponseSchema: GenMessage<DeleteRoleResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.ListRoleBindingsRequest
 */
export type ListRoleBindingsRequest = Message<"obiente.cloud.admin.v1.ListRoleBindingsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.ListRoleBindingsRequest.
 * Use `create(ListRoleBindingsRequestSchema)` to create a new message.
 */
export declare const ListRoleBindingsRequestSchema: GenMessage<ListRoleBindingsRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.RoleBinding
 */
export type RoleBinding = Message<"obiente.cloud.admin.v1.RoleBinding"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string user_id = 3;
     */
    userId: string;
    /**
     * @generated from field: string role_id = 4;
     */
    roleId: string;
    /**
     * @generated from field: string resource_type = 5;
     */
    resourceType: string;
    /**
     * @generated from field: string resource_id = 6;
     */
    resourceId: string;
    /**
     * JSON
     *
     * @generated from field: string resource_selector = 7;
     */
    resourceSelector: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.RoleBinding.
 * Use `create(RoleBindingSchema)` to create a new message.
 */
export declare const RoleBindingSchema: GenMessage<RoleBinding>;
/**
 * @generated from message obiente.cloud.admin.v1.ListRoleBindingsResponse
 */
export type ListRoleBindingsResponse = Message<"obiente.cloud.admin.v1.ListRoleBindingsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.admin.v1.RoleBinding bindings = 1;
     */
    bindings: RoleBinding[];
};
/**
 * Describes the message obiente.cloud.admin.v1.ListRoleBindingsResponse.
 * Use `create(ListRoleBindingsResponseSchema)` to create a new message.
 */
export declare const ListRoleBindingsResponseSchema: GenMessage<ListRoleBindingsResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.CreateRoleBindingRequest
 */
export type CreateRoleBindingRequest = Message<"obiente.cloud.admin.v1.CreateRoleBindingRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string user_id = 2;
     */
    userId: string;
    /**
     * @generated from field: string role_id = 3;
     */
    roleId: string;
    /**
     * @generated from field: string resource_type = 4;
     */
    resourceType: string;
    /**
     * @generated from field: string resource_id = 5;
     */
    resourceId: string;
    /**
     * @generated from field: string resource_selector = 6;
     */
    resourceSelector: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.CreateRoleBindingRequest.
 * Use `create(CreateRoleBindingRequestSchema)` to create a new message.
 */
export declare const CreateRoleBindingRequestSchema: GenMessage<CreateRoleBindingRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.CreateRoleBindingResponse
 */
export type CreateRoleBindingResponse = Message<"obiente.cloud.admin.v1.CreateRoleBindingResponse"> & {
    /**
     * @generated from field: obiente.cloud.admin.v1.RoleBinding binding = 1;
     */
    binding?: RoleBinding;
};
/**
 * Describes the message obiente.cloud.admin.v1.CreateRoleBindingResponse.
 * Use `create(CreateRoleBindingResponseSchema)` to create a new message.
 */
export declare const CreateRoleBindingResponseSchema: GenMessage<CreateRoleBindingResponse>;
/**
 * @generated from message obiente.cloud.admin.v1.DeleteRoleBindingRequest
 */
export type DeleteRoleBindingRequest = Message<"obiente.cloud.admin.v1.DeleteRoleBindingRequest"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
};
/**
 * Describes the message obiente.cloud.admin.v1.DeleteRoleBindingRequest.
 * Use `create(DeleteRoleBindingRequestSchema)` to create a new message.
 */
export declare const DeleteRoleBindingRequestSchema: GenMessage<DeleteRoleBindingRequest>;
/**
 * @generated from message obiente.cloud.admin.v1.DeleteRoleBindingResponse
 */
export type DeleteRoleBindingResponse = Message<"obiente.cloud.admin.v1.DeleteRoleBindingResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.admin.v1.DeleteRoleBindingResponse.
 * Use `create(DeleteRoleBindingResponseSchema)` to create a new message.
 */
export declare const DeleteRoleBindingResponseSchema: GenMessage<DeleteRoleBindingResponse>;
/**
 * @generated from service obiente.cloud.admin.v1.AdminService
 */
export declare const AdminService: GenService<{
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.UpsertOrgQuota
     */
    upsertOrgQuota: {
        methodKind: "unary";
        input: typeof UpsertOrgQuotaRequestSchema;
        output: typeof UpsertOrgQuotaResponseSchema;
    };
    /**
     * Permissions catalog for UI (comes from server definitions)
     *
     * @generated from rpc obiente.cloud.admin.v1.AdminService.ListPermissions
     */
    listPermissions: {
        methodKind: "unary";
        input: typeof ListPermissionsRequestSchema;
        output: typeof ListPermissionsResponseSchema;
    };
    /**
     * Roles CRUD
     *
     * @generated from rpc obiente.cloud.admin.v1.AdminService.ListRoles
     */
    listRoles: {
        methodKind: "unary";
        input: typeof ListRolesRequestSchema;
        output: typeof ListRolesResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.CreateRole
     */
    createRole: {
        methodKind: "unary";
        input: typeof CreateRoleRequestSchema;
        output: typeof CreateRoleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.UpdateRole
     */
    updateRole: {
        methodKind: "unary";
        input: typeof UpdateRoleRequestSchema;
        output: typeof UpdateRoleResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.DeleteRole
     */
    deleteRole: {
        methodKind: "unary";
        input: typeof DeleteRoleRequestSchema;
        output: typeof DeleteRoleResponseSchema;
    };
    /**
     * Role bindings CRUD (assign roles to users, scoped to resources)
     *
     * @generated from rpc obiente.cloud.admin.v1.AdminService.ListRoleBindings
     */
    listRoleBindings: {
        methodKind: "unary";
        input: typeof ListRoleBindingsRequestSchema;
        output: typeof ListRoleBindingsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.CreateRoleBinding
     */
    createRoleBinding: {
        methodKind: "unary";
        input: typeof CreateRoleBindingRequestSchema;
        output: typeof CreateRoleBindingResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.admin.v1.AdminService.DeleteRoleBinding
     */
    deleteRoleBinding: {
        methodKind: "unary";
        input: typeof DeleteRoleBindingRequestSchema;
        output: typeof DeleteRoleBindingResponseSchema;
    };
}>;
