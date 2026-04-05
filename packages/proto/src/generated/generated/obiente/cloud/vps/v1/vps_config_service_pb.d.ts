import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { CloudInitConfig } from "./vps_service_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/vps/v1/vps_config_service.proto.
 */
export declare const file_obiente_cloud_vps_v1_vps_config_service: GenFile;
/**
 * GetCloudInitConfigRequest requests the cloud-init configuration for a VPS
 *
 * @generated from message obiente.cloud.vps.v1.GetCloudInitConfigRequest
 */
export type GetCloudInitConfigRequest = Message<"obiente.cloud.vps.v1.GetCloudInitConfigRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.GetCloudInitConfigRequest.
 * Use `create(GetCloudInitConfigRequestSchema)` to create a new message.
 */
export declare const GetCloudInitConfigRequestSchema: GenMessage<GetCloudInitConfigRequest>;
/**
 * GetCloudInitConfigResponse returns the cloud-init configuration
 *
 * @generated from message obiente.cloud.vps.v1.GetCloudInitConfigResponse
 */
export type GetCloudInitConfigResponse = Message<"obiente.cloud.vps.v1.GetCloudInitConfigResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.CloudInitConfig cloud_init = 1;
     */
    cloudInit?: CloudInitConfig;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetCloudInitConfigResponse.
 * Use `create(GetCloudInitConfigResponseSchema)` to create a new message.
 */
export declare const GetCloudInitConfigResponseSchema: GenMessage<GetCloudInitConfigResponse>;
/**
 * GetCloudInitUserDataRequest requests the actual generated cloud-init userData
 *
 * @generated from message obiente.cloud.vps.v1.GetCloudInitUserDataRequest
 */
export type GetCloudInitUserDataRequest = Message<"obiente.cloud.vps.v1.GetCloudInitUserDataRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.GetCloudInitUserDataRequest.
 * Use `create(GetCloudInitUserDataRequestSchema)` to create a new message.
 */
export declare const GetCloudInitUserDataRequestSchema: GenMessage<GetCloudInitUserDataRequest>;
/**
 * GetCloudInitUserDataResponse returns the actual generated cloud-init userData
 * This includes bastion and terminal keys that are dynamically added
 *
 * @generated from message obiente.cloud.vps.v1.GetCloudInitUserDataResponse
 */
export type GetCloudInitUserDataResponse = Message<"obiente.cloud.vps.v1.GetCloudInitUserDataResponse"> & {
    /**
     * The actual generated cloud-init YAML
     *
     * @generated from field: string user_data = 1;
     */
    userData: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetCloudInitUserDataResponse.
 * Use `create(GetCloudInitUserDataResponseSchema)` to create a new message.
 */
export declare const GetCloudInitUserDataResponseSchema: GenMessage<GetCloudInitUserDataResponse>;
/**
 * UpdateCloudInitConfigRequest updates the cloud-init configuration
 *
 * @generated from message obiente.cloud.vps.v1.UpdateCloudInitConfigRequest
 */
export type UpdateCloudInitConfigRequest = Message<"obiente.cloud.vps.v1.UpdateCloudInitConfigRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: obiente.cloud.vps.v1.CloudInitConfig cloud_init = 3;
     */
    cloudInit?: CloudInitConfig;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateCloudInitConfigRequest.
 * Use `create(UpdateCloudInitConfigRequestSchema)` to create a new message.
 */
export declare const UpdateCloudInitConfigRequestSchema: GenMessage<UpdateCloudInitConfigRequest>;
/**
 * UpdateCloudInitConfigResponse confirms the update
 *
 * @generated from message obiente.cloud.vps.v1.UpdateCloudInitConfigResponse
 */
export type UpdateCloudInitConfigResponse = Message<"obiente.cloud.vps.v1.UpdateCloudInitConfigResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.CloudInitConfig cloud_init = 1;
     */
    cloudInit?: CloudInitConfig;
    /**
     * Information message about when changes take effect
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateCloudInitConfigResponse.
 * Use `create(UpdateCloudInitConfigResponseSchema)` to create a new message.
 */
export declare const UpdateCloudInitConfigResponseSchema: GenMessage<UpdateCloudInitConfigResponse>;
/**
 * ListVPSUsersRequest lists all users configured for a VPS
 *
 * @generated from message obiente.cloud.vps.v1.ListVPSUsersRequest
 */
export type ListVPSUsersRequest = Message<"obiente.cloud.vps.v1.ListVPSUsersRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.ListVPSUsersRequest.
 * Use `create(ListVPSUsersRequestSchema)` to create a new message.
 */
export declare const ListVPSUsersRequestSchema: GenMessage<ListVPSUsersRequest>;
/**
 * ListVPSUsersResponse returns the list of users
 *
 * @generated from message obiente.cloud.vps.v1.ListVPSUsersResponse
 */
export type ListVPSUsersResponse = Message<"obiente.cloud.vps.v1.ListVPSUsersResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.vps.v1.VPSUser users = 1;
     */
    users: VPSUser[];
};
/**
 * Describes the message obiente.cloud.vps.v1.ListVPSUsersResponse.
 * Use `create(ListVPSUsersResponseSchema)` to create a new message.
 */
