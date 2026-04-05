import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { ChunkedUploadPayload, ChunkedUploadResponsePayload, CreateServerFileArchiveRequest, CreateServerFileArchiveResponse, LogLevel } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/gameservers/v1/game_server_service.proto.
 */
export declare const file_obiente_cloud_gameservers_v1_game_server_service: GenFile;
/**
 * Request/Response messages
 *
 * @generated from message obiente.cloud.gameservers.v1.ListGameServersRequest
 */
export type ListGameServersRequest = Message<"obiente.cloud.gameservers.v1.ListGameServersRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Filter by game type
     *
     * @generated from field: optional string game_type = 2;
     */
    gameType?: string;
    /**
     * Filter by status
     *
     * @generated from field: optional obiente.cloud.gameservers.v1.GameServerStatus status = 3;
     */
    status?: GameServerStatus;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListGameServersRequest.
 * Use `create(ListGameServersRequestSchema)` to create a new message.
 */
export declare const ListGameServersRequestSchema: GenMessage<ListGameServersRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ListGameServersResponse
 */
export type ListGameServersResponse = Message<"obiente.cloud.gameservers.v1.ListGameServersResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServer game_servers = 1;
     */
    gameServers: GameServer[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListGameServersResponse.
 * Use `create(ListGameServersResponseSchema)` to create a new message.
 */
export declare const ListGameServersResponseSchema: GenMessage<ListGameServersResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerRequest
 */
export type CreateGameServerRequest = Message<"obiente.cloud.gameservers.v1.CreateGameServerRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameType game_type = 3;
     */
    gameType: GameType;
    /**
     * Resource configuration
     *
     * Memory limit in bytes (e.g., 2147483648 for 2GB)
     *
     * @generated from field: optional int64 memory_bytes = 4;
     */
    memoryBytes?: bigint;
    /**
     * CPU cores (e.g., 2 for 2 cores)
     *
     * @generated from field: optional int32 cpu_cores = 5;
     */
    cpuCores?: number;
    /**
     * Game server port (auto-assigned if not specified)
     *
     * @generated from field: optional int32 port = 6;
     */
    port?: number;
    /**
     * Docker image configuration
     *
     * Custom Docker image (uses default for game type if not specified)
     *
     * @generated from field: optional string docker_image = 7;
     */
    dockerImage?: string;
    /**
     * Custom start command (uses default for game type if not specified)
     *
     * @generated from field: optional string start_command = 8;
     */
    startCommand?: string;
    /**
     * Environment variables
     *
     * @generated from field: map<string, string> env_vars = 9;
     */
    envVars: {
        [key: string]: string;
    };
    /**
     * Additional configuration
     *
     * Game server version (e.g., "1.20.1" for Minecraft)
     *
     * @generated from field: optional string server_version = 10;
     */
    serverVersion?: string;
    /**
     * Optional description
     *
     * @generated from field: optional string description = 11;
     */
    description?: string;
    /**
     * Number of additional ports to allocate (0-2)
     *
     * @generated from field: optional int32 extra_ports_count = 12;
     */
    extraPortsCount?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerRequest.
 * Use `create(CreateGameServerRequestSchema)` to create a new message.
 */
export declare const CreateGameServerRequestSchema: GenMessage<CreateGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerResponse
 */
export type CreateGameServerResponse = Message<"obiente.cloud.gameservers.v1.CreateGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerResponse.
 * Use `create(CreateGameServerResponseSchema)` to create a new message.
 */
export declare const CreateGameServerResponseSchema: GenMessage<CreateGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerRequest
 */
export type GetGameServerRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerRequest.
 * Use `create(GetGameServerRequestSchema)` to create a new message.
 */
export declare const GetGameServerRequestSchema: GenMessage<GetGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerResponse
 */
export type GetGameServerResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerResponse.
 * Use `create(GetGameServerResponseSchema)` to create a new message.
 */
export declare const GetGameServerResponseSchema: GenMessage<GetGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UpdateGameServerRequest
 */
export type UpdateGameServerRequest = Message<"obiente.cloud.gameservers.v1.UpdateGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: optional string name = 2;
     */
    name?: string;
    /**
     * @generated from field: optional int64 memory_bytes = 3;
     */
    memoryBytes?: bigint;
    /**
     * @generated from field: optional int32 cpu_cores = 4;
     */
    cpuCores?: number;
    /**
     * Maps are optional by default in proto3
     *
     * @generated from field: map<string, string> env_vars = 5;
     */
    envVars: {
        [key: string]: string;
    };
    /**
     * @generated from field: optional string start_command = 6;
     */
    startCommand?: string;
    /**
     * @generated from field: optional string description = 7;
     */
    description?: string;
    /**
     * Game server version (e.g., "1.20.1" for Minecraft)
     *
     * @generated from field: optional string server_version = 8;
     */
    serverVersion?: string;
    /**
     * Number of additional ports to allocate (0-2)
     *
     * @generated from field: optional int32 extra_ports_count = 9;
     */
    extraPortsCount?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UpdateGameServerRequest.
 * Use `create(UpdateGameServerRequestSchema)` to create a new message.
 */
export declare const UpdateGameServerRequestSchema: GenMessage<UpdateGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UpdateGameServerResponse
 */
export type UpdateGameServerResponse = Message<"obiente.cloud.gameservers.v1.UpdateGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UpdateGameServerResponse.
 * Use `create(UpdateGameServerResponseSchema)` to create a new message.
 */
export declare const UpdateGameServerResponseSchema: GenMessage<UpdateGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerRequest
 */
export type DeleteGameServerRequest = Message<"obiente.cloud.gameservers.v1.DeleteGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerRequest.
 * Use `create(DeleteGameServerRequestSchema)` to create a new message.
 */
export declare const DeleteGameServerRequestSchema: GenMessage<DeleteGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerResponse
 */
export type DeleteGameServerResponse = Message<"obiente.cloud.gameservers.v1.DeleteGameServerResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerResponse.
 * Use `create(DeleteGameServerResponseSchema)` to create a new message.
 */
export declare const DeleteGameServerResponseSchema: GenMessage<DeleteGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StartGameServerRequest
 */
export type StartGameServerRequest = Message<"obiente.cloud.gameservers.v1.StartGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StartGameServerRequest.
 * Use `create(StartGameServerRequestSchema)` to create a new message.
 */
export declare const StartGameServerRequestSchema: GenMessage<StartGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StartGameServerResponse
 */
export type StartGameServerResponse = Message<"obiente.cloud.gameservers.v1.StartGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StartGameServerResponse.
 * Use `create(StartGameServerResponseSchema)` to create a new message.
 */
export declare const StartGameServerResponseSchema: GenMessage<StartGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StopGameServerRequest
 */
export type StopGameServerRequest = Message<"obiente.cloud.gameservers.v1.StopGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StopGameServerRequest.
 * Use `create(StopGameServerRequestSchema)` to create a new message.
 */
export declare const StopGameServerRequestSchema: GenMessage<StopGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StopGameServerResponse
 */
export type StopGameServerResponse = Message<"obiente.cloud.gameservers.v1.StopGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StopGameServerResponse.
 * Use `create(StopGameServerResponseSchema)` to create a new message.
 */
export declare const StopGameServerResponseSchema: GenMessage<StopGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.RestartGameServerRequest
 */
export type RestartGameServerRequest = Message<"obiente.cloud.gameservers.v1.RestartGameServerRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.RestartGameServerRequest.
 * Use `create(RestartGameServerRequestSchema)` to create a new message.
 */
export declare const RestartGameServerRequestSchema: GenMessage<RestartGameServerRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.RestartGameServerResponse
 */
export type RestartGameServerResponse = Message<"obiente.cloud.gameservers.v1.RestartGameServerResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServer game_server = 1;
     */
    gameServer?: GameServer;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.RestartGameServerResponse.
 * Use `create(RestartGameServerResponseSchema)` to create a new message.
 */
export declare const RestartGameServerResponseSchema: GenMessage<RestartGameServerResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerHTTPRoute
 */
export type GameServerHTTPRoute = Message<"obiente.cloud.gameservers.v1.GameServerHTTPRoute"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string game_server_id = 2;
     */
    gameServerId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
    /**
     * @generated from field: string path_prefix = 4;
     */
    pathPrefix: string;
    /**
     * @generated from field: int32 target_port = 5;
     */
    targetPort: number;
    /**
     * @generated from field: string protocol = 6;
     */
    protocol: string;
    /**
     * @generated from field: bool ssl_enabled = 7;
     */
    sslEnabled: boolean;
    /**
     * @generated from field: optional string ssl_cert_resolver = 8;
     */
    sslCertResolver?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerHTTPRoute.
 * Use `create(GameServerHTTPRouteSchema)` to create a new message.
 */
export declare const GameServerHTTPRouteSchema: GenMessage<GameServerHTTPRoute>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesRequest
 */
export type GetGameServerHTTPRoutesRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesRequest.
 * Use `create(GetGameServerHTTPRoutesRequestSchema)` to create a new message.
 */
export declare const GetGameServerHTTPRoutesRequestSchema: GenMessage<GetGameServerHTTPRoutesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesResponse
 */
export type GetGameServerHTTPRoutesResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerHTTPRoute routes = 1;
     */
    routes: GameServerHTTPRoute[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerHTTPRoutesResponse.
 * Use `create(GetGameServerHTTPRoutesResponseSchema)` to create a new message.
 */
export declare const GetGameServerHTTPRoutesResponseSchema: GenMessage<GetGameServerHTTPRoutesResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteRequest
 */
export type UpsertGameServerHTTPRouteRequest = Message<"obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: optional string route_id = 3;
     */
    routeId?: string;
    /**
     * @generated from field: string domain = 4;
     */
    domain: string;
    /**
     * @generated from field: optional string path_prefix = 5;
     */
    pathPrefix?: string;
    /**
     * @generated from field: int32 target_port = 6;
     */
    targetPort: number;
    /**
     * @generated from field: optional string protocol = 7;
     */
    protocol?: string;
    /**
     * @generated from field: optional bool ssl_enabled = 8;
     */
    sslEnabled?: boolean;
    /**
     * @generated from field: optional string ssl_cert_resolver = 9;
     */
    sslCertResolver?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteRequest.
 * Use `create(UpsertGameServerHTTPRouteRequestSchema)` to create a new message.
 */
export declare const UpsertGameServerHTTPRouteRequestSchema: GenMessage<UpsertGameServerHTTPRouteRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteResponse
 */
export type UpsertGameServerHTTPRouteResponse = Message<"obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerHTTPRoute route = 1;
     */
    route?: GameServerHTTPRoute;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UpsertGameServerHTTPRouteResponse.
 * Use `create(UpsertGameServerHTTPRouteResponseSchema)` to create a new message.
 */
export declare const UpsertGameServerHTTPRouteResponseSchema: GenMessage<UpsertGameServerHTTPRouteResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteRequest
 */
export type DeleteGameServerHTTPRouteRequest = Message<"obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string route_id = 3;
     */
    routeId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteRequest.
 * Use `create(DeleteGameServerHTTPRouteRequestSchema)` to create a new message.
 */
export declare const DeleteGameServerHTTPRouteRequestSchema: GenMessage<DeleteGameServerHTTPRouteRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteResponse
 */
export type DeleteGameServerHTTPRouteResponse = Message<"obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerHTTPRouteResponse.
 * Use `create(DeleteGameServerHTTPRouteResponseSchema)` to create a new message.
 */
export declare const DeleteGameServerHTTPRouteResponseSchema: GenMessage<DeleteGameServerHTTPRouteResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenRequest
 */
export type GetGameServerDomainVerificationTokenRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenRequest.
 * Use `create(GetGameServerDomainVerificationTokenRequestSchema)` to create a new message.
 */
export declare const GetGameServerDomainVerificationTokenRequestSchema: GenMessage<GetGameServerDomainVerificationTokenRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenResponse
 */
export type GetGameServerDomainVerificationTokenResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenResponse"> & {
    /**
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * @generated from field: string token = 2;
     */
    token: string;
    /**
     * @generated from field: string txt_record_name = 3;
     */
    txtRecordName: string;
    /**
     * @generated from field: string txt_record_value = 4;
     */
    txtRecordValue: string;
    /**
     * @generated from field: string status = 5;
     */
    status: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerDomainVerificationTokenResponse.
 * Use `create(GetGameServerDomainVerificationTokenResponseSchema)` to create a new message.
 */
export declare const GetGameServerDomainVerificationTokenResponseSchema: GenMessage<GetGameServerDomainVerificationTokenResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.VerifyGameServerDomainRequest
 */
export type VerifyGameServerDomainRequest = Message<"obiente.cloud.gameservers.v1.VerifyGameServerDomainRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.VerifyGameServerDomainRequest.
 * Use `create(VerifyGameServerDomainRequestSchema)` to create a new message.
 */
export declare const VerifyGameServerDomainRequestSchema: GenMessage<VerifyGameServerDomainRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.VerifyGameServerDomainResponse
 */
export type VerifyGameServerDomainResponse = Message<"obiente.cloud.gameservers.v1.VerifyGameServerDomainResponse"> & {
    /**
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * @generated from field: bool verified = 2;
     */
    verified: boolean;
    /**
     * @generated from field: string status = 3;
     */
    status: string;
    /**
     * @generated from field: optional string message = 4;
     */
    message?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.VerifyGameServerDomainResponse.
 * Use `create(VerifyGameServerDomainResponseSchema)` to create a new message.
 */
export declare const VerifyGameServerDomainResponseSchema: GenMessage<VerifyGameServerDomainResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StreamGameServerStatusRequest
 */
export type StreamGameServerStatusRequest = Message<"obiente.cloud.gameservers.v1.StreamGameServerStatusRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StreamGameServerStatusRequest.
 * Use `create(StreamGameServerStatusRequestSchema)` to create a new message.
 */
export declare const StreamGameServerStatusRequestSchema: GenMessage<StreamGameServerStatusRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerStatusUpdate
 */
export type GameServerStatusUpdate = Message<"obiente.cloud.gameservers.v1.GameServerStatusUpdate"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerStatus status = 2;
     */
    status: GameServerStatus;
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
 * Describes the message obiente.cloud.gameservers.v1.GameServerStatusUpdate.
 * Use `create(GameServerStatusUpdateSchema)` to create a new message.
 */
export declare const GameServerStatusUpdateSchema: GenMessage<GameServerStatusUpdate>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerLogsRequest
 */
export type GetGameServerLogsRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerLogsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Max number of lines to return
     *
     * @generated from field: optional int32 limit = 2;
     */
    limit?: number;
    /**
     * Get logs since this timestamp (for pagination)
     *
     * @generated from field: optional google.protobuf.Timestamp since = 3;
     */
    since?: Timestamp;
    /**
     * Get logs until this timestamp (for historical loading)
     *
     * @generated from field: optional google.protobuf.Timestamp until = 4;
     */
    until?: Timestamp;
    /**
     * Search query to filter logs (case-insensitive substring match)
     *
     * @generated from field: optional string search_query = 5;
     */
    searchQuery?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerLogsRequest.
 * Use `create(GetGameServerLogsRequestSchema)` to create a new message.
 */
export declare const GetGameServerLogsRequestSchema: GenMessage<GetGameServerLogsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerLogsResponse
 */
export type GetGameServerLogsResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerLogsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerLogLine lines = 1;
     */
    lines: GameServerLogLine[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerLogsResponse.
 * Use `create(GetGameServerLogsResponseSchema)` to create a new message.
 */
export declare const GetGameServerLogsResponseSchema: GenMessage<GetGameServerLogsResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StreamGameServerLogsRequest
 */
export type StreamGameServerLogsRequest = Message<"obiente.cloud.gameservers.v1.StreamGameServerLogsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Follow logs (like tail -f, default: true)
     *
     * @generated from field: optional bool follow = 2;
     */
    follow?: boolean;
    /**
     * Number of lines to tail (default: 100)
     *
     * @generated from field: optional int32 tail = 3;
     */
    tail?: number;
    /**
     * Get historical logs since this timestamp
     *
     * @generated from field: optional google.protobuf.Timestamp since = 4;
     */
    since?: Timestamp;
    /**
     * Get historical logs until this timestamp (for pagination)
     *
     * @generated from field: optional google.protobuf.Timestamp until = 5;
     */
    until?: Timestamp;
    /**
     * Search query to filter logs (case-insensitive substring match)
     *
     * @generated from field: optional string search_query = 6;
     */
    searchQuery?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StreamGameServerLogsRequest.
 * Use `create(StreamGameServerLogsRequestSchema)` to create a new message.
 */
export declare const StreamGameServerLogsRequestSchema: GenMessage<StreamGameServerLogsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerLogLine
 */
export type GameServerLogLine = Message<"obiente.cloud.gameservers.v1.GameServerLogLine"> & {
    /**
     * @generated from field: string line = 1;
     */
    line: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 2;
     */
    timestamp?: Timestamp;
    /**
     * @generated from field: optional obiente.cloud.common.v1.LogLevel level = 3;
     */
    level?: LogLevel;
    /**
     * @generated from field: bool stderr = 4;
     */
    stderr: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerLogLine.
 * Use `create(GameServerLogLineSchema)` to create a new message.
 */
export declare const GameServerLogLineSchema: GenMessage<GameServerLogLine>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerMetricsRequest
 */
export type GetGameServerMetricsRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerMetricsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: optional google.protobuf.Timestamp start_time = 2;
     */
    startTime?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp end_time = 3;
     */
    endTime?: Timestamp;
    /**
     * "1m", "5m", "1h", etc.
     *
     * @generated from field: optional string aggregation = 4;
     */
    aggregation?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerMetricsRequest.
 * Use `create(GetGameServerMetricsRequestSchema)` to create a new message.
 */
export declare const GetGameServerMetricsRequestSchema: GenMessage<GetGameServerMetricsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerMetricsResponse
 */
export type GetGameServerMetricsResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerMetricsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerMetric metrics = 1;
     */
    metrics: GameServerMetric[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerMetricsResponse.
 * Use `create(GetGameServerMetricsResponseSchema)` to create a new message.
 */
export declare const GetGameServerMetricsResponseSchema: GenMessage<GetGameServerMetricsResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.StreamGameServerMetricsRequest
 */
export type StreamGameServerMetricsRequest = Message<"obiente.cloud.gameservers.v1.StreamGameServerMetricsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.StreamGameServerMetricsRequest.
 * Use `create(StreamGameServerMetricsRequestSchema)` to create a new message.
 */
export declare const StreamGameServerMetricsRequestSchema: GenMessage<StreamGameServerMetricsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerMetric
 */
export type GameServerMetric = Message<"obiente.cloud.gameservers.v1.GameServerMetric"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 2;
     */
    timestamp?: Timestamp;
    /**
     * CPU usage percentage (0-100)
     *
     * @generated from field: optional double cpu_usage_percent = 3;
     */
    cpuUsagePercent?: number;
    /**
     * Current memory usage in bytes
     *
     * @generated from field: optional int64 memory_usage_bytes = 4;
     */
    memoryUsageBytes?: bigint;
    /**
     * Memory limit in bytes
     *
     * @generated from field: optional int64 memory_limit_bytes = 5;
     */
    memoryLimitBytes?: bigint;
    /**
     * Network received bytes
     *
     * @generated from field: optional int64 network_rx_bytes = 6;
     */
    networkRxBytes?: bigint;
    /**
     * Network transmitted bytes
     *
     * @generated from field: optional int64 network_tx_bytes = 7;
     */
    networkTxBytes?: bigint;
    /**
     * Disk read bytes
     *
     * @generated from field: optional int64 disk_read_bytes = 10;
     */
    diskReadBytes?: bigint;
    /**
     * Disk write bytes
     *
     * @generated from field: optional int64 disk_write_bytes = 11;
     */
    diskWriteBytes?: bigint;
    /**
     * Current player count (if available)
     *
     * @generated from field: optional int32 player_count = 8;
     */
    playerCount?: number;
    /**
     * Max player count (if available)
     *
     * @generated from field: optional int32 max_players = 9;
     */
    maxPlayers?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerMetric.
 * Use `create(GameServerMetricSchema)` to create a new message.
 */
export declare const GameServerMetricSchema: GenMessage<GameServerMetric>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerUsageRequest
 */
export type GetGameServerUsageRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerUsageRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Optional: specify a month (YYYY-MM format). Defaults to current month.
     *
     * @generated from field: optional string month = 3;
     */
    month?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerUsageRequest.
 * Use `create(GetGameServerUsageRequestSchema)` to create a new message.
 */
export declare const GetGameServerUsageRequestSchema: GenMessage<GetGameServerUsageRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerUsageResponse
 */
export type GetGameServerUsageResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerUsageResponse"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * YYYY-MM format
     *
     * @generated from field: string month = 3;
     */
    month: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerUsageMetrics current = 4;
     */
    current?: GameServerUsageMetrics;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerUsageMetrics estimated_monthly = 5;
     */
    estimatedMonthly?: GameServerUsageMetrics;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerUsageResponse.
 * Use `create(GetGameServerUsageResponseSchema)` to create a new message.
 */
export declare const GetGameServerUsageResponseSchema: GenMessage<GetGameServerUsageResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerUsageMetrics
 */
export type GameServerUsageMetrics = Message<"obiente.cloud.gameservers.v1.GameServerUsageMetrics"> & {
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
 * Describes the message obiente.cloud.gameservers.v1.GameServerUsageMetrics.
 * Use `create(GameServerUsageMetricsSchema)` to create a new message.
 */
export declare const GameServerUsageMetricsSchema: GenMessage<GameServerUsageMetrics>;
/**
 * GameServer represents a game server instance
 *
 * @generated from message obiente.cloud.gameservers.v1.GameServer
 */
export type GameServer = Message<"obiente.cloud.gameservers.v1.GameServer"> & {
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
     * @generated from field: optional string description = 4;
     */
    description?: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameType game_type = 5;
     */
    gameType: GameType;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerStatus status = 6;
     */
    status: GameServerStatus;
    /**
     * Resource configuration
     *
     * @generated from field: int64 memory_bytes = 7;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int32 cpu_cores = 8;
     */
    cpuCores: number;
    /**
     * @generated from field: int32 port = 9;
     */
    port: number;
    /**
     * @generated from field: repeated int32 extra_ports = 23;
     */
    extraPorts: number[];
    /**
     * Docker configuration
     *
     * @generated from field: string docker_image = 10;
     */
    dockerImage: string;
    /**
     * @generated from field: optional string start_command = 11;
     */
    startCommand?: string;
    /**
     * Environment variables
     *
     * @generated from field: map<string, string> env_vars = 12;
     */
    envVars: {
        [key: string]: string;
    };
    /**
     * Game-specific configuration
     *
     * @generated from field: optional string server_version = 13;
     */
    serverVersion?: string;
    /**
     * Resource usage (if available)
     *
     * @generated from field: optional int32 player_count = 14;
     */
    playerCount?: number;
    /**
     * @generated from field: optional int32 max_players = 15;
     */
    maxPlayers?: number;
    /**
     * Timestamps
     *
     * @generated from field: google.protobuf.Timestamp created_at = 16;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 17;
     */
    updatedAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp last_started_at = 18;
     */
    lastStartedAt?: Timestamp;
    /**
     * Container information
     *
     * @generated from field: optional string container_id = 19;
     */
    containerId?: string;
    /**
     * @generated from field: optional string container_name = 20;
     */
    containerName?: string;
    /**
     * Storage
     *
     * @generated from field: int64 storage_bytes = 21;
     */
    storageBytes: bigint;
    /**
     * Created by
     *
     * @generated from field: string created_by = 22;
     */
    createdBy: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServer.
 * Use `create(GameServerSchema)` to create a new message.
 */
export declare const GameServerSchema: GenMessage<GameServer>;
/**
 * File system messages (similar to deployments)
 *
 * @generated from message obiente.cloud.gameservers.v1.GameServerFile
 */
export type GameServerFile = Message<"obiente.cloud.gameservers.v1.GameServerFile"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string path = 2;
     */
    path: string;
    /**
     * @generated from field: bool is_directory = 3;
     */
    isDirectory: boolean;
    /**
     * @generated from field: int64 size = 4;
     */
    size: bigint;
    /**
     * @generated from field: string permissions = 5;
     */
    permissions: string;
    /**
     * If this file is in a volume, the volume name
     *
     * @generated from field: optional string volume_name = 6;
     */
    volumeName?: string;
    /**
     * @generated from field: optional string owner = 7;
     */
    owner?: string;
    /**
     * @generated from field: optional string group = 8;
     */
    group?: string;
    /**
     * @generated from field: optional uint32 mode_octal = 9;
     */
    modeOctal?: number;
    /**
     * @generated from field: optional bool is_symlink = 10;
     */
    isSymlink?: boolean;
    /**
     * @generated from field: optional string symlink_target = 11;
     */
    symlinkTarget?: string;
    /**
     * @generated from field: optional string mime_type = 12;
     */
    mimeType?: string;
    /**
     * @generated from field: optional google.protobuf.Timestamp modified_time = 13;
     */
    modifiedTime?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp created_time = 14;
     */
    createdTime?: Timestamp;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerFile.
 * Use `create(GameServerFileSchema)` to create a new message.
 */
export declare const GameServerFileSchema: GenMessage<GameServerFile>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerVolumeInfo
 */
export type GameServerVolumeInfo = Message<"obiente.cloud.gameservers.v1.GameServerVolumeInfo"> & {
    /**
     * Volume name
     *
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * Mount point inside container (e.g., "/data")
     *
     * @generated from field: string mount_point = 2;
     */
    mountPoint: string;
    /**
     * Volume source path on host
     *
     * @generated from field: string source = 3;
     */
    source: string;
    /**
     * Whether this is a persistent volume
     *
     * @generated from field: bool is_persistent = 4;
     */
    isPersistent: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerVolumeInfo.
 * Use `create(GameServerVolumeInfoSchema)` to create a new message.
 */
export declare const GameServerVolumeInfoSchema: GenMessage<GameServerVolumeInfo>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ListGameServerFilesRequest
 */
export type ListGameServerFilesRequest = Message<"obiente.cloud.gameservers.v1.ListGameServerFilesRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Directory path (default: "/")
     *
     * @generated from field: string path = 2;
     */
    path: string;
    /**
     * If specified, list files from this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
    /**
     * Pagination cursor
     *
     * @generated from field: optional string cursor = 4;
     */
    cursor?: string;
    /**
     * Number of files per page
     *
     * @generated from field: optional int32 page_size = 5;
     */
    pageSize?: number;
    /**
     * If true, only return available volumes
     *
     * @generated from field: optional bool list_volumes = 6;
     */
    listVolumes?: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListGameServerFilesRequest.
 * Use `create(ListGameServerFilesRequestSchema)` to create a new message.
 */
export declare const ListGameServerFilesRequestSchema: GenMessage<ListGameServerFilesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ListGameServerFilesResponse
 */
export type ListGameServerFilesResponse = Message<"obiente.cloud.gameservers.v1.ListGameServerFilesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerFile files = 1;
     */
    files: GameServerFile[];
    /**
     * @generated from field: string current_path = 2;
     */
    currentPath: string;
    /**
     * Available persistent volumes (only when list_volumes=true)
     *
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerVolumeInfo volumes = 3;
     */
    volumes: GameServerVolumeInfo[];
    /**
     * Whether the current listing is from a volume
     *
     * @generated from field: bool is_volume = 4;
     */
    isVolume: boolean;
    /**
     * Whether the container is currently running
     *
     * @generated from field: bool container_running = 5;
     */
    containerRunning: boolean;
    /**
     * @generated from field: bool has_more = 6;
     */
    hasMore: boolean;
    /**
     * @generated from field: optional string next_cursor = 7;
     */
    nextCursor?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListGameServerFilesResponse.
 * Use `create(ListGameServerFilesResponseSchema)` to create a new message.
 */
export declare const ListGameServerFilesResponseSchema: GenMessage<ListGameServerFilesResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.SearchGameServerFilesRequest
 */
export type SearchGameServerFilesRequest = Message<"obiente.cloud.gameservers.v1.SearchGameServerFilesRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Search query (searches in file and directory names, case-insensitive)
     *
     * @generated from field: string query = 2;
     */
    query: string;
    /**
     * Root path to search from (default: "/")
     *
     * @generated from field: optional string root_path = 3;
     */
    rootPath?: string;
    /**
     * If specified, search in this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * Maximum number of results to return (default: 100)
     *
     * @generated from field: optional int32 max_results = 5;
     */
    maxResults?: number;
    /**
     * If true, only return files (exclude directories)
     *
     * @generated from field: optional bool files_only = 6;
     */
    filesOnly?: boolean;
    /**
     * If true, only return directories (exclude files)
     *
     * @generated from field: optional bool directories_only = 7;
     */
    directoriesOnly?: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.SearchGameServerFilesRequest.
 * Use `create(SearchGameServerFilesRequestSchema)` to create a new message.
 */
export declare const SearchGameServerFilesRequestSchema: GenMessage<SearchGameServerFilesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.SearchGameServerFilesResponse
 */
export type SearchGameServerFilesResponse = Message<"obiente.cloud.gameservers.v1.SearchGameServerFilesResponse"> & {
    /**
     * Matching files and directories
     *
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerFile results = 1;
     */
    results: GameServerFile[];
    /**
     * Total number of matches found
     *
     * @generated from field: int32 total_found = 2;
     */
    totalFound: number;
    /**
     * Whether there are more results beyond max_results
     *
     * @generated from field: bool has_more = 3;
     */
    hasMore: boolean;
    /**
     * Whether the container is currently running
     *
     * @generated from field: bool container_running = 4;
     */
    containerRunning: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.SearchGameServerFilesResponse.
 * Use `create(SearchGameServerFilesResponseSchema)` to create a new message.
 */
export declare const SearchGameServerFilesResponseSchema: GenMessage<SearchGameServerFilesResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerFileRequest
 */
export type GetGameServerFileRequest = Message<"obiente.cloud.gameservers.v1.GetGameServerFileRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * File path
     *
     * @generated from field: string path = 2;
     */
    path: string;
    /**
     * If specified, read file from this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerFileRequest.
 * Use `create(GetGameServerFileRequestSchema)` to create a new message.
 */
export declare const GetGameServerFileRequestSchema: GenMessage<GetGameServerFileRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetGameServerFileResponse
 */
export type GetGameServerFileResponse = Message<"obiente.cloud.gameservers.v1.GetGameServerFileResponse"> & {
    /**
     * @generated from field: string content = 1;
     */
    content: string;
    /**
     * "text" or "base64"
     *
     * @generated from field: string encoding = 2;
     */
    encoding: string;
    /**
     * @generated from field: int64 size = 3;
     */
    size: bigint;
    /**
     * @generated from field: optional bool truncated = 4;
     */
    truncated?: boolean;
    /**
     * @generated from field: optional obiente.cloud.gameservers.v1.GameServerFile metadata = 5;
     */
    metadata?: GameServerFile;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetGameServerFileResponse.
 * Use `create(GetGameServerFileResponseSchema)` to create a new message.
 */
export declare const GetGameServerFileResponseSchema: GenMessage<GetGameServerFileResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UploadGameServerFilesRequest
 */
export type UploadGameServerFilesRequest = Message<"obiente.cloud.gameservers.v1.UploadGameServerFilesRequest"> & {
    /**
     * Metadata about the upload
     *
     * @generated from field: obiente.cloud.gameservers.v1.UploadGameServerFilesMetadata metadata = 1;
     */
    metadata?: UploadGameServerFilesMetadata;
    /**
     * Complete tar archive containing all files
     *
     * @generated from field: bytes tar_data = 2;
     */
    tarData: Uint8Array;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UploadGameServerFilesRequest.
 * Use `create(UploadGameServerFilesRequestSchema)` to create a new message.
 */
export declare const UploadGameServerFilesRequestSchema: GenMessage<UploadGameServerFilesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UploadGameServerFilesMetadata
 */
export type UploadGameServerFilesMetadata = Message<"obiente.cloud.gameservers.v1.UploadGameServerFilesMetadata"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Directory path where files should be extracted (default: "/")
     *
     * @generated from field: string destination_path = 2;
     */
    destinationPath: string;
    /**
     * Metadata about files being uploaded
     *
     * @generated from field: repeated obiente.cloud.gameservers.v1.GameServerFileMetadata files = 3;
     */
    files: GameServerFileMetadata[];
    /**
     * If specified, upload files to this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UploadGameServerFilesMetadata.
 * Use `create(UploadGameServerFilesMetadataSchema)` to create a new message.
 */
export declare const UploadGameServerFilesMetadataSchema: GenMessage<UploadGameServerFilesMetadata>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GameServerFileMetadata
 */
export type GameServerFileMetadata = Message<"obiente.cloud.gameservers.v1.GameServerFileMetadata"> & {
    /**
     * File name
     *
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * File size in bytes
     *
     * @generated from field: int64 size = 2;
     */
    size: bigint;
    /**
     * Whether this is a directory
     *
     * @generated from field: bool is_directory = 3;
     */
    isDirectory: boolean;
    /**
     * Relative path within the archive
     *
     * @generated from field: string path = 4;
     */
    path: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GameServerFileMetadata.
 * Use `create(GameServerFileMetadataSchema)` to create a new message.
 */
export declare const GameServerFileMetadataSchema: GenMessage<GameServerFileMetadata>;
/**
 * @generated from message obiente.cloud.gameservers.v1.UploadGameServerFilesResponse
 */
export type UploadGameServerFilesResponse = Message<"obiente.cloud.gameservers.v1.UploadGameServerFilesResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
    /**
     * Number of files successfully uploaded
     *
     * @generated from field: int32 files_uploaded = 3;
     */
    filesUploaded: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.UploadGameServerFilesResponse.
 * Use `create(UploadGameServerFilesResponseSchema)` to create a new message.
 */
export declare const UploadGameServerFilesResponseSchema: GenMessage<UploadGameServerFilesResponse>;
/**
 * Chunked upload uses the shared payload/response for consistency across services.
 *
 * @generated from message obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesRequest
 */
export type ChunkUploadGameServerFilesRequest = Message<"obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: obiente.cloud.common.v1.ChunkedUploadPayload upload = 2;
     */
    upload?: ChunkedUploadPayload;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesRequest.
 * Use `create(ChunkUploadGameServerFilesRequestSchema)` to create a new message.
 */
export declare const ChunkUploadGameServerFilesRequestSchema: GenMessage<ChunkUploadGameServerFilesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesResponse
 */
export type ChunkUploadGameServerFilesResponse = Message<"obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesResponse"> & {
    /**
     * @generated from field: obiente.cloud.common.v1.ChunkedUploadResponsePayload result = 1;
     */
    result?: ChunkedUploadResponsePayload;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ChunkUploadGameServerFilesResponse.
 * Use `create(ChunkUploadGameServerFilesResponseSchema)` to create a new message.
 */
export declare const ChunkUploadGameServerFilesResponseSchema: GenMessage<ChunkUploadGameServerFilesResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerEntriesRequest
 */
export type DeleteGameServerEntriesRequest = Message<"obiente.cloud.gameservers.v1.DeleteGameServerEntriesRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: repeated string paths = 2;
     */
    paths: string[];
    /**
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
    /**
     * @generated from field: bool recursive = 4;
     */
    recursive: boolean;
    /**
     * @generated from field: bool force = 5;
     */
    force: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerEntriesRequest.
 * Use `create(DeleteGameServerEntriesRequestSchema)` to create a new message.
 */
export declare const DeleteGameServerEntriesRequestSchema: GenMessage<DeleteGameServerEntriesRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerEntriesError
 */
export type DeleteGameServerEntriesError = Message<"obiente.cloud.gameservers.v1.DeleteGameServerEntriesError"> & {
    /**
     * @generated from field: string path = 1;
     */
    path: string;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerEntriesError.
 * Use `create(DeleteGameServerEntriesErrorSchema)` to create a new message.
 */
export declare const DeleteGameServerEntriesErrorSchema: GenMessage<DeleteGameServerEntriesError>;
/**
 * @generated from message obiente.cloud.gameservers.v1.DeleteGameServerEntriesResponse
 */
export type DeleteGameServerEntriesResponse = Message<"obiente.cloud.gameservers.v1.DeleteGameServerEntriesResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: repeated string deleted_paths = 2;
     */
    deletedPaths: string[];
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.DeleteGameServerEntriesError errors = 3;
     */
    errors: DeleteGameServerEntriesError[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.DeleteGameServerEntriesResponse.
 * Use `create(DeleteGameServerEntriesResponseSchema)` to create a new message.
 */
export declare const DeleteGameServerEntriesResponseSchema: GenMessage<DeleteGameServerEntriesResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.RenameGameServerEntryRequest
 */
export type RenameGameServerEntryRequest = Message<"obiente.cloud.gameservers.v1.RenameGameServerEntryRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string source_path = 2;
     */
    sourcePath: string;
    /**
     * @generated from field: string target_path = 3;
     */
    targetPath: string;
    /**
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * @generated from field: bool overwrite = 5;
     */
    overwrite: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.RenameGameServerEntryRequest.
 * Use `create(RenameGameServerEntryRequestSchema)` to create a new message.
 */
export declare const RenameGameServerEntryRequestSchema: GenMessage<RenameGameServerEntryRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.RenameGameServerEntryResponse
 */
export type RenameGameServerEntryResponse = Message<"obiente.cloud.gameservers.v1.RenameGameServerEntryResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional obiente.cloud.gameservers.v1.GameServerFile entry = 2;
     */
    entry?: GameServerFile;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.RenameGameServerEntryResponse.
 * Use `create(RenameGameServerEntryResponseSchema)` to create a new message.
 */
export declare const RenameGameServerEntryResponseSchema: GenMessage<RenameGameServerEntryResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerEntryRequest
 */
export type CreateGameServerEntryRequest = Message<"obiente.cloud.gameservers.v1.CreateGameServerEntryRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string parent_path = 2;
     */
    parentPath: string;
    /**
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerEntryType type = 4;
     */
    type: GameServerEntryType;
    /**
     * Optional template identifier for seeded content (required for symlinks - target path)
     *
     * @generated from field: optional string template = 5;
     */
    template?: string;
    /**
     * @generated from field: optional string volume_name = 6;
     */
    volumeName?: string;
    /**
     * @generated from field: optional uint32 mode_octal = 7;
     */
    modeOctal?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerEntryRequest.
 * Use `create(CreateGameServerEntryRequestSchema)` to create a new message.
 */
export declare const CreateGameServerEntryRequestSchema: GenMessage<CreateGameServerEntryRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerEntryResponse
 */
export type CreateGameServerEntryResponse = Message<"obiente.cloud.gameservers.v1.CreateGameServerEntryResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.GameServerFile entry = 1;
     */
    entry?: GameServerFile;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerEntryResponse.
 * Use `create(CreateGameServerEntryResponseSchema)` to create a new message.
 */
export declare const CreateGameServerEntryResponseSchema: GenMessage<CreateGameServerEntryResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.WriteGameServerFileRequest
 */
export type WriteGameServerFileRequest = Message<"obiente.cloud.gameservers.v1.WriteGameServerFileRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string path = 2;
     */
    path: string;
    /**
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
    /**
     * @generated from field: string content = 4;
     */
    content: string;
    /**
     * "text" or "base64"
     *
     * @generated from field: string encoding = 5;
     */
    encoding: string;
    /**
     * @generated from field: bool create_if_missing = 6;
     */
    createIfMissing: boolean;
    /**
     * @generated from field: optional uint32 mode_octal = 7;
     */
    modeOctal?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.WriteGameServerFileRequest.
 * Use `create(WriteGameServerFileRequestSchema)` to create a new message.
 */
export declare const WriteGameServerFileRequestSchema: GenMessage<WriteGameServerFileRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.WriteGameServerFileResponse
 */
export type WriteGameServerFileResponse = Message<"obiente.cloud.gameservers.v1.WriteGameServerFileResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional obiente.cloud.gameservers.v1.GameServerFile entry = 2;
     */
    entry?: GameServerFile;
    /**
     * @generated from field: optional string error = 3;
     */
    error?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.WriteGameServerFileResponse.
 * Use `create(WriteGameServerFileResponseSchema)` to create a new message.
 */
export declare const WriteGameServerFileResponseSchema: GenMessage<WriteGameServerFileResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ExtractGameServerFileRequest
 */
export type ExtractGameServerFileRequest = Message<"obiente.cloud.gameservers.v1.ExtractGameServerFileRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Path to the zip file to extract (from common.ExtractServerFileRequest)
     *
     * @generated from field: string zip_path = 2;
     */
    zipPath: string;
    /**
     * Directory path where files should be extracted (from common.ExtractServerFileRequest)
     *
     * @generated from field: string destination_path = 3;
     */
    destinationPath: string;
    /**
     * If specified, extract to this volume instead of container filesystem (from common.ExtractServerFileRequest)
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ExtractGameServerFileRequest.
 * Use `create(ExtractGameServerFileRequestSchema)` to create a new message.
 */
export declare const ExtractGameServerFileRequestSchema: GenMessage<ExtractGameServerFileRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ExtractGameServerFileResponse
 */
export type ExtractGameServerFileResponse = Message<"obiente.cloud.gameservers.v1.ExtractGameServerFileResponse"> & {
    /**
     * From common.ExtractServerFileResponse
     *
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * From common.ExtractServerFileResponse
     *
     * @generated from field: optional string error = 2;
     */
    error?: string;
    /**
     * Number of files successfully extracted (from common.ExtractServerFileResponse)
     *
     * @generated from field: int32 files_extracted = 3;
     */
    filesExtracted: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ExtractGameServerFileResponse.
 * Use `create(ExtractGameServerFileResponseSchema)` to create a new message.
 */
export declare const ExtractGameServerFileResponseSchema: GenMessage<ExtractGameServerFileResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerFileArchiveRequest
 */
export type CreateGameServerFileArchiveRequest = Message<"obiente.cloud.gameservers.v1.CreateGameServerFileArchiveRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * Shared archive request
     *
     * @generated from field: obiente.cloud.common.v1.CreateServerFileArchiveRequest archive_request = 2;
     */
    archiveRequest?: CreateServerFileArchiveRequest;
    /**
     * If specified, create archive in this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerFileArchiveRequest.
 * Use `create(CreateGameServerFileArchiveRequestSchema)` to create a new message.
 */
export declare const CreateGameServerFileArchiveRequestSchema: GenMessage<CreateGameServerFileArchiveRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.CreateGameServerFileArchiveResponse
 */
export type CreateGameServerFileArchiveResponse = Message<"obiente.cloud.gameservers.v1.CreateGameServerFileArchiveResponse"> & {
    /**
     * Shared archive response
     *
     * @generated from field: obiente.cloud.common.v1.CreateServerFileArchiveResponse archive_response = 1;
     */
    archiveResponse?: CreateServerFileArchiveResponse;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.CreateGameServerFileArchiveResponse.
 * Use `create(CreateGameServerFileArchiveResponseSchema)` to create a new message.
 */
export declare const CreateGameServerFileArchiveResponseSchema: GenMessage<CreateGameServerFileArchiveResponse>;
/**
 * Minecraft player lookup messages
 *
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDRequest
 */
export type GetMinecraftPlayerUUIDRequest = Message<"obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDRequest"> & {
    /**
     * Minecraft username
     *
     * @generated from field: string username = 1;
     */
    username: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDRequest.
 * Use `create(GetMinecraftPlayerUUIDRequestSchema)` to create a new message.
 */
export declare const GetMinecraftPlayerUUIDRequestSchema: GenMessage<GetMinecraftPlayerUUIDRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDResponse
 */
export type GetMinecraftPlayerUUIDResponse = Message<"obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDResponse"> & {
    /**
     * Player UUID (dashed format, e.g., "550e8400-e29b-41d4-a716-446655440000")
     *
     * @generated from field: optional string uuid = 1;
     */
    uuid?: string;
    /**
     * Current username (may differ from requested username if changed)
     *
     * @generated from field: optional string name = 2;
     */
    name?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftPlayerUUIDResponse.
 * Use `create(GetMinecraftPlayerUUIDResponseSchema)` to create a new message.
 */
export declare const GetMinecraftPlayerUUIDResponseSchema: GenMessage<GetMinecraftPlayerUUIDResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileRequest
 */
export type GetMinecraftPlayerProfileRequest = Message<"obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileRequest"> & {
    /**
     * Player UUID (dashed or undashed format)
     *
     * @generated from field: string uuid = 1;
     */
    uuid: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileRequest.
 * Use `create(GetMinecraftPlayerProfileRequestSchema)` to create a new message.
 */
export declare const GetMinecraftPlayerProfileRequestSchema: GenMessage<GetMinecraftPlayerProfileRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileResponse
 */
export type GetMinecraftPlayerProfileResponse = Message<"obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileResponse"> & {
    /**
     * Player UUID (dashed format)
     *
     * @generated from field: optional string uuid = 1;
     */
    uuid?: string;
    /**
     * Current username
     *
     * @generated from field: optional string name = 2;
     */
    name?: string;
    /**
     * Avatar URL (from Crafatar)
     *
     * @generated from field: optional string avatar_url = 3;
     */
    avatarUrl?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftPlayerProfileResponse.
 * Use `create(GetMinecraftPlayerProfileResponseSchema)` to create a new message.
 */
export declare const GetMinecraftPlayerProfileResponseSchema: GenMessage<GetMinecraftPlayerProfileResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.MinecraftProject
 */
export type MinecraftProject = Message<"obiente.cloud.gameservers.v1.MinecraftProject"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string slug = 2;
     */
    slug: string;
    /**
     * @generated from field: string title = 3;
     */
    title: string;
    /**
     * @generated from field: string description = 4;
     */
    description: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.MinecraftProjectType project_type = 5;
     */
    projectType: MinecraftProjectType;
    /**
     * @generated from field: string icon_url = 6;
     */
    iconUrl: string;
    /**
     * @generated from field: repeated string categories = 7;
     */
    categories: string[];
    /**
     * @generated from field: repeated string loaders = 8;
     */
    loaders: string[];
    /**
     * @generated from field: repeated string game_versions = 9;
     */
    gameVersions: string[];
    /**
     * @generated from field: repeated string authors = 10;
     */
    authors: string[];
    /**
     * @generated from field: int64 downloads = 11;
     */
    downloads: bigint;
    /**
     * @generated from field: double rating = 12;
     */
    rating: number;
    /**
     * @generated from field: optional string latest_version_id = 13;
     */
    latestVersionId?: string;
    /**
     * @generated from field: optional string project_url = 14;
     */
    projectUrl?: string;
    /**
     * @generated from field: optional string source_url = 15;
     */
    sourceUrl?: string;
    /**
     * @generated from field: optional string issues_url = 16;
     */
    issuesUrl?: string;
    /**
     * Full body/description with markdown
     *
     * @generated from field: optional string body = 17;
     */
    body?: string;
    /**
     * Screenshot/image URLs
     *
     * @generated from field: repeated string gallery = 18;
     */
    gallery: string[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.MinecraftProject.
 * Use `create(MinecraftProjectSchema)` to create a new message.
 */
export declare const MinecraftProjectSchema: GenMessage<MinecraftProject>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ListMinecraftProjectsRequest
 */
export type ListMinecraftProjectsRequest = Message<"obiente.cloud.gameservers.v1.ListMinecraftProjectsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: optional string query = 2;
     */
    query?: string;
    /**
     * @generated from field: repeated string game_versions = 3;
     */
    gameVersions: string[];
    /**
     * @generated from field: repeated string loaders = 4;
     */
    loaders: string[];
    /**
     * @generated from field: repeated string categories = 5;
     */
    categories: string[];
    /**
     * @generated from field: optional string cursor = 6;
     */
    cursor?: string;
    /**
     * @generated from field: optional int32 limit = 7;
     */
    limit?: number;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.MinecraftProjectType project_type = 8;
     */
    projectType: MinecraftProjectType;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListMinecraftProjectsRequest.
 * Use `create(ListMinecraftProjectsRequestSchema)` to create a new message.
 */
export declare const ListMinecraftProjectsRequestSchema: GenMessage<ListMinecraftProjectsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.ListMinecraftProjectsResponse
 */
export type ListMinecraftProjectsResponse = Message<"obiente.cloud.gameservers.v1.ListMinecraftProjectsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.MinecraftProject projects = 1;
     */
    projects: MinecraftProject[];
    /**
     * @generated from field: bool has_more = 2;
     */
    hasMore: boolean;
    /**
     * @generated from field: optional string next_cursor = 3;
     */
    nextCursor?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.ListMinecraftProjectsResponse.
 * Use `create(ListMinecraftProjectsResponseSchema)` to create a new message.
 */
export declare const ListMinecraftProjectsResponseSchema: GenMessage<ListMinecraftProjectsResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.MinecraftProjectFile
 */
export type MinecraftProjectFile = Message<"obiente.cloud.gameservers.v1.MinecraftProjectFile"> & {
    /**
     * @generated from field: string filename = 1;
     */
    filename: string;
    /**
     * @generated from field: string url = 2;
     */
    url: string;
    /**
     * @generated from field: int64 size_bytes = 3;
     */
    sizeBytes: bigint;
    /**
     * @generated from field: map<string, string> hashes = 4;
     */
    hashes: {
        [key: string]: string;
    };
    /**
     * @generated from field: bool primary = 5;
     */
    primary: boolean;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.MinecraftProjectFile.
 * Use `create(MinecraftProjectFileSchema)` to create a new message.
 */
export declare const MinecraftProjectFileSchema: GenMessage<MinecraftProjectFile>;
/**
 * @generated from message obiente.cloud.gameservers.v1.MinecraftProjectVersion
 */
export type MinecraftProjectVersion = Message<"obiente.cloud.gameservers.v1.MinecraftProjectVersion"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string version_number = 3;
     */
    versionNumber: string;
    /**
     * @generated from field: repeated string game_versions = 4;
     */
    gameVersions: string[];
    /**
     * @generated from field: repeated string loaders = 5;
     */
    loaders: string[];
    /**
     * @generated from field: bool server_side_supported = 6;
     */
    serverSideSupported: boolean;
    /**
     * @generated from field: bool client_side_supported = 7;
     */
    clientSideSupported: boolean;
    /**
     * @generated from field: optional google.protobuf.Timestamp published_at = 8;
     */
    publishedAt?: Timestamp;
    /**
     * @generated from field: optional string changelog = 9;
     */
    changelog?: string;
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.MinecraftProjectFile files = 10;
     */
    files: MinecraftProjectFile[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.MinecraftProjectVersion.
 * Use `create(MinecraftProjectVersionSchema)` to create a new message.
 */
export declare const MinecraftProjectVersionSchema: GenMessage<MinecraftProjectVersion>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsRequest
 */
export type GetMinecraftProjectVersionsRequest = Message<"obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string project_id = 2;
     */
    projectId: string;
    /**
     * @generated from field: repeated string game_versions = 3;
     */
    gameVersions: string[];
    /**
     * @generated from field: repeated string loaders = 4;
     */
    loaders: string[];
    /**
     * @generated from field: obiente.cloud.gameservers.v1.MinecraftProjectType project_type = 5;
     */
    projectType: MinecraftProjectType;
    /**
     * @generated from field: optional int32 limit = 6;
     */
    limit?: number;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsRequest.
 * Use `create(GetMinecraftProjectVersionsRequestSchema)` to create a new message.
 */
export declare const GetMinecraftProjectVersionsRequestSchema: GenMessage<GetMinecraftProjectVersionsRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsResponse
 */
export type GetMinecraftProjectVersionsResponse = Message<"obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.gameservers.v1.MinecraftProjectVersion versions = 1;
     */
    versions: MinecraftProjectVersion[];
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftProjectVersionsResponse.
 * Use `create(GetMinecraftProjectVersionsResponseSchema)` to create a new message.
 */
export declare const GetMinecraftProjectVersionsResponseSchema: GenMessage<GetMinecraftProjectVersionsResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftProjectRequest
 */
export type GetMinecraftProjectRequest = Message<"obiente.cloud.gameservers.v1.GetMinecraftProjectRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string project_id = 2;
     */
    projectId: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftProjectRequest.
 * Use `create(GetMinecraftProjectRequestSchema)` to create a new message.
 */
export declare const GetMinecraftProjectRequestSchema: GenMessage<GetMinecraftProjectRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.GetMinecraftProjectResponse
 */
export type GetMinecraftProjectResponse = Message<"obiente.cloud.gameservers.v1.GetMinecraftProjectResponse"> & {
    /**
     * @generated from field: obiente.cloud.gameservers.v1.MinecraftProject project = 1;
     */
    project?: MinecraftProject;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.GetMinecraftProjectResponse.
 * Use `create(GetMinecraftProjectResponseSchema)` to create a new message.
 */
export declare const GetMinecraftProjectResponseSchema: GenMessage<GetMinecraftProjectResponse>;
/**
 * @generated from message obiente.cloud.gameservers.v1.InstallMinecraftProjectFileRequest
 */
export type InstallMinecraftProjectFileRequest = Message<"obiente.cloud.gameservers.v1.InstallMinecraftProjectFileRequest"> & {
    /**
     * @generated from field: string game_server_id = 1;
     */
    gameServerId: string;
    /**
     * @generated from field: string project_id = 2;
     */
    projectId: string;
    /**
     * @generated from field: string version_id = 3;
     */
    versionId: string;
    /**
     * @generated from field: obiente.cloud.gameservers.v1.MinecraftProjectType project_type = 4;
     */
    projectType: MinecraftProjectType;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.InstallMinecraftProjectFileRequest.
 * Use `create(InstallMinecraftProjectFileRequestSchema)` to create a new message.
 */
export declare const InstallMinecraftProjectFileRequestSchema: GenMessage<InstallMinecraftProjectFileRequest>;
/**
 * @generated from message obiente.cloud.gameservers.v1.InstallMinecraftProjectFileResponse
 */
export type InstallMinecraftProjectFileResponse = Message<"obiente.cloud.gameservers.v1.InstallMinecraftProjectFileResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: string filename = 2;
     */
    filename: string;
    /**
     * @generated from field: string installed_path = 3;
     */
    installedPath: string;
    /**
     * @generated from field: bool restart_required = 4;
     */
    restartRequired: boolean;
    /**
     * @generated from field: optional string message = 5;
     */
    message?: string;
};
/**
 * Describes the message obiente.cloud.gameservers.v1.InstallMinecraftProjectFileResponse.
 * Use `create(InstallMinecraftProjectFileResponseSchema)` to create a new message.
 */
export declare const InstallMinecraftProjectFileResponseSchema: GenMessage<InstallMinecraftProjectFileResponse>;
/**
 * GameType represents the type of game server
 *
 * @generated from enum obiente.cloud.gameservers.v1.GameType
 */
export declare enum GameType {
    /**
     * @generated from enum value: GAME_TYPE_UNSPECIFIED = 0;
     */
    GAME_TYPE_UNSPECIFIED = 0,
    /**
     * Minecraft (Java/Bedrock)
     *
     * @generated from enum value: MINECRAFT = 1;
     */
    MINECRAFT = 1,
    /**
     * Minecraft Java Edition
     *
     * @generated from enum value: MINECRAFT_JAVA = 2;
     */
    MINECRAFT_JAVA = 2,
    /**
     * Minecraft Bedrock Edition
     *
     * @generated from enum value: MINECRAFT_BEDROCK = 3;
     */
    MINECRAFT_BEDROCK = 3,
    /**
     * Valheim
     *
     * @generated from enum value: VALHEIM = 4;
     */
    VALHEIM = 4,
    /**
     * Terraria
     *
     * @generated from enum value: TERRARIA = 5;
     */
    TERRARIA = 5,
    /**
     * Rust
     *
     * @generated from enum value: RUST = 6;
     */
    RUST = 6,
    /**
     * Counter-Strike 2
     *
     * @generated from enum value: CS2 = 7;
     */
    CS2 = 7,
    /**
     * Team Fortress 2
     *
     * @generated from enum value: TF2 = 8;
     */
    TF2 = 8,
    /**
     * ARK: Survival Evolved
     *
     * @generated from enum value: ARK = 9;
     */
    ARK = 9,
    /**
     * Conan Exiles
     *
     * @generated from enum value: CONAN = 10;
     */
    CONAN = 10,
    /**
     * 7 Days to Die
     *
     * @generated from enum value: SEVEN_DAYS = 11;
     */
    SEVEN_DAYS = 11,
    /**
     * Factorio
     *
     * @generated from enum value: FACTORIO = 12;
     */
    FACTORIO = 12,
    /**
     * Space Engineers
     *
     * @generated from enum value: SPACED_ENGINEERS = 13;
     */
    SPACED_ENGINEERS = 13,
    /**
     * Other/Unknown game
     *
     * @generated from enum value: OTHER = 99;
     */
    OTHER = 99
}
/**
 * Describes the enum obiente.cloud.gameservers.v1.GameType.
 */
export declare const GameTypeSchema: GenEnum<GameType>;
/**
 * GameServerStatus represents the current status of a game server
 *
 * @generated from enum obiente.cloud.gameservers.v1.GameServerStatus
 */
export declare enum GameServerStatus {
    /**
     * @generated from enum value: GAME_SERVER_STATUS_UNSPECIFIED = 0;
     */
    GAME_SERVER_STATUS_UNSPECIFIED = 0,
    /**
     * Server created but not started
     *
     * @generated from enum value: CREATED = 1;
     */
    CREATED = 1,
    /**
     * Server is starting
     *
     * @generated from enum value: STARTING = 2;
     */
    STARTING = 2,
    /**
     * Server is running
     *
     * @generated from enum value: RUNNING = 3;
     */
    RUNNING = 3,
    /**
     * Server is stopping
     *
     * @generated from enum value: STOPPING = 4;
     */
    STOPPING = 4,
    /**
     * Server is stopped
     *
     * @generated from enum value: STOPPED = 5;
     */
    STOPPED = 5,
    /**
     * Server failed to start or crashed
     *
     * @generated from enum value: FAILED = 6;
     */
    FAILED = 6,
    /**
     * Server is restarting
     *
     * @generated from enum value: RESTARTING = 7;
     */
    RESTARTING = 7
}
/**
 * Describes the enum obiente.cloud.gameservers.v1.GameServerStatus.
 */
export declare const GameServerStatusSchema: GenEnum<GameServerStatus>;
/**
 * @generated from enum obiente.cloud.gameservers.v1.GameServerEntryType
 */
export declare enum GameServerEntryType {
    /**
     * @generated from enum value: GAME_SERVER_ENTRY_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: GAME_SERVER_ENTRY_TYPE_FILE = 1;
     */
    FILE = 1,
    /**
     * @generated from enum value: GAME_SERVER_ENTRY_TYPE_DIRECTORY = 2;
     */
    DIRECTORY = 2,
    /**
     * @generated from enum value: GAME_SERVER_ENTRY_TYPE_SYMLINK = 3;
     */
    SYMLINK = 3
}
/**
 * Describes the enum obiente.cloud.gameservers.v1.GameServerEntryType.
 */
export declare const GameServerEntryTypeSchema: GenEnum<GameServerEntryType>;
/**
 * Minecraft catalog messages
 *
 * @generated from enum obiente.cloud.gameservers.v1.MinecraftProjectType
 */
export declare enum MinecraftProjectType {
    /**
     * @generated from enum value: MINECRAFT_PROJECT_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: MINECRAFT_PROJECT_TYPE_MOD = 1;
     */
    MOD = 1,
    /**
     * @generated from enum value: MINECRAFT_PROJECT_TYPE_PLUGIN = 2;
     */
    PLUGIN = 2
}
/**
 * Describes the enum obiente.cloud.gameservers.v1.MinecraftProjectType.
 */
export declare const MinecraftProjectTypeSchema: GenEnum<MinecraftProjectType>;
/**
 * @generated from service obiente.cloud.gameservers.v1.GameServerService
 */
export declare const GameServerService: GenService<{
    /**
     * List organization game servers
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.ListGameServers
     */
    listGameServers: {
        methodKind: "unary";
        input: typeof ListGameServersRequestSchema;
        output: typeof ListGameServersResponseSchema;
    };
    /**
     * Create new game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.CreateGameServer
     */
    createGameServer: {
        methodKind: "unary";
        input: typeof CreateGameServerRequestSchema;
        output: typeof CreateGameServerResponseSchema;
    };
    /**
     * Get game server details
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServer
     */
    getGameServer: {
        methodKind: "unary";
        input: typeof GetGameServerRequestSchema;
        output: typeof GetGameServerResponseSchema;
    };
    /**
     * Update game server configuration
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.UpdateGameServer
     */
    updateGameServer: {
        methodKind: "unary";
        input: typeof UpdateGameServerRequestSchema;
        output: typeof UpdateGameServerResponseSchema;
    };
    /**
     * Delete game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.DeleteGameServer
     */
    deleteGameServer: {
        methodKind: "unary";
        input: typeof DeleteGameServerRequestSchema;
        output: typeof DeleteGameServerResponseSchema;
    };
    /**
     * Start a stopped game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.StartGameServer
     */
    startGameServer: {
        methodKind: "unary";
        input: typeof StartGameServerRequestSchema;
        output: typeof StartGameServerResponseSchema;
    };
    /**
     * Stop a running game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.StopGameServer
     */
    stopGameServer: {
        methodKind: "unary";
        input: typeof StopGameServerRequestSchema;
        output: typeof StopGameServerResponseSchema;
    };
    /**
     * Restart a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.RestartGameServer
     */
    restartGameServer: {
        methodKind: "unary";
        input: typeof RestartGameServerRequestSchema;
        output: typeof RestartGameServerResponseSchema;
    };
    /**
     * Get HTTP routing rules for a game server (Traefik)
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerHTTPRoutes
     */
    getGameServerHTTPRoutes: {
        methodKind: "unary";
        input: typeof GetGameServerHTTPRoutesRequestSchema;
        output: typeof GetGameServerHTTPRoutesResponseSchema;
    };
    /**
     * Create or update an HTTP routing rule for a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.UpsertGameServerHTTPRoute
     */
    upsertGameServerHTTPRoute: {
        methodKind: "unary";
        input: typeof UpsertGameServerHTTPRouteRequestSchema;
        output: typeof UpsertGameServerHTTPRouteResponseSchema;
    };
    /**
     * Delete an HTTP routing rule for a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.DeleteGameServerHTTPRoute
     */
    deleteGameServerHTTPRoute: {
        methodKind: "unary";
        input: typeof DeleteGameServerHTTPRouteRequestSchema;
        output: typeof DeleteGameServerHTTPRouteResponseSchema;
    };
    /**
     * Get DNS TXT verification token for a custom game server domain
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerDomainVerificationToken
     */
    getGameServerDomainVerificationToken: {
        methodKind: "unary";
        input: typeof GetGameServerDomainVerificationTokenRequestSchema;
        output: typeof GetGameServerDomainVerificationTokenResponseSchema;
    };
    /**
     * Verify ownership of a custom game server domain
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.VerifyGameServerDomain
     */
    verifyGameServerDomain: {
        methodKind: "unary";
        input: typeof VerifyGameServerDomainRequestSchema;
        output: typeof VerifyGameServerDomainResponseSchema;
    };
    /**
     * Stream game server status updates
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.StreamGameServerStatus
     */
    streamGameServerStatus: {
        methodKind: "server_streaming";
        input: typeof StreamGameServerStatusRequestSchema;
        output: typeof GameServerStatusUpdateSchema;
    };
    /**
     * Get game server logs
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerLogs
     */
    getGameServerLogs: {
        methodKind: "unary";
        input: typeof GetGameServerLogsRequestSchema;
        output: typeof GetGameServerLogsResponseSchema;
    };
    /**
     * Stream game server logs (tail/follow)
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.StreamGameServerLogs
     */
    streamGameServerLogs: {
        methodKind: "server_streaming";
        input: typeof StreamGameServerLogsRequestSchema;
        output: typeof GameServerLogLineSchema;
    };
    /**
     * Get game server metrics (real-time or historical)
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerMetrics
     */
    getGameServerMetrics: {
        methodKind: "unary";
        input: typeof GetGameServerMetricsRequestSchema;
        output: typeof GetGameServerMetricsResponseSchema;
    };
    /**
     * Stream real-time game server metrics
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.StreamGameServerMetrics
     */
    streamGameServerMetrics: {
        methodKind: "server_streaming";
        input: typeof StreamGameServerMetricsRequestSchema;
        output: typeof GameServerMetricSchema;
    };
    /**
     * Get aggregated usage for a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerUsage
     */
    getGameServerUsage: {
        methodKind: "unary";
        input: typeof GetGameServerUsageRequestSchema;
        output: typeof GetGameServerUsageResponseSchema;
    };
    /**
     * File system operations
     * List files in a game server container or volume
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.ListGameServerFiles
     */
    listGameServerFiles: {
        methodKind: "unary";
        input: typeof ListGameServerFilesRequestSchema;
        output: typeof ListGameServerFilesResponseSchema;
    };
    /**
     * Search for files in a game server container or volume (recursive search)
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.SearchGameServerFiles
     */
    searchGameServerFiles: {
        methodKind: "unary";
        input: typeof SearchGameServerFilesRequestSchema;
        output: typeof SearchGameServerFilesResponseSchema;
    };
    /**
     * Get file contents from a game server container or volume
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetGameServerFile
     */
    getGameServerFile: {
        methodKind: "unary";
        input: typeof GetGameServerFileRequestSchema;
        output: typeof GetGameServerFileResponseSchema;
    };
    /**
     * Upload files to a game server container or volume
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.UploadGameServerFiles
     */
    uploadGameServerFiles: {
        methodKind: "unary";
        input: typeof UploadGameServerFilesRequestSchema;
        output: typeof UploadGameServerFilesResponseSchema;
    };
    /**
     * Chunk-based file upload that streams without buffering large payloads in memory.
     * Uses shared chunk payload for consistency across resources.
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.ChunkUploadGameServerFiles
     */
    chunkUploadGameServerFiles: {
        methodKind: "unary";
        input: typeof ChunkUploadGameServerFilesRequestSchema;
        output: typeof ChunkUploadGameServerFilesResponseSchema;
    };
    /**
     * Delete files or directories from a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.DeleteGameServerEntries
     */
    deleteGameServerEntries: {
        methodKind: "unary";
        input: typeof DeleteGameServerEntriesRequestSchema;
        output: typeof DeleteGameServerEntriesResponseSchema;
    };
    /**
     * Create a file, directory, or symlink in a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.CreateGameServerEntry
     */
    createGameServerEntry: {
        methodKind: "unary";
        input: typeof CreateGameServerEntryRequestSchema;
        output: typeof CreateGameServerEntryResponseSchema;
    };
    /**
     * Write or update file contents in a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.WriteGameServerFile
     */
    writeGameServerFile: {
        methodKind: "unary";
        input: typeof WriteGameServerFileRequestSchema;
        output: typeof WriteGameServerFileResponseSchema;
    };
    /**
     * Rename a file or directory in a game server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.RenameGameServerEntry
     */
    renameGameServerEntry: {
        methodKind: "unary";
        input: typeof RenameGameServerEntryRequestSchema;
        output: typeof RenameGameServerEntryResponseSchema;
    };
    /**
     * Extract a zip file to a destination directory
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.ExtractGameServerFile
     */
    extractGameServerFile: {
        methodKind: "unary";
        input: typeof ExtractGameServerFileRequestSchema;
        output: typeof ExtractGameServerFileResponseSchema;
    };
    /**
     * Create a zip archive from files or folders
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.CreateGameServerFileArchive
     */
    createGameServerFileArchive: {
        methodKind: "unary";
        input: typeof CreateGameServerFileArchiveRequestSchema;
        output: typeof CreateGameServerFileArchiveResponseSchema;
    };
    /**
     * Minecraft player lookup (proxies Mojang API to avoid CORS)
     * Get player UUID from username
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetMinecraftPlayerUUID
     */
    getMinecraftPlayerUUID: {
        methodKind: "unary";
        input: typeof GetMinecraftPlayerUUIDRequestSchema;
        output: typeof GetMinecraftPlayerUUIDResponseSchema;
    };
    /**
     * Get player profile (name, UUID) from UUID
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetMinecraftPlayerProfile
     */
    getMinecraftPlayerProfile: {
        methodKind: "unary";
        input: typeof GetMinecraftPlayerProfileRequestSchema;
        output: typeof GetMinecraftPlayerProfileResponseSchema;
    };
    /**
     * Minecraft content catalog (Modrinth integration)
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.ListMinecraftProjects
     */
    listMinecraftProjects: {
        methodKind: "unary";
        input: typeof ListMinecraftProjectsRequestSchema;
        output: typeof ListMinecraftProjectsResponseSchema;
    };
    /**
     * Retrieve available versions/files for a specific project
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetMinecraftProjectVersions
     */
    getMinecraftProjectVersions: {
        methodKind: "unary";
        input: typeof GetMinecraftProjectVersionsRequestSchema;
        output: typeof GetMinecraftProjectVersionsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.GetMinecraftProject
     */
    getMinecraftProject: {
        methodKind: "unary";
        input: typeof GetMinecraftProjectRequestSchema;
        output: typeof GetMinecraftProjectResponseSchema;
    };
    /**
     * Download and install a Minecraft mod/plugin directly onto the server
     *
     * @generated from rpc obiente.cloud.gameservers.v1.GameServerService.InstallMinecraftProjectFile
     */
    installMinecraftProjectFile: {
        methodKind: "unary";
        input: typeof InstallMinecraftProjectFileRequestSchema;
        output: typeof InstallMinecraftProjectFileResponseSchema;
    };
}>;
