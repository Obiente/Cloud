import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/auth/v1/auth_service.proto.
 */
export declare const file_obiente_cloud_auth_v1_auth_service: GenFile;
/**
 * Public Configuration Messages
 *
 * @generated from message obiente.cloud.auth.v1.GetPublicConfigRequest
 */
export type GetPublicConfigRequest = Message<"obiente.cloud.auth.v1.GetPublicConfigRequest"> & {};
/**
 * Describes the message obiente.cloud.auth.v1.GetPublicConfigRequest.
 * Use `create(GetPublicConfigRequestSchema)` to create a new message.
 */
export declare const GetPublicConfigRequestSchema: GenMessage<GetPublicConfigRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.GetPublicConfigResponse
 */
export type GetPublicConfigResponse = Message<"obiente.cloud.auth.v1.GetPublicConfigResponse"> & {
    /**
     * @generated from field: bool billing_enabled = 1;
     */
    billingEnabled: boolean;
    /**
     * @generated from field: bool self_hosted = 2;
     */
    selfHosted: boolean;
    /**
     * @generated from field: bool disable_auth = 3;
     */
    disableAuth: boolean;
};
/**
 * Describes the message obiente.cloud.auth.v1.GetPublicConfigResponse.
 * Use `create(GetPublicConfigResponseSchema)` to create a new message.
 */
export declare const GetPublicConfigResponseSchema: GenMessage<GetPublicConfigResponse>;
/**
 * Empty request for getting current user
 *
 * @generated from message obiente.cloud.auth.v1.GetCurrentUserRequest
 */
export type GetCurrentUserRequest = Message<"obiente.cloud.auth.v1.GetCurrentUserRequest"> & {};
/**
 * Describes the message obiente.cloud.auth.v1.GetCurrentUserRequest.
 * Use `create(GetCurrentUserRequestSchema)` to create a new message.
 */
export declare const GetCurrentUserRequestSchema: GenMessage<GetCurrentUserRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.GetCurrentUserResponse
 */
export type GetCurrentUserResponse = Message<"obiente.cloud.auth.v1.GetCurrentUserResponse"> & {
    /**
     * @generated from field: obiente.cloud.auth.v1.User user = 1;
     */
    user?: User;
};
/**
 * Describes the message obiente.cloud.auth.v1.GetCurrentUserResponse.
 * Use `create(GetCurrentUserResponseSchema)` to create a new message.
 */
export declare const GetCurrentUserResponseSchema: GenMessage<GetCurrentUserResponse>;
/**
 * @generated from message obiente.cloud.auth.v1.User
 */
export type User = Message<"obiente.cloud.auth.v1.User"> & {
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
     * @generated from field: string avatar_url = 4;
     */
    avatarUrl: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 5;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: string timezone = 6;
     */
    timezone: string;
    /**
     * Extended Zitadel userinfo fields
     *
     * @generated from field: string given_name = 7;
     */
    givenName: string;
    /**
     * @generated from field: string family_name = 8;
     */
    familyName: string;
    /**
     * @generated from field: string preferred_username = 9;
     */
    preferredUsername: string;
    /**
     * @generated from field: bool email_verified = 10;
     */
    emailVerified: boolean;
    /**
     * @generated from field: string locale = 11;
     */
    locale: string;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 12;
     */
    updatedAt?: Timestamp;
    /**
     * @generated from field: repeated string roles = 13;
     */
    roles: string[];
};
/**
 * Describes the message obiente.cloud.auth.v1.User.
 * Use `create(UserSchema)` to create a new message.
 */
export declare const UserSchema: GenMessage<User>;
/**
 * GitHub Integration Messages
 *
 * @generated from message obiente.cloud.auth.v1.ConnectGitHubRequest
 */