export declare const ListVPSUsersResponseSchema: GenMessage<ListVPSUsersResponse>;
/**
 * VPSUser represents a user configured on a VPS instance
 *
 * @generated from message obiente.cloud.vps.v1.VPSUser
 */
export type VPSUser = Message<"obiente.cloud.vps.v1.VPSUser"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * Whether user has a password set (password itself is never returned)
     *
     * @generated from field: bool has_password = 2;
     */
    hasPassword: boolean;
    /**
     * SSH public keys
     *
     * @generated from field: repeated string ssh_authorized_keys = 3;
     */
    sshAuthorizedKeys: string[];
    /**
     * @generated from field: bool sudo = 4;
     */
    sudo: boolean;
    /**
     * @generated from field: bool sudo_nopasswd = 5;
     */
    sudoNopasswd: boolean;
    /**
     * @generated from field: repeated string groups = 6;
     */
    groups: string[];
    /**
     * @generated from field: optional string shell = 7;
     */
    shell?: string;
    /**
     * @generated from field: bool lock_passwd = 8;
     */
    lockPasswd: boolean;
    /**
     * @generated from field: optional string gecos = 9;
     */
    gecos?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.VPSUser.
 * Use `create(VPSUserSchema)` to create a new message.
 */
export declare const VPSUserSchema: GenMessage<VPSUser>;
/**
 * CreateVPSUserRequest creates a new user
 *
 * @generated from message obiente.cloud.vps.v1.CreateVPSUserRequest
 */
export type CreateVPSUserRequest = Message<"obiente.cloud.vps.v1.CreateVPSUserRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * @generated from field: optional string password = 4;
     */
    password?: string;
    /**
     * @generated from field: repeated string ssh_authorized_keys = 5;
     */
    sshAuthorizedKeys: string[];
    /**
     * @generated from field: optional bool sudo = 6;
     */
    sudo?: boolean;
    /**
     * @generated from field: optional bool sudo_nopasswd = 7;
     */
    sudoNopasswd?: boolean;
    /**
     * @generated from field: repeated string groups = 8;
     */
    groups: string[];
    /**
     * @generated from field: optional string shell = 9;
     */
    shell?: string;
    /**
     * @generated from field: optional bool lock_passwd = 10;
     */
    lockPasswd?: boolean;
    /**
     * @generated from field: optional string gecos = 11;
     */
    gecos?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSUserRequest.
 * Use `create(CreateVPSUserRequestSchema)` to create a new message.
 */
export declare const CreateVPSUserRequestSchema: GenMessage<CreateVPSUserRequest>;
/**
 * CreateVPSUserResponse confirms user creation
 *
 * @generated from message obiente.cloud.vps.v1.CreateVPSUserResponse
 */
export type CreateVPSUserResponse = Message<"obiente.cloud.vps.v1.CreateVPSUserResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSUser user = 1;
     */
    user?: VPSUser;
    /**
     * Information about when user will be created
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.CreateVPSUserResponse.
 * Use `create(CreateVPSUserResponseSchema)` to create a new message.
 */
export declare const CreateVPSUserResponseSchema: GenMessage<CreateVPSUserResponse>;
/**
 * UpdateVPSUserRequest updates an existing user
 *
 * @generated from message obiente.cloud.vps.v1.UpdateVPSUserRequest
 */
export type UpdateVPSUserRequest = Message<"obiente.cloud.vps.v1.UpdateVPSUserRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * User name to update
     *
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * New name (if renaming)
     *
     * @generated from field: optional string new_name = 4;
     */
    newName?: string;
    /**
     * @generated from field: repeated string ssh_authorized_keys = 5;
     */
    sshAuthorizedKeys: string[];
    /**
     * @generated from field: optional bool sudo = 6;
     */
    sudo?: boolean;
    /**
     * @generated from field: optional bool sudo_nopasswd = 7;
     */
    sudoNopasswd?: boolean;
    /**
     * @generated from field: repeated string groups = 8;
     */
    groups: string[];
    /**
     * @generated from field: optional string shell = 9;
     */
    shell?: string;
    /**
     * @generated from field: optional bool lock_passwd = 10;
     */
    lockPasswd?: boolean;
    /**
     * @generated from field: optional string gecos = 11;
     */
    gecos?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSUserRequest.
 * Use `create(UpdateVPSUserRequestSchema)` to create a new message.
 */
export declare const UpdateVPSUserRequestSchema: GenMessage<UpdateVPSUserRequest>;
/**
 * UpdateVPSUserResponse confirms the update
 *
 * @generated from message obiente.cloud.vps.v1.UpdateVPSUserResponse
 */
export type UpdateVPSUserResponse = Message<"obiente.cloud.vps.v1.UpdateVPSUserResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSUser user = 1;
     */
    user?: VPSUser;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateVPSUserResponse.
 * Use `create(UpdateVPSUserResponseSchema)` to create a new message.
 */
export declare const UpdateVPSUserResponseSchema: GenMessage<UpdateVPSUserResponse>;
/**
 * DeleteVPSUserRequest deletes a user
 *
 * @generated from message obiente.cloud.vps.v1.DeleteVPSUserRequest
 */
export type DeleteVPSUserRequest = Message<"obiente.cloud.vps.v1.DeleteVPSUserRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * User name to delete
     *
     * @generated from field: string name = 3;
     */
    name: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSUserRequest.
 * Use `create(DeleteVPSUserRequestSchema)` to create a new message.
 */
export declare const DeleteVPSUserRequestSchema: GenMessage<DeleteVPSUserRequest>;
/**
 * DeleteVPSUserResponse confirms deletion
 *
 * @generated from message obiente.cloud.vps.v1.DeleteVPSUserResponse
 */