export type ConnectGitHubRequest = Message<"obiente.cloud.auth.v1.ConnectGitHubRequest"> & {
    /**
     * GitHub OAuth access token
     *
     * @generated from field: string access_token = 1;
     */
    accessToken: string;
    /**
     * GitHub username
     *
     * @generated from field: string username = 2;
     */
    username: string;
    /**
     * Granted OAuth scopes
     *
     * @generated from field: string scope = 3;
     */
    scope: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.ConnectGitHubRequest.
 * Use `create(ConnectGitHubRequestSchema)` to create a new message.
 */
export declare const ConnectGitHubRequestSchema: GenMessage<ConnectGitHubRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.ConnectGitHubResponse
 */
export type ConnectGitHubResponse = Message<"obiente.cloud.auth.v1.ConnectGitHubResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * GitHub username
     *
     * @generated from field: string username = 2;
     */
    username: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.ConnectGitHubResponse.
 * Use `create(ConnectGitHubResponseSchema)` to create a new message.
 */
export declare const ConnectGitHubResponseSchema: GenMessage<ConnectGitHubResponse>;
/**
 * @generated from message obiente.cloud.auth.v1.DisconnectGitHubRequest
 */
export type DisconnectGitHubRequest = Message<"obiente.cloud.auth.v1.DisconnectGitHubRequest"> & {};
/**
 * Describes the message obiente.cloud.auth.v1.DisconnectGitHubRequest.
 * Use `create(DisconnectGitHubRequestSchema)` to create a new message.
 */
export declare const DisconnectGitHubRequestSchema: GenMessage<DisconnectGitHubRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.DisconnectGitHubResponse
 */
export type DisconnectGitHubResponse = Message<"obiente.cloud.auth.v1.DisconnectGitHubResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.auth.v1.DisconnectGitHubResponse.
 * Use `create(DisconnectGitHubResponseSchema)` to create a new message.
 */
export declare const DisconnectGitHubResponseSchema: GenMessage<DisconnectGitHubResponse>;
/**
 * @generated from message obiente.cloud.auth.v1.GetGitHubStatusRequest
 */
export type GetGitHubStatusRequest = Message<"obiente.cloud.auth.v1.GetGitHubStatusRequest"> & {};
/**
 * Describes the message obiente.cloud.auth.v1.GetGitHubStatusRequest.
 * Use `create(GetGitHubStatusRequestSchema)` to create a new message.
 */
export declare const GetGitHubStatusRequestSchema: GenMessage<GetGitHubStatusRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.GetGitHubStatusResponse
 */
export type GetGitHubStatusResponse = Message<"obiente.cloud.auth.v1.GetGitHubStatusResponse"> & {
    /**
     * @generated from field: bool connected = 1;
     */
    connected: boolean;
    /**
     * GitHub username if connected
     *
     * @generated from field: string username = 2;
     */
    username: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.GetGitHubStatusResponse.
 * Use `create(GetGitHubStatusResponseSchema)` to create a new message.
 */
export declare const GetGitHubStatusResponseSchema: GenMessage<GetGitHubStatusResponse>;
/**
 * Organization GitHub Integration Messages
 *
 * @generated from message obiente.cloud.auth.v1.ConnectOrganizationGitHubRequest
 */
export type ConnectOrganizationGitHubRequest = Message<"obiente.cloud.auth.v1.ConnectOrganizationGitHubRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * GitHub OAuth access token
     *
     * @generated from field: string access_token = 2;
     */
    accessToken: string;
    /**
     * GitHub username
     *
     * @generated from field: string username = 3;
     */
    username: string;
    /**
     * Granted OAuth scopes
     *
     * @generated from field: string scope = 4;
     */
    scope: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.ConnectOrganizationGitHubRequest.
 * Use `create(ConnectOrganizationGitHubRequestSchema)` to create a new message.
 */
export declare const ConnectOrganizationGitHubRequestSchema: GenMessage<ConnectOrganizationGitHubRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.ConnectOrganizationGitHubResponse
 */
export type ConnectOrganizationGitHubResponse = Message<"obiente.cloud.auth.v1.ConnectOrganizationGitHubResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * GitHub username
     *
     * @generated from field: string username = 2;
     */
    username: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.ConnectOrganizationGitHubResponse.
 * Use `create(ConnectOrganizationGitHubResponseSchema)` to create a new message.
 */
export declare const ConnectOrganizationGitHubResponseSchema: GenMessage<ConnectOrganizationGitHubResponse>;
/**
 * @generated from message obiente.cloud.auth.v1.DisconnectOrganizationGitHubRequest
 */
export type DisconnectOrganizationGitHubRequest = Message<"obiente.cloud.auth.v1.DisconnectOrganizationGitHubRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.DisconnectOrganizationGitHubRequest.
 * Use `create(DisconnectOrganizationGitHubRequestSchema)` to create a new message.
 */
export declare const DisconnectOrganizationGitHubRequestSchema: GenMessage<DisconnectOrganizationGitHubRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.DisconnectOrganizationGitHubResponse
 */
export type DisconnectOrganizationGitHubResponse = Message<"obiente.cloud.auth.v1.DisconnectOrganizationGitHubResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.auth.v1.DisconnectOrganizationGitHubResponse.
 * Use `create(DisconnectOrganizationGitHubResponseSchema)` to create a new message.
 */
export declare const DisconnectOrganizationGitHubResponseSchema: GenMessage<DisconnectOrganizationGitHubResponse>;
/**
 * @generated from message obiente.cloud.auth.v1.ListGitHubIntegrationsRequest
 */
export type ListGitHubIntegrationsRequest = Message<"obiente.cloud.auth.v1.ListGitHubIntegrationsRequest"> & {};
/**
 * Describes the message obiente.cloud.auth.v1.ListGitHubIntegrationsRequest.
 * Use `create(ListGitHubIntegrationsRequestSchema)` to create a new message.
 */
export declare const ListGitHubIntegrationsRequestSchema: GenMessage<ListGitHubIntegrationsRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.GitHubIntegrationInfo
 */
export type GitHubIntegrationInfo = Message<"obiente.cloud.auth.v1.GitHubIntegrationInfo"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string username = 2;
     */
    username: string;
    /**
     * @generated from field: string scope = 3;
     */
    scope: string;
    /**
     * true if user integration, false if organization
     *
     * @generated from field: bool is_user = 4;
     */
    isUser: boolean;
    /**
     * Only set if is_user is false
     *
     * @generated from field: string organization_id = 5;
     */
    organizationId: string;
    /**
     * Only set if is_user is false
     *
     * @generated from field: string organization_name = 6;
     */
    organizationName: string;
    /**
     * @generated from field: google.protobuf.Timestamp connected_at = 7;
     */
    connectedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.auth.v1.GitHubIntegrationInfo.
 * Use `create(GitHubIntegrationInfoSchema)` to create a new message.
 */
export declare const GitHubIntegrationInfoSchema: GenMessage<GitHubIntegrationInfo>;
/**
 * @generated from message obiente.cloud.auth.v1.ListGitHubIntegrationsResponse
 */
export type ListGitHubIntegrationsResponse = Message<"obiente.cloud.auth.v1.ListGitHubIntegrationsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.auth.v1.GitHubIntegrationInfo integrations = 1;
     */
    integrations: GitHubIntegrationInfo[];
};
/**
 * Describes the message obiente.cloud.auth.v1.ListGitHubIntegrationsResponse.
 * Use `create(ListGitHubIntegrationsResponseSchema)` to create a new message.
 */
export declare const ListGitHubIntegrationsResponseSchema: GenMessage<ListGitHubIntegrationsResponse>;
/**
 * User Profile Update Messages
 *
 * @generated from message obiente.cloud.auth.v1.UpdateUserProfileRequest
 */
export type UpdateUserProfileRequest = Message<"obiente.cloud.auth.v1.UpdateUserProfileRequest"> & {
    /**
     * Optional: Update given name (first name)
     *
     * @generated from field: optional string given_name = 1;
     */
    givenName?: string;
    /**
     * Optional: Update family name (last name)
     *
     * @generated from field: optional string family_name = 2;
     */
    familyName?: string;
    /**
     * Optional: Update display name
     *
     * @generated from field: optional string name = 3;
     */
    name?: string;
    /**
     * Optional: Update preferred username
     *
     * @generated from field: optional string preferred_username = 4;
     */
    preferredUsername?: string;
    /**
     * Optional: Update locale
     *
     * @generated from field: optional string locale = 5;
     */
    locale?: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.UpdateUserProfileRequest.
 * Use `create(UpdateUserProfileRequestSchema)` to create a new message.
 */
export declare const UpdateUserProfileRequestSchema: GenMessage<UpdateUserProfileRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.UpdateUserProfileResponse
 */
export type UpdateUserProfileResponse = Message<"obiente.cloud.auth.v1.UpdateUserProfileResponse"> & {
    /**
     * Updated user information
     *
     * @generated from field: obiente.cloud.auth.v1.User user = 1;
     */
    user?: User;
};
/**
 * Describes the message obiente.cloud.auth.v1.UpdateUserProfileResponse.
 * Use `create(UpdateUserProfileResponseSchema)` to create a new message.
 */
export declare const UpdateUserProfileResponseSchema: GenMessage<UpdateUserProfileResponse>;
/**
 * Login Messages
 *
 * @generated from message obiente.cloud.auth.v1.LoginRequest
 */
export type LoginRequest = Message<"obiente.cloud.auth.v1.LoginRequest"> & {
    /**
     * @generated from field: string email = 1;
     */
    email: string;
    /**
     * @generated from field: string password = 2;
     */
    password: string;
    /**
     * Whether to set a longer session expiry
     *
     * @generated from field: bool remember_me = 3;
     */
    rememberMe: boolean;
};
/**
 * Describes the message obiente.cloud.auth.v1.LoginRequest.
 * Use `create(LoginRequestSchema)` to create a new message.
 */
export declare const LoginRequestSchema: GenMessage<LoginRequest>;
/**
 * @generated from message obiente.cloud.auth.v1.LoginResponse
 */
export type LoginResponse = Message<"obiente.cloud.auth.v1.LoginResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * OAuth2 access token
     *
     * @generated from field: string access_token = 2;
     */
    accessToken: string;
    /**
     * OAuth2 refresh token
     *
     * @generated from field: string refresh_token = 3;
     */
    refreshToken: string;
    /**
     * Token expiry in seconds
     *
     * @generated from field: int32 expires_in = 4;
     */
    expiresIn: number;
    /**
     * Error message if success is false
     *
     * @generated from field: string message = 5;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.auth.v1.LoginResponse.
 * Use `create(LoginResponseSchema)` to create a new message.
 */
export declare const LoginResponseSchema: GenMessage<LoginResponse>;
/**
 * @generated from service obiente.cloud.auth.v1.AuthService
 */
export declare const AuthService: GenService<{
    /**
     * Get public configuration (no authentication required)
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.GetPublicConfig
     */
    getPublicConfig: {
        methodKind: "unary";
        input: typeof GetPublicConfigRequestSchema;
        output: typeof GetPublicConfigResponseSchema;
    };
    /**
     * Login with email and password using service account
     * This endpoint does not require authentication (public)
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.Login
     */
    login: {
        methodKind: "unary";
        input: typeof LoginRequestSchema;
        output: typeof LoginResponseSchema;
    };
    /**
     * Get current authenticated user
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.GetCurrentUser
     */
    getCurrentUser: {
        methodKind: "unary";
        input: typeof GetCurrentUserRequestSchema;
        output: typeof GetCurrentUserResponseSchema;
    };
    /**
     * Update user profile information
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.UpdateUserProfile
     */
    updateUserProfile: {
        methodKind: "unary";
        input: typeof UpdateUserProfileRequestSchema;
        output: typeof UpdateUserProfileResponseSchema;
    };
    /**
     * GitHub Integration
     * Connect GitHub account by storing OAuth token
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.ConnectGitHub
     */
    connectGitHub: {
        methodKind: "unary";
        input: typeof ConnectGitHubRequestSchema;
        output: typeof ConnectGitHubResponseSchema;
    };
    /**
     * Disconnect GitHub account
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.DisconnectGitHub
     */
    disconnectGitHub: {
        methodKind: "unary";
        input: typeof DisconnectGitHubRequestSchema;
        output: typeof DisconnectGitHubResponseSchema;
    };
    /**
     * Get GitHub connection status
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.GetGitHubStatus
     */
    getGitHubStatus: {
        methodKind: "unary";
        input: typeof GetGitHubStatusRequestSchema;
        output: typeof GetGitHubStatusResponseSchema;
    };
    /**
     * Organization GitHub Integration
     * Connect organization GitHub account by storing OAuth token
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.ConnectOrganizationGitHub
     */
    connectOrganizationGitHub: {
        methodKind: "unary";
        input: typeof ConnectOrganizationGitHubRequestSchema;
        output: typeof ConnectOrganizationGitHubResponseSchema;
    };
    /**
     * Disconnect organization GitHub account
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.DisconnectOrganizationGitHub
     */
    disconnectOrganizationGitHub: {
        methodKind: "unary";
        input: typeof DisconnectOrganizationGitHubRequestSchema;
        output: typeof DisconnectOrganizationGitHubResponseSchema;
    };
    /**
     * List all GitHub integrations (user and organizations)
     *
     * @generated from rpc obiente.cloud.auth.v1.AuthService.ListGitHubIntegrations
     */
    listGitHubIntegrations: {
        methodKind: "unary";
        input: typeof ListGitHubIntegrationsRequestSchema;
        output: typeof ListGitHubIntegrationsResponseSchema;
    };
}>;