export type DeleteVPSUserResponse = Message<"obiente.cloud.vps.v1.DeleteVPSUserResponse"> & {
    /**
     * @generated from field: string message = 1;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.DeleteVPSUserResponse.
 * Use `create(DeleteVPSUserResponseSchema)` to create a new message.
 */
export declare const DeleteVPSUserResponseSchema: GenMessage<DeleteVPSUserResponse>;
/**
 * SetUserPasswordRequest sets or resets a user's password
 *
 * @generated from message obiente.cloud.vps.v1.SetUserPasswordRequest
 */
export type SetUserPasswordRequest = Message<"obiente.cloud.vps.v1.SetUserPasswordRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: string user_name = 3;
     */
    userName: string;
    /**
     * New password (will be hashed and stored in cloud-init)
     *
     * @generated from field: string password = 4;
     */
    password: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.SetUserPasswordRequest.
 * Use `create(SetUserPasswordRequestSchema)` to create a new message.
 */
export declare const SetUserPasswordRequestSchema: GenMessage<SetUserPasswordRequest>;
/**
 * SetUserPasswordResponse confirms password update
 *
 * @generated from message obiente.cloud.vps.v1.SetUserPasswordResponse
 */
export type SetUserPasswordResponse = Message<"obiente.cloud.vps.v1.SetUserPasswordResponse"> & {
    /**
     * Information about when password will take effect
     *
     * @generated from field: string message = 1;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.SetUserPasswordResponse.
 * Use `create(SetUserPasswordResponseSchema)` to create a new message.
 */
export declare const SetUserPasswordResponseSchema: GenMessage<SetUserPasswordResponse>;
/**
 * UpdateUserSSHKeysRequest updates SSH keys for a user
 *
 * @generated from message obiente.cloud.vps.v1.UpdateUserSSHKeysRequest
 */
export type UpdateUserSSHKeysRequest = Message<"obiente.cloud.vps.v1.UpdateUserSSHKeysRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * @generated from field: string user_name = 3;
     */
    userName: string;
    /**
     * SSH key IDs from organization/VPS SSH keys
     *
     * @generated from field: repeated string ssh_key_ids = 4;
     */
    sshKeyIds: string[];
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateUserSSHKeysRequest.
 * Use `create(UpdateUserSSHKeysRequestSchema)` to create a new message.
 */
export declare const UpdateUserSSHKeysRequestSchema: GenMessage<UpdateUserSSHKeysRequest>;
/**
 * UpdateUserSSHKeysResponse confirms the update
 *
 * @generated from message obiente.cloud.vps.v1.UpdateUserSSHKeysResponse
 */
export type UpdateUserSSHKeysResponse = Message<"obiente.cloud.vps.v1.UpdateUserSSHKeysResponse"> & {
    /**
     * @generated from field: obiente.cloud.vps.v1.VPSUser user = 1;
     */
    user?: VPSUser;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.UpdateUserSSHKeysResponse.
 * Use `create(UpdateUserSSHKeysResponseSchema)` to create a new message.
 */
export declare const UpdateUserSSHKeysResponseSchema: GenMessage<UpdateUserSSHKeysResponse>;
/**
 * RotateTerminalKeyRequest rotates the web terminal SSH key
 *
 * @generated from message obiente.cloud.vps.v1.RotateTerminalKeyRequest
 */
export type RotateTerminalKeyRequest = Message<"obiente.cloud.vps.v1.RotateTerminalKeyRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.RotateTerminalKeyRequest.
 * Use `create(RotateTerminalKeyRequestSchema)` to create a new message.
 */
export declare const RotateTerminalKeyRequestSchema: GenMessage<RotateTerminalKeyRequest>;
/**
 * RotateTerminalKeyResponse confirms the key rotation
 *
 * @generated from message obiente.cloud.vps.v1.RotateTerminalKeyResponse
 */
export type RotateTerminalKeyResponse = Message<"obiente.cloud.vps.v1.RotateTerminalKeyResponse"> & {
    /**
     * Fingerprint of the new key
     *
     * @generated from field: string fingerprint = 1;
     */
    fingerprint: string;
    /**
     * Information about when the new key will take effect
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RotateTerminalKeyResponse.
 * Use `create(RotateTerminalKeyResponseSchema)` to create a new message.
 */
export declare const RotateTerminalKeyResponseSchema: GenMessage<RotateTerminalKeyResponse>;
/**
 * RemoveTerminalKeyRequest removes the web terminal SSH key
 *
 * @generated from message obiente.cloud.vps.v1.RemoveTerminalKeyRequest
 */
export type RemoveTerminalKeyRequest = Message<"obiente.cloud.vps.v1.RemoveTerminalKeyRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.RemoveTerminalKeyRequest.
 * Use `create(RemoveTerminalKeyRequestSchema)` to create a new message.
 */
export declare const RemoveTerminalKeyRequestSchema: GenMessage<RemoveTerminalKeyRequest>;
/**
 * RemoveTerminalKeyResponse confirms the key removal
 *
 * @generated from message obiente.cloud.vps.v1.RemoveTerminalKeyResponse
 */
export type RemoveTerminalKeyResponse = Message<"obiente.cloud.vps.v1.RemoveTerminalKeyResponse"> & {
    /**
     * Information about when the key will be removed
     *
     * @generated from field: string message = 1;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RemoveTerminalKeyResponse.
 * Use `create(RemoveTerminalKeyResponseSchema)` to create a new message.
 */
export declare const RemoveTerminalKeyResponseSchema: GenMessage<RemoveTerminalKeyResponse>;
/**
 * GetTerminalKeyRequest gets the web terminal SSH key status
 *
 * @generated from message obiente.cloud.vps.v1.GetTerminalKeyRequest
 */
export type GetTerminalKeyRequest = Message<"obiente.cloud.vps.v1.GetTerminalKeyRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.GetTerminalKeyRequest.
 * Use `create(GetTerminalKeyRequestSchema)` to create a new message.
 */
export declare const GetTerminalKeyRequestSchema: GenMessage<GetTerminalKeyRequest>;
/**
 * GetTerminalKeyResponse returns the terminal key status
 *
 * @generated from message obiente.cloud.vps.v1.GetTerminalKeyResponse
 */
export type GetTerminalKeyResponse = Message<"obiente.cloud.vps.v1.GetTerminalKeyResponse"> & {
    /**
     * Fingerprint of the key
     *
     * @generated from field: string fingerprint = 1;
     */
    fingerprint: string;
    /**
     * Timestamp when the key was created
     *
     * @generated from field: google.protobuf.Timestamp created_at = 2;
     */
    createdAt?: Timestamp;
    /**
     * Timestamp when the key was last updated
     *
     * @generated from field: google.protobuf.Timestamp updated_at = 3;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetTerminalKeyResponse.
 * Use `create(GetTerminalKeyResponseSchema)` to create a new message.
 */
export declare const GetTerminalKeyResponseSchema: GenMessage<GetTerminalKeyResponse>;
/**
 * RotateBastionKeyRequest rotates the bastion SSH key
 *
 * @generated from message obiente.cloud.vps.v1.RotateBastionKeyRequest
 */
export type RotateBastionKeyRequest = Message<"obiente.cloud.vps.v1.RotateBastionKeyRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.RotateBastionKeyRequest.
 * Use `create(RotateBastionKeyRequestSchema)` to create a new message.
 */
export declare const RotateBastionKeyRequestSchema: GenMessage<RotateBastionKeyRequest>;
/**
 * RotateBastionKeyResponse confirms the key rotation
 *
 * @generated from message obiente.cloud.vps.v1.RotateBastionKeyResponse
 */
export type RotateBastionKeyResponse = Message<"obiente.cloud.vps.v1.RotateBastionKeyResponse"> & {
    /**
     * Fingerprint of the new key
     *
     * @generated from field: string fingerprint = 1;
     */
    fingerprint: string;
    /**
     * Information about when the new key will take effect
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RotateBastionKeyResponse.
 * Use `create(RotateBastionKeyResponseSchema)` to create a new message.
 */
export declare const RotateBastionKeyResponseSchema: GenMessage<RotateBastionKeyResponse>;
/**
 * GetBastionKeyRequest gets the bastion SSH key status
 *
 * @generated from message obiente.cloud.vps.v1.GetBastionKeyRequest
 */
export type GetBastionKeyRequest = Message<"obiente.cloud.vps.v1.GetBastionKeyRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.GetBastionKeyRequest.
 * Use `create(GetBastionKeyRequestSchema)` to create a new message.
 */
export declare const GetBastionKeyRequestSchema: GenMessage<GetBastionKeyRequest>;
/**
 * GetBastionKeyResponse returns the bastion key status
 *
 * @generated from message obiente.cloud.vps.v1.GetBastionKeyResponse
 */
export type GetBastionKeyResponse = Message<"obiente.cloud.vps.v1.GetBastionKeyResponse"> & {
    /**
     * Fingerprint of the key
     *
     * @generated from field: string fingerprint = 1;
     */
    fingerprint: string;
    /**
     * Timestamp when the key was created
     *
     * @generated from field: google.protobuf.Timestamp created_at = 2;
     */
    createdAt?: Timestamp;
    /**
     * Timestamp when the key was last updated
     *
     * @generated from field: google.protobuf.Timestamp updated_at = 3;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetBastionKeyResponse.
 * Use `create(GetBastionKeyResponseSchema)` to create a new message.
 */
export declare const GetBastionKeyResponseSchema: GenMessage<GetBastionKeyResponse>;
/**
 * GetSSHAliasRequest gets the SSH alias for a VPS instance
 *
 * @generated from message obiente.cloud.vps.v1.GetSSHAliasRequest
 */
export type GetSSHAliasRequest = Message<"obiente.cloud.vps.v1.GetSSHAliasRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.GetSSHAliasRequest.
 * Use `create(GetSSHAliasRequestSchema)` to create a new message.
 */
export declare const GetSSHAliasRequestSchema: GenMessage<GetSSHAliasRequest>;
/**
 * GetSSHAliasResponse returns the SSH alias
 *
 * @generated from message obiente.cloud.vps.v1.GetSSHAliasResponse
 */
export type GetSSHAliasResponse = Message<"obiente.cloud.vps.v1.GetSSHAliasResponse"> & {
    /**
     * The SSH alias, or empty if not set
     *
     * @generated from field: optional string alias = 1;
     */
    alias?: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.GetSSHAliasResponse.
 * Use `create(GetSSHAliasResponseSchema)` to create a new message.
 */
export declare const GetSSHAliasResponseSchema: GenMessage<GetSSHAliasResponse>;
/**
 * SetSSHAliasRequest sets the SSH alias for a VPS instance
 *
 * @generated from message obiente.cloud.vps.v1.SetSSHAliasRequest
 */
export type SetSSHAliasRequest = Message<"obiente.cloud.vps.v1.SetSSHAliasRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string vps_id = 2;
     */
    vpsId: string;
    /**
     * The alias to set (must be unique, alphanumeric with hyphens/underscores)
     *
     * @generated from field: string alias = 3;
     */
    alias: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.SetSSHAliasRequest.
 * Use `create(SetSSHAliasRequestSchema)` to create a new message.
 */
export declare const SetSSHAliasRequestSchema: GenMessage<SetSSHAliasRequest>;
/**
 * SetSSHAliasResponse confirms the alias was set
 *
 * @generated from message obiente.cloud.vps.v1.SetSSHAliasResponse
 */
export type SetSSHAliasResponse = Message<"obiente.cloud.vps.v1.SetSSHAliasResponse"> & {
    /**
     * The alias that was set
     *
     * @generated from field: string alias = 1;
     */
    alias: string;
    /**
     * Information message
     *
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.SetSSHAliasResponse.
 * Use `create(SetSSHAliasResponseSchema)` to create a new message.
 */
export declare const SetSSHAliasResponseSchema: GenMessage<SetSSHAliasResponse>;
/**
 * RemoveSSHAliasRequest removes the SSH alias for a VPS instance
 *
 * @generated from message obiente.cloud.vps.v1.RemoveSSHAliasRequest
 */
export type RemoveSSHAliasRequest = Message<"obiente.cloud.vps.v1.RemoveSSHAliasRequest"> & {
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
 * Describes the message obiente.cloud.vps.v1.RemoveSSHAliasRequest.
 * Use `create(RemoveSSHAliasRequestSchema)` to create a new message.
 */
export declare const RemoveSSHAliasRequestSchema: GenMessage<RemoveSSHAliasRequest>;
/**
 * RemoveSSHAliasResponse confirms the alias was removed
 *
 * @generated from message obiente.cloud.vps.v1.RemoveSSHAliasResponse
 */
export type RemoveSSHAliasResponse = Message<"obiente.cloud.vps.v1.RemoveSSHAliasResponse"> & {
    /**
     * Confirmation message
     *
     * @generated from field: string message = 1;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.vps.v1.RemoveSSHAliasResponse.
 * Use `create(RemoveSSHAliasResponseSchema)` to create a new message.
 */
export declare const RemoveSSHAliasResponseSchema: GenMessage<RemoveSSHAliasResponse>;
/**
 * VPSConfigService provides endpoints for managing VPS configuration
 * including cloud-init settings and user management
 *
 * @generated from service obiente.cloud.vps.v1.VPSConfigService
 */
export declare const VPSConfigService: GenService<{
    /**
     * Get cloud-init configuration for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.GetCloudInitConfig
     */
    getCloudInitConfig: {
        methodKind: "unary";
        input: typeof GetCloudInitConfigRequestSchema;
        output: typeof GetCloudInitConfigResponseSchema;
    };
    /**
     * Get the actual generated cloud-init userData (includes bastion/terminal keys)
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.GetCloudInitUserData
     */
    getCloudInitUserData: {
        methodKind: "unary";
        input: typeof GetCloudInitUserDataRequestSchema;
        output: typeof GetCloudInitUserDataResponseSchema;
    };
    /**
     * Update cloud-init configuration for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.UpdateCloudInitConfig
     */
    updateCloudInitConfig: {
        methodKind: "unary";
        input: typeof UpdateCloudInitConfigRequestSchema;
        output: typeof UpdateCloudInitConfigResponseSchema;
    };
    /**
     * List users configured for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.ListVPSUsers
     */
    listVPSUsers: {
        methodKind: "unary";
        input: typeof ListVPSUsersRequestSchema;
        output: typeof ListVPSUsersResponseSchema;
    };
    /**
     * Create a new user on a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.CreateVPSUser
     */
    createVPSUser: {
        methodKind: "unary";
        input: typeof CreateVPSUserRequestSchema;
        output: typeof CreateVPSUserResponseSchema;
    };
    /**
     * Update an existing user on a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.UpdateVPSUser
     */
    updateVPSUser: {
        methodKind: "unary";
        input: typeof UpdateVPSUserRequestSchema;
        output: typeof UpdateVPSUserResponseSchema;
    };
    /**
     * Delete a user from a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.DeleteVPSUser
     */
    deleteVPSUser: {
        methodKind: "unary";
        input: typeof DeleteVPSUserRequestSchema;
        output: typeof DeleteVPSUserResponseSchema;
    };
    /**
     * Set or reset password for a user
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.SetUserPassword
     */
    setUserPassword: {
        methodKind: "unary";
        input: typeof SetUserPasswordRequestSchema;
        output: typeof SetUserPasswordResponseSchema;
    };
    /**
     * Manage SSH keys for a specific user
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.UpdateUserSSHKeys
     */
    updateUserSSHKeys: {
        methodKind: "unary";
        input: typeof UpdateUserSSHKeysRequestSchema;
        output: typeof UpdateUserSSHKeysResponseSchema;
    };
    /**
     * Rotate the web terminal SSH key for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.RotateTerminalKey
     */
    rotateTerminalKey: {
        methodKind: "unary";
        input: typeof RotateTerminalKeyRequestSchema;
        output: typeof RotateTerminalKeyResponseSchema;
    };
    /**
     * Remove the web terminal SSH key for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.RemoveTerminalKey
     */
    removeTerminalKey: {
        methodKind: "unary";
        input: typeof RemoveTerminalKeyRequestSchema;
        output: typeof RemoveTerminalKeyResponseSchema;
    };
    /**
     * Get the web terminal SSH key status for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.GetTerminalKey
     */
    getTerminalKey: {
        methodKind: "unary";
        input: typeof GetTerminalKeyRequestSchema;
        output: typeof GetTerminalKeyResponseSchema;
    };
    /**
     * Rotate the bastion SSH key for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.RotateBastionKey
     */
    rotateBastionKey: {
        methodKind: "unary";
        input: typeof RotateBastionKeyRequestSchema;
        output: typeof RotateBastionKeyResponseSchema;
    };
    /**
     * Get the bastion SSH key status for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.GetBastionKey
     */
    getBastionKey: {
        methodKind: "unary";
        input: typeof GetBastionKeyRequestSchema;
        output: typeof GetBastionKeyResponseSchema;
    };
    /**
     * Get the SSH alias for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.GetSSHAlias
     */
    getSSHAlias: {
        methodKind: "unary";
        input: typeof GetSSHAliasRequestSchema;
        output: typeof GetSSHAliasResponseSchema;
    };
    /**
     * Set the SSH alias for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.SetSSHAlias
     */
    setSSHAlias: {
        methodKind: "unary";
        input: typeof SetSSHAliasRequestSchema;
        output: typeof SetSSHAliasResponseSchema;
    };
    /**
     * Remove the SSH alias for a VPS instance
     *
     * @generated from rpc obiente.cloud.vps.v1.VPSConfigService.RemoveSSHAlias
     */
    removeSSHAlias: {
        methodKind: "unary";
        input: typeof RemoveSSHAliasRequestSchema;
        output: typeof RemoveSSHAliasResponseSchema;
    };
}>;
