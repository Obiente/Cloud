import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { ChunkedUploadPayload, ChunkedUploadResponsePayload, CreateServerFileArchiveRequest, CreateServerFileArchiveResponse, LogLevel, Pagination } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/deployments/v1/deployment_service.proto.
 */
export declare const file_obiente_cloud_deployments_v1_deployment_service: GenFile;
/**
 * @generated from message obiente.cloud.deployments.v1.ListDeploymentsRequest
 */
export type ListDeploymentsRequest = Message<"obiente.cloud.deployments.v1.ListDeploymentsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: optional obiente.cloud.deployments.v1.DeploymentStatus status = 2;
     */
    status?: DeploymentStatus;
    /**
     * @generated from field: int32 page = 3;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 4;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListDeploymentsRequest.
 * Use `create(ListDeploymentsRequestSchema)` to create a new message.
 */
export declare const ListDeploymentsRequestSchema: GenMessage<ListDeploymentsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListDeploymentsResponse
 */
export type ListDeploymentsResponse = Message<"obiente.cloud.deployments.v1.ListDeploymentsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.Deployment deployments = 1;
     */
    deployments: Deployment[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListDeploymentsResponse.
 * Use `create(ListDeploymentsResponseSchema)` to create a new message.
 */
export declare const ListDeploymentsResponseSchema: GenMessage<ListDeploymentsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateDeploymentRequest
 */
export type CreateDeploymentRequest = Message<"obiente.cloud.deployments.v1.CreateDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * Environment (production/staging/development)
     *
     * @generated from field: obiente.cloud.deployments.v1.Environment environment = 3;
     */
    environment: Environment;
    /**
     * Optional groups/labels for organizing deployments
     *
     * @generated from field: repeated string groups = 4;
     */
    groups: string[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateDeploymentRequest.
 * Use `create(CreateDeploymentRequestSchema)` to create a new message.
 */
export declare const CreateDeploymentRequestSchema: GenMessage<CreateDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateDeploymentResponse
 */
export type CreateDeploymentResponse = Message<"obiente.cloud.deployments.v1.CreateDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateDeploymentResponse.
 * Use `create(CreateDeploymentResponseSchema)` to create a new message.
 */
export declare const CreateDeploymentResponseSchema: GenMessage<CreateDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentRequest
 */
export type GetDeploymentRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentRequest.
 * Use `create(GetDeploymentRequestSchema)` to create a new message.
 */
export declare const GetDeploymentRequestSchema: GenMessage<GetDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentResponse
 */
export type GetDeploymentResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentResponse.
 * Use `create(GetDeploymentResponseSchema)` to create a new message.
 */
export declare const GetDeploymentResponseSchema: GenMessage<GetDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentRequest
 */
export type UpdateDeploymentRequest = Message<"obiente.cloud.deployments.v1.UpdateDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: optional string name = 3;
     */
    name?: string;
    /**
     * @generated from field: optional string repository_url = 4;
     */
    repositoryUrl?: string;
    /**
     * GitHub integration ID used for this deployment
     *
     * @generated from field: optional string github_integration_id = 14;
     */
    githubIntegrationId?: string;
    /**
     * @generated from field: optional string branch = 5;
     */
    branch?: string;
    /**
     * @generated from field: optional string build_command = 6;
     */
    buildCommand?: string;
    /**
     * @generated from field: optional string install_command = 7;
     */
    installCommand?: string;
    /**
     * Start command for running the application
     *
     * @generated from field: optional string start_command = 17;
     */
    startCommand?: string;
    /**
     * Path to Dockerfile (relative to repo root)
     *
     * @generated from field: optional string dockerfile_path = 12;
     */
    dockerfilePath?: string;
    /**
     * Path to compose file (relative to repo root)
     *
     * @generated from field: optional string compose_file_path = 13;
     */
    composeFilePath?: string;
    /**
     * Working directory for build (relative to repo root, defaults to ".")
     *
     * @generated from field: optional string build_path = 18;
     */
    buildPath?: string;
    /**
     * Path to built output files (relative to repo root, auto-detected if empty)
     *
     * @generated from field: optional string build_output_path = 19;
     */
    buildOutputPath?: string;
    /**
     * Use nginx for static deployments
     *
     * @generated from field: optional bool use_nginx = 20;
     */
    useNginx?: boolean;
    /**
     * Custom nginx configuration (optional, uses default if empty)
     *
     * @generated from field: optional string nginx_config = 21;
     */
    nginxConfig?: string;
    /**
     * @generated from field: optional string domain = 8;
     */
    domain?: string;
    /**
     * @generated from field: repeated string custom_domains = 9;
     */
    customDomains: string[];
    /**
     * @generated from field: optional int32 port = 10;
     */
    port?: number;
    /**
     * build strategy enum
     *
     * @generated from field: optional obiente.cloud.deployments.v1.BuildStrategy build_strategy = 11;
     */
    buildStrategy?: BuildStrategy;
    /**
     * Environment (production/staging/development)
     *
     * @generated from field: optional obiente.cloud.deployments.v1.Environment environment = 15;
     */
    environment?: Environment;
    /**
     * Optional groups/labels for organizing deployments
     *
     * @generated from field: repeated string groups = 16;
     */
    groups: string[];
    /**
     * CPU limit in cores
     *
     * @generated from field: optional double cpu_limit = 22;
     */
    cpuLimit?: number;
    /**
     * Memory limit in MB
     *
     * @generated from field: optional int64 memory_limit = 23;
     */
    memoryLimit?: bigint;
    /**
     * Health check configuration
     *
     * Type of health check
     *
     * @generated from field: optional obiente.cloud.deployments.v1.HealthCheckType healthcheck_type = 24;
     */
    healthcheckType?: HealthCheckType;
    /**
     * Port to check (if different from main port)
     *
     * @generated from field: optional int32 healthcheck_port = 25;
     */
    healthcheckPort?: number;
    /**
     * HTTP path (default: "/", used with HEALTHCHECK_HTTP)
     *
     * @generated from field: optional string healthcheck_path = 26;
     */
    healthcheckPath?: string;
    /**
     * Expected HTTP status code (default: 200, used with HEALTHCHECK_HTTP)
     *
     * @generated from field: optional int32 healthcheck_expected_status = 27;
     */
    healthcheckExpectedStatus?: number;
    /**
     * Custom command (used with HEALTHCHECK_CUSTOM)
     *
     * @generated from field: optional string healthcheck_custom_command = 28;
     */
    healthcheckCustomCommand?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentRequest.
 * Use `create(UpdateDeploymentRequestSchema)` to create a new message.
 */
export declare const UpdateDeploymentRequestSchema: GenMessage<UpdateDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentResponse
 */
export type UpdateDeploymentResponse = Message<"obiente.cloud.deployments.v1.UpdateDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentResponse.
 * Use `create(UpdateDeploymentResponseSchema)` to create a new message.
 */
export declare const UpdateDeploymentResponseSchema: GenMessage<UpdateDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.TriggerDeploymentRequest
 */
export type TriggerDeploymentRequest = Message<"obiente.cloud.deployments.v1.TriggerDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.TriggerDeploymentRequest.
 * Use `create(TriggerDeploymentRequestSchema)` to create a new message.
 */
export declare const TriggerDeploymentRequestSchema: GenMessage<TriggerDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.TriggerDeploymentResponse
 */
export type TriggerDeploymentResponse = Message<"obiente.cloud.deployments.v1.TriggerDeploymentResponse"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string status = 2;
     */
    status: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.TriggerDeploymentResponse.
 * Use `create(TriggerDeploymentResponseSchema)` to create a new message.
 */
export declare const TriggerDeploymentResponseSchema: GenMessage<TriggerDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamDeploymentStatusRequest
 */
export type StreamDeploymentStatusRequest = Message<"obiente.cloud.deployments.v1.StreamDeploymentStatusRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamDeploymentStatusRequest.
 * Use `create(StreamDeploymentStatusRequestSchema)` to create a new message.
 */
export declare const StreamDeploymentStatusRequestSchema: GenMessage<StreamDeploymentStatusRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeploymentStatusUpdate
 */
export type DeploymentStatusUpdate = Message<"obiente.cloud.deployments.v1.DeploymentStatusUpdate"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: obiente.cloud.deployments.v1.DeploymentStatus status = 2;
     */
    status: DeploymentStatus;
    /**
     * @generated from field: string health_status = 3;
     */
    healthStatus: string;
    /**
     * @generated from field: optional string message = 4;
     */
    message?: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 5;
     */
    timestamp?: Timestamp;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeploymentStatusUpdate.
 * Use `create(DeploymentStatusUpdateSchema)` to create a new message.
 */
export declare const DeploymentStatusUpdateSchema: GenMessage<DeploymentStatusUpdate>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentLogsRequest
 */
export type GetDeploymentLogsRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: optional int32 lines = 3;
     */
    lines?: number;
    /**
     * @generated from field: optional bool follow = 4;
     */
    follow?: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentLogsRequest.
 * Use `create(GetDeploymentLogsRequestSchema)` to create a new message.
 */
export declare const GetDeploymentLogsRequestSchema: GenMessage<GetDeploymentLogsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentLogsResponse
 */
export type GetDeploymentLogsResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentLogsResponse"> & {
    /**
     * @generated from field: repeated string logs = 1;
     */
    logs: string[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentLogsResponse.
 * Use `create(GetDeploymentLogsResponseSchema)` to create a new message.
 */
export declare const GetDeploymentLogsResponseSchema: GenMessage<GetDeploymentLogsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamDeploymentLogsRequest
 */
export type StreamDeploymentLogsRequest = Message<"obiente.cloud.deployments.v1.StreamDeploymentLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * number of lines to tail before following
     *
     * @generated from field: optional int32 tail = 3;
     */
    tail?: number;
    /**
     * Optional: stream logs from a specific container. If not specified, uses first container.
     *
     * @generated from field: optional string container_id = 4;
     */
    containerId?: string;
    /**
     * Optional: stream logs from a specific service. If container_id is specified, this is ignored.
     *
     * @generated from field: optional string service_name = 5;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamDeploymentLogsRequest.
 * Use `create(StreamDeploymentLogsRequestSchema)` to create a new message.
 */
export declare const StreamDeploymentLogsRequestSchema: GenMessage<StreamDeploymentLogsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamBuildLogsRequest
 */
export type StreamBuildLogsRequest = Message<"obiente.cloud.deployments.v1.StreamBuildLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamBuildLogsRequest.
 * Use `create(StreamBuildLogsRequestSchema)` to create a new message.
 */
export declare const StreamBuildLogsRequestSchema: GenMessage<StreamBuildLogsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeploymentLogLine
 */
export type DeploymentLogLine = Message<"obiente.cloud.deployments.v1.DeploymentLogLine"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string line = 2;
     */
    line: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 3;
     */
    timestamp?: Timestamp;
    /**
     * Deprecated: Use log_level instead. Kept for backward compatibility.
     *
     * @generated from field: bool stderr = 4;
     */
    stderr: boolean;
    /**
     * Log level (INFO, WARN, ERROR, DEBUG, TRACE)
     *
     * @generated from field: obiente.cloud.common.v1.LogLevel log_level = 5;
     */
    logLevel: LogLevel;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeploymentLogLine.
 * Use `create(DeploymentLogLineSchema)` to create a new message.
 */
export declare const DeploymentLogLineSchema: GenMessage<DeploymentLogLine>;
/**
 * @generated from message obiente.cloud.deployments.v1.StartDeploymentRequest
 */
export type StartDeploymentRequest = Message<"obiente.cloud.deployments.v1.StartDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StartDeploymentRequest.
 * Use `create(StartDeploymentRequestSchema)` to create a new message.
 */
export declare const StartDeploymentRequestSchema: GenMessage<StartDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StartDeploymentResponse
 */
export type StartDeploymentResponse = Message<"obiente.cloud.deployments.v1.StartDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StartDeploymentResponse.
 * Use `create(StartDeploymentResponseSchema)` to create a new message.
 */
export declare const StartDeploymentResponseSchema: GenMessage<StartDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StopDeploymentRequest
 */
export type StopDeploymentRequest = Message<"obiente.cloud.deployments.v1.StopDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StopDeploymentRequest.
 * Use `create(StopDeploymentRequestSchema)` to create a new message.
 */
export declare const StopDeploymentRequestSchema: GenMessage<StopDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StopDeploymentResponse
 */
export type StopDeploymentResponse = Message<"obiente.cloud.deployments.v1.StopDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StopDeploymentResponse.
 * Use `create(StopDeploymentResponseSchema)` to create a new message.
 */
export declare const StopDeploymentResponseSchema: GenMessage<StopDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteDeploymentRequest
 */
export type DeleteDeploymentRequest = Message<"obiente.cloud.deployments.v1.DeleteDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteDeploymentRequest.
 * Use `create(DeleteDeploymentRequestSchema)` to create a new message.
 */
export declare const DeleteDeploymentRequestSchema: GenMessage<DeleteDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteDeploymentResponse
 */
export type DeleteDeploymentResponse = Message<"obiente.cloud.deployments.v1.DeleteDeploymentResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteDeploymentResponse.
 * Use `create(DeleteDeploymentResponseSchema)` to create a new message.
 */
export declare const DeleteDeploymentResponseSchema: GenMessage<DeleteDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.RestartDeploymentRequest
 */
export type RestartDeploymentRequest = Message<"obiente.cloud.deployments.v1.RestartDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RestartDeploymentRequest.
 * Use `create(RestartDeploymentRequestSchema)` to create a new message.
 */
export declare const RestartDeploymentRequestSchema: GenMessage<RestartDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.RestartDeploymentResponse
 */
export type RestartDeploymentResponse = Message<"obiente.cloud.deployments.v1.RestartDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RestartDeploymentResponse.
 * Use `create(RestartDeploymentResponseSchema)` to create a new message.
 */
export declare const RestartDeploymentResponseSchema: GenMessage<RestartDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.ScaleDeploymentRequest
 */
export type ScaleDeploymentRequest = Message<"obiente.cloud.deployments.v1.ScaleDeploymentRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: int32 replicas = 3;
     */
    replicas: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ScaleDeploymentRequest.
 * Use `create(ScaleDeploymentRequestSchema)` to create a new message.
 */
export declare const ScaleDeploymentRequestSchema: GenMessage<ScaleDeploymentRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ScaleDeploymentResponse
 */
export type ScaleDeploymentResponse = Message<"obiente.cloud.deployments.v1.ScaleDeploymentResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ScaleDeploymentResponse.
 * Use `create(ScaleDeploymentResponseSchema)` to create a new message.
 */
export declare const ScaleDeploymentResponseSchema: GenMessage<ScaleDeploymentResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentEnvVarsRequest
 */
export type GetDeploymentEnvVarsRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentEnvVarsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentEnvVarsRequest.
 * Use `create(GetDeploymentEnvVarsRequestSchema)` to create a new message.
 */
export declare const GetDeploymentEnvVarsRequestSchema: GenMessage<GetDeploymentEnvVarsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentEnvVarsResponse
 */
export type GetDeploymentEnvVarsResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentEnvVarsResponse"> & {
    /**
     * Raw .env file content (preserves comments and formatting)
     *
     * @generated from field: string env_file_content = 1;
     */
    envFileContent: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentEnvVarsResponse.
 * Use `create(GetDeploymentEnvVarsResponseSchema)` to create a new message.
 */
export declare const GetDeploymentEnvVarsResponseSchema: GenMessage<GetDeploymentEnvVarsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsRequest
 */
export type UpdateDeploymentEnvVarsRequest = Message<"obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Raw .env file content (preserves comments and formatting)
     *
     * @generated from field: string env_file_content = 3;
     */
    envFileContent: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsRequest.
 * Use `create(UpdateDeploymentEnvVarsRequestSchema)` to create a new message.
 */
export declare const UpdateDeploymentEnvVarsRequestSchema: GenMessage<UpdateDeploymentEnvVarsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsResponse
 */
export type UpdateDeploymentEnvVarsResponse = Message<"obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentEnvVarsResponse.
 * Use `create(UpdateDeploymentEnvVarsResponseSchema)` to create a new message.
 */
export declare const UpdateDeploymentEnvVarsResponseSchema: GenMessage<UpdateDeploymentEnvVarsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentComposeRequest
 */
export type GetDeploymentComposeRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentComposeRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentComposeRequest.
 * Use `create(GetDeploymentComposeRequestSchema)` to create a new message.
 */
export declare const GetDeploymentComposeRequestSchema: GenMessage<GetDeploymentComposeRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentComposeResponse
 */
export type GetDeploymentComposeResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentComposeResponse"> & {
    /**
     * @generated from field: string compose_yaml = 1;
     */
    composeYaml: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentComposeResponse.
 * Use `create(GetDeploymentComposeResponseSchema)` to create a new message.
 */
export declare const GetDeploymentComposeResponseSchema: GenMessage<GetDeploymentComposeResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.ValidateDeploymentComposeRequest
 */
export type ValidateDeploymentComposeRequest = Message<"obiente.cloud.deployments.v1.ValidateDeploymentComposeRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string compose_yaml = 3;
     */
    composeYaml: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ValidateDeploymentComposeRequest.
 * Use `create(ValidateDeploymentComposeRequestSchema)` to create a new message.
 */
export declare const ValidateDeploymentComposeRequestSchema: GenMessage<ValidateDeploymentComposeRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ValidateDeploymentComposeResponse
 */
export type ValidateDeploymentComposeResponse = Message<"obiente.cloud.deployments.v1.ValidateDeploymentComposeResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.ComposeValidationError validation_errors = 1;
     */
    validationErrors: ComposeValidationError[];
    /**
     * Deprecated: Use validation_errors instead
     *
     * @generated from field: optional string validation_error = 2;
     */
    validationError?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ValidateDeploymentComposeResponse.
 * Use `create(ValidateDeploymentComposeResponseSchema)` to create a new message.
 */
export declare const ValidateDeploymentComposeResponseSchema: GenMessage<ValidateDeploymentComposeResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentComposeRequest
 */
export type UpdateDeploymentComposeRequest = Message<"obiente.cloud.deployments.v1.UpdateDeploymentComposeRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string compose_yaml = 3;
     */
    composeYaml: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentComposeRequest.
 * Use `create(UpdateDeploymentComposeRequestSchema)` to create a new message.
 */
export declare const UpdateDeploymentComposeRequestSchema: GenMessage<UpdateDeploymentComposeRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentComposeResponse
 */
export type UpdateDeploymentComposeResponse = Message<"obiente.cloud.deployments.v1.UpdateDeploymentComposeResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
    /**
     * Deprecated: Use validation_errors instead
     *
     * @generated from field: optional string validation_error = 2;
     */
    validationError?: string;
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.ComposeValidationError validation_errors = 3;
     */
    validationErrors: ComposeValidationError[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentComposeResponse.
 * Use `create(UpdateDeploymentComposeResponseSchema)` to create a new message.
 */
export declare const UpdateDeploymentComposeResponseSchema: GenMessage<UpdateDeploymentComposeResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.ComposeValidationError
 */
export type ComposeValidationError = Message<"obiente.cloud.deployments.v1.ComposeValidationError"> & {
    /**
     * 1-based line number
     *
     * @generated from field: int32 line = 1;
     */
    line: number;
    /**
     * 1-based column number
     *
     * @generated from field: int32 column = 2;
     */
    column: number;
    /**
     * @generated from field: string message = 3;
     */
    message: string;
    /**
     * "error" or "warning"
     *
     * @generated from field: string severity = 4;
     */
    severity: string;
    /**
     * Start line for multi-line errors
     *
     * @generated from field: int32 start_line = 5;
     */
    startLine: number;
    /**
     * End line for multi-line errors
     *
     * @generated from field: int32 end_line = 6;
     */
    endLine: number;
    /**
     * Start column
     *
     * @generated from field: int32 start_column = 7;
     */
    startColumn: number;
    /**
     * End column
     *
     * @generated from field: int32 end_column = 8;
     */
    endColumn: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ComposeValidationError.
 * Use `create(ComposeValidationErrorSchema)` to create a new message.
 */
export declare const ComposeValidationErrorSchema: GenMessage<ComposeValidationError>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListGitHubReposRequest
 */
export type ListGitHubReposRequest = Message<"obiente.cloud.deployments.v1.ListGitHubReposRequest"> & {
    /**
     * Organization ID to use for GitHub token (optional, will use user token if not provided)
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: use specific GitHub integration ID
     *
     * @generated from field: string integration_id = 2;
     */
    integrationId: string;
    /**
     * @generated from field: int32 page = 3;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 4;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListGitHubReposRequest.
 * Use `create(ListGitHubReposRequestSchema)` to create a new message.
 */
export declare const ListGitHubReposRequestSchema: GenMessage<ListGitHubReposRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GitHubRepo
 */
export type GitHubRepo = Message<"obiente.cloud.deployments.v1.GitHubRepo"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string full_name = 3;
     */
    fullName: string;
    /**
     * @generated from field: string description = 4;
     */
    description: string;
    /**
     * @generated from field: string url = 5;
     */
    url: string;
    /**
     * @generated from field: bool is_private = 6;
     */
    isPrivate: boolean;
    /**
     * @generated from field: string default_branch = 7;
     */
    defaultBranch: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GitHubRepo.
 * Use `create(GitHubRepoSchema)` to create a new message.
 */
export declare const GitHubRepoSchema: GenMessage<GitHubRepo>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListGitHubReposResponse
 */
export type ListGitHubReposResponse = Message<"obiente.cloud.deployments.v1.ListGitHubReposResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.GitHubRepo repos = 1;
     */
    repos: GitHubRepo[];
    /**
     * @generated from field: int32 total = 2;
     */
    total: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListGitHubReposResponse.
 * Use `create(ListGitHubReposResponseSchema)` to create a new message.
 */
export declare const ListGitHubReposResponseSchema: GenMessage<ListGitHubReposResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetGitHubBranchesRequest
 */
export type GetGitHubBranchesRequest = Message<"obiente.cloud.deployments.v1.GetGitHubBranchesRequest"> & {
    /**
     * Organization ID to use for GitHub token (optional, will use user token if not provided)
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: use specific GitHub integration ID
     *
     * @generated from field: string integration_id = 2;
     */
    integrationId: string;
    /**
     * e.g., "owner/repo"
     *
     * @generated from field: string repo_full_name = 3;
     */
    repoFullName: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetGitHubBranchesRequest.
 * Use `create(GetGitHubBranchesRequestSchema)` to create a new message.
 */
export declare const GetGitHubBranchesRequestSchema: GenMessage<GetGitHubBranchesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GitHubBranch
 */
export type GitHubBranch = Message<"obiente.cloud.deployments.v1.GitHubBranch"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: bool is_default = 2;
     */
    isDefault: boolean;
    /**
     * @generated from field: string sha = 3;
     */
    sha: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GitHubBranch.
 * Use `create(GitHubBranchSchema)` to create a new message.
 */
export declare const GitHubBranchSchema: GenMessage<GitHubBranch>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetGitHubBranchesResponse
 */
export type GetGitHubBranchesResponse = Message<"obiente.cloud.deployments.v1.GetGitHubBranchesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.GitHubBranch branches = 1;
     */
    branches: GitHubBranch[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetGitHubBranchesResponse.
 * Use `create(GetGitHubBranchesResponseSchema)` to create a new message.
 */
export declare const GetGitHubBranchesResponseSchema: GenMessage<GetGitHubBranchesResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetGitHubFileRequest
 */
export type GetGitHubFileRequest = Message<"obiente.cloud.deployments.v1.GetGitHubFileRequest"> & {
    /**
     * Organization ID to use for GitHub token (optional, will use user token if not provided)
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * Optional: use specific GitHub integration ID
     *
     * @generated from field: string integration_id = 2;
     */
    integrationId: string;
    /**
     * e.g., "owner/repo"
     *
     * @generated from field: string repo_full_name = 3;
     */
    repoFullName: string;
    /**
     * @generated from field: string branch = 4;
     */
    branch: string;
    /**
     * e.g., "docker-compose.yml"
     *
     * @generated from field: string path = 5;
     */
    path: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetGitHubFileRequest.
 * Use `create(GetGitHubFileRequestSchema)` to create a new message.
 */
export declare const GetGitHubFileRequestSchema: GenMessage<GetGitHubFileRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetGitHubFileResponse
 */
export type GetGitHubFileResponse = Message<"obiente.cloud.deployments.v1.GetGitHubFileResponse"> & {
    /**
     * @generated from field: string content = 1;
     */
    content: string;
    /**
     * "base64" or "text"
     *
     * @generated from field: string encoding = 2;
     */
    encoding: string;
    /**
     * @generated from field: int64 size = 3;
     */
    size: bigint;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetGitHubFileResponse.
 * Use `create(GetGitHubFileResponseSchema)` to create a new message.
 */
export declare const GetGitHubFileResponseSchema: GenMessage<GetGitHubFileResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsRequest
 */
export type ListAvailableGitHubIntegrationsRequest = Message<"obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsRequest"> & {
    /**
     * Optional: filter integrations for this organization
     *
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsRequest.
 * Use `create(ListAvailableGitHubIntegrationsRequestSchema)` to create a new message.
 */
export declare const ListAvailableGitHubIntegrationsRequestSchema: GenMessage<ListAvailableGitHubIntegrationsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GitHubIntegrationOption
 */
export type GitHubIntegrationOption = Message<"obiente.cloud.deployments.v1.GitHubIntegrationOption"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string username = 2;
     */
    username: string;
    /**
     * true if user integration, false if organization
     *
     * @generated from field: bool is_user = 3;
     */
    isUser: boolean;
    /**
     * Obiente cloud organization ID (only set if is_user is false)
     *
     * @generated from field: string obiente_org_id = 4;
     */
    obienteOrgId: string;
    /**
     * Obiente cloud organization name (only set if is_user is false)
     *
     * @generated from field: string obiente_org_name = 5;
     */
    obienteOrgName: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GitHubIntegrationOption.
 * Use `create(GitHubIntegrationOptionSchema)` to create a new message.
 */
export declare const GitHubIntegrationOptionSchema: GenMessage<GitHubIntegrationOption>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsResponse
 */
export type ListAvailableGitHubIntegrationsResponse = Message<"obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.GitHubIntegrationOption integrations = 1;
     */
    integrations: GitHubIntegrationOption[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListAvailableGitHubIntegrationsResponse.
 * Use `create(ListAvailableGitHubIntegrationsResponseSchema)` to create a new message.
 */
export declare const ListAvailableGitHubIntegrationsResponseSchema: GenMessage<ListAvailableGitHubIntegrationsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamTerminalOutputRequest
 */
export type StreamTerminalOutputRequest = Message<"obiente.cloud.deployments.v1.StreamTerminalOutputRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Terminal width
     *
     * @generated from field: int32 cols = 3;
     */
    cols: number;
    /**
     * Terminal height
     *
     * @generated from field: int32 rows = 4;
     */
    rows: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamTerminalOutputRequest.
 * Use `create(StreamTerminalOutputRequestSchema)` to create a new message.
 */
export declare const StreamTerminalOutputRequestSchema: GenMessage<StreamTerminalOutputRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.SendTerminalInputRequest
 */
export type SendTerminalInputRequest = Message<"obiente.cloud.deployments.v1.SendTerminalInputRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * User keyboard input
     *
     * @generated from field: bytes input = 3;
     */
    input: Uint8Array;
    /**
     * Terminal width (for resize)
     *
     * @generated from field: int32 cols = 4;
     */
    cols: number;
    /**
     * Terminal height (for resize)
     *
     * @generated from field: int32 rows = 5;
     */
    rows: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.SendTerminalInputRequest.
 * Use `create(SendTerminalInputRequestSchema)` to create a new message.
 */
export declare const SendTerminalInputRequestSchema: GenMessage<SendTerminalInputRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.SendTerminalInputResponse
 */
export type SendTerminalInputResponse = Message<"obiente.cloud.deployments.v1.SendTerminalInputResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.SendTerminalInputResponse.
 * Use `create(SendTerminalInputResponseSchema)` to create a new message.
 */
export declare const SendTerminalInputResponseSchema: GenMessage<SendTerminalInputResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.TerminalInput
 */
export type TerminalInput = Message<"obiente.cloud.deployments.v1.TerminalInput"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Optional: specific container ID to connect to
     *
     * @generated from field: string container_id = 6;
     */
    containerId: string;
    /**
     * Optional: service name to connect to (for Docker Compose)
     *
     * @generated from field: string service_name = 7;
     */
    serviceName: string;
    /**
     * User keyboard input
     *
     * @generated from field: bytes input = 3;
     */
    input: Uint8Array;
    /**
     * Terminal width (for resize/initial setup)
     *
     * @generated from field: int32 cols = 4;
     */
    cols: number;
    /**
     * Terminal height (for resize/initial setup)
     *
     * @generated from field: int32 rows = 5;
     */
    rows: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.TerminalInput.
 * Use `create(TerminalInputSchema)` to create a new message.
 */
export declare const TerminalInputSchema: GenMessage<TerminalInput>;
/**
 * @generated from message obiente.cloud.deployments.v1.TerminalOutput
 */
export type TerminalOutput = Message<"obiente.cloud.deployments.v1.TerminalOutput"> & {
    /**
     * Terminal stdout/stderr
     *
     * @generated from field: bytes output = 1;
     */
    output: Uint8Array;
    /**
     * Whether the terminal session has ended
     *
     * @generated from field: bool exit = 2;
     */
    exit: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.TerminalOutput.
 * Use `create(TerminalOutputSchema)` to create a new message.
 */
export declare const TerminalOutputSchema: GenMessage<TerminalOutput>;
/**
 * @generated from message obiente.cloud.deployments.v1.VolumeInfo
 */
export type VolumeInfo = Message<"obiente.cloud.deployments.v1.VolumeInfo"> & {
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
 * Describes the message obiente.cloud.deployments.v1.VolumeInfo.
 * Use `create(VolumeInfoSchema)` to create a new message.
 */
export declare const VolumeInfoSchema: GenMessage<VolumeInfo>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListContainerFilesRequest
 */
export type ListContainerFilesRequest = Message<"obiente.cloud.deployments.v1.ListContainerFilesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Directory path (default: "/")
     *
     * @generated from field: string path = 3;
     */
    path: string;
    /**
     * If specified, list files from this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * If true, return list of available volumes instead of files
     *
     * @generated from field: bool list_volumes = 5;
     */
    listVolumes: boolean;
    /**
     * @generated from field: optional string cursor = 6;
     */
    cursor?: string;
    /**
     * @generated from field: optional int32 page_size = 7;
     */
    pageSize?: number;
    /**
     * Optional: use a specific container. If not specified, uses first container.
     *
     * @generated from field: optional string container_id = 8;
     */
    containerId?: string;
    /**
     * Optional: use a specific service. If container_id is specified, this is ignored.
     *
     * @generated from field: optional string service_name = 9;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListContainerFilesRequest.
 * Use `create(ListContainerFilesRequestSchema)` to create a new message.
 */
export declare const ListContainerFilesRequestSchema: GenMessage<ListContainerFilesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ContainerFile
 */
export type ContainerFile = Message<"obiente.cloud.deployments.v1.ContainerFile"> & {
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
 * Describes the message obiente.cloud.deployments.v1.ContainerFile.
 * Use `create(ContainerFileSchema)` to create a new message.
 */
export declare const ContainerFileSchema: GenMessage<ContainerFile>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListContainerFilesResponse
 */
export type ListContainerFilesResponse = Message<"obiente.cloud.deployments.v1.ListContainerFilesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.ContainerFile files = 1;
     */
    files: ContainerFile[];
    /**
     * @generated from field: string current_path = 2;
     */
    currentPath: string;
    /**
     * Available persistent volumes (only when list_volumes=true)
     *
     * @generated from field: repeated obiente.cloud.deployments.v1.VolumeInfo volumes = 3;
     */
    volumes: VolumeInfo[];
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
 * Describes the message obiente.cloud.deployments.v1.ListContainerFilesResponse.
 * Use `create(ListContainerFilesResponseSchema)` to create a new message.
 */
export declare const ListContainerFilesResponseSchema: GenMessage<ListContainerFilesResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetContainerFileRequest
 */
export type GetContainerFileRequest = Message<"obiente.cloud.deployments.v1.GetContainerFileRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * File path
     *
     * @generated from field: string path = 3;
     */
    path: string;
    /**
     * If specified, read file from this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * Optional: use a specific container. If not specified, uses first container.
     *
     * @generated from field: optional string container_id = 5;
     */
    containerId?: string;
    /**
     * Optional: use a specific service. If container_id is specified, this is ignored.
     *
     * @generated from field: optional string service_name = 6;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetContainerFileRequest.
 * Use `create(GetContainerFileRequestSchema)` to create a new message.
 */
export declare const GetContainerFileRequestSchema: GenMessage<GetContainerFileRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetContainerFileResponse
 */
export type GetContainerFileResponse = Message<"obiente.cloud.deployments.v1.GetContainerFileResponse"> & {
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
     * @generated from field: optional obiente.cloud.deployments.v1.ContainerFile metadata = 5;
     */
    metadata?: ContainerFile;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetContainerFileResponse.
 * Use `create(GetContainerFileResponseSchema)` to create a new message.
 */
export declare const GetContainerFileResponseSchema: GenMessage<GetContainerFileResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.UploadContainerFilesRequest
 */
export type UploadContainerFilesRequest = Message<"obiente.cloud.deployments.v1.UploadContainerFilesRequest"> & {
    /**
     * Metadata about the upload
     *
     * @generated from field: obiente.cloud.deployments.v1.UploadContainerFilesMetadata metadata = 1;
     */
    metadata?: UploadContainerFilesMetadata;
    /**
     * Complete tar archive containing all files
     *
     * @generated from field: bytes tar_data = 2;
     */
    tarData: Uint8Array;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UploadContainerFilesRequest.
 * Use `create(UploadContainerFilesRequestSchema)` to create a new message.
 */
export declare const UploadContainerFilesRequestSchema: GenMessage<UploadContainerFilesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.UploadContainerFilesMetadata
 */
export type UploadContainerFilesMetadata = Message<"obiente.cloud.deployments.v1.UploadContainerFilesMetadata"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Directory path where files should be extracted (default: "/")
     *
     * @generated from field: string destination_path = 3;
     */
    destinationPath: string;
    /**
     * Metadata about files being uploaded
     *
     * @generated from field: repeated obiente.cloud.deployments.v1.FileMetadata files = 4;
     */
    files: FileMetadata[];
    /**
     * If specified, upload files to this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 5;
     */
    volumeName?: string;
    /**
     * Optional: use a specific container. If not specified, uses first container.
     *
     * @generated from field: optional string container_id = 6;
     */
    containerId?: string;
    /**
     * Optional: use a specific service. If container_id is specified, this is ignored.
     *
     * @generated from field: optional string service_name = 7;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.UploadContainerFilesMetadata.
 * Use `create(UploadContainerFilesMetadataSchema)` to create a new message.
 */
export declare const UploadContainerFilesMetadataSchema: GenMessage<UploadContainerFilesMetadata>;
/**
 * @generated from message obiente.cloud.deployments.v1.FileMetadata
 */
export type FileMetadata = Message<"obiente.cloud.deployments.v1.FileMetadata"> & {
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
 * Describes the message obiente.cloud.deployments.v1.FileMetadata.
 * Use `create(FileMetadataSchema)` to create a new message.
 */
export declare const FileMetadataSchema: GenMessage<FileMetadata>;
/**
 * @generated from message obiente.cloud.deployments.v1.UploadContainerFilesResponse
 */
export type UploadContainerFilesResponse = Message<"obiente.cloud.deployments.v1.UploadContainerFilesResponse"> & {
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
 * Describes the message obiente.cloud.deployments.v1.UploadContainerFilesResponse.
 * Use `create(UploadContainerFilesResponseSchema)` to create a new message.
 */
export declare const UploadContainerFilesResponseSchema: GenMessage<UploadContainerFilesResponse>;
/**
 * Chunked upload for deployments using shared payload for consistency with other services.
 *
 * @generated from message obiente.cloud.deployments.v1.ChunkUploadContainerFilesRequest
 */
export type ChunkUploadContainerFilesRequest = Message<"obiente.cloud.deployments.v1.ChunkUploadContainerFilesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: obiente.cloud.common.v1.ChunkedUploadPayload upload = 3;
     */
    upload?: ChunkedUploadPayload;
    /**
     * Optional: target a specific container
     *
     * @generated from field: optional string container_id = 4;
     */
    containerId?: string;
    /**
     * Optional: target a specific service (ignored if container_id provided)
     *
     * @generated from field: optional string service_name = 5;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ChunkUploadContainerFilesRequest.
 * Use `create(ChunkUploadContainerFilesRequestSchema)` to create a new message.
 */
export declare const ChunkUploadContainerFilesRequestSchema: GenMessage<ChunkUploadContainerFilesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ChunkUploadContainerFilesResponse
 */
export type ChunkUploadContainerFilesResponse = Message<"obiente.cloud.deployments.v1.ChunkUploadContainerFilesResponse"> & {
    /**
     * @generated from field: obiente.cloud.common.v1.ChunkedUploadResponsePayload result = 1;
     */
    result?: ChunkedUploadResponsePayload;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ChunkUploadContainerFilesResponse.
 * Use `create(ChunkUploadContainerFilesResponseSchema)` to create a new message.
 */
export declare const ChunkUploadContainerFilesResponseSchema: GenMessage<ChunkUploadContainerFilesResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteContainerEntriesRequest
 */
export type DeleteContainerEntriesRequest = Message<"obiente.cloud.deployments.v1.DeleteContainerEntriesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: repeated string paths = 3;
     */
    paths: string[];
    /**
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * @generated from field: bool recursive = 5;
     */
    recursive: boolean;
    /**
     * @generated from field: bool force = 6;
     */
    force: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteContainerEntriesRequest.
 * Use `create(DeleteContainerEntriesRequestSchema)` to create a new message.
 */
export declare const DeleteContainerEntriesRequestSchema: GenMessage<DeleteContainerEntriesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteContainerEntriesError
 */
export type DeleteContainerEntriesError = Message<"obiente.cloud.deployments.v1.DeleteContainerEntriesError"> & {
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
 * Describes the message obiente.cloud.deployments.v1.DeleteContainerEntriesError.
 * Use `create(DeleteContainerEntriesErrorSchema)` to create a new message.
 */
export declare const DeleteContainerEntriesErrorSchema: GenMessage<DeleteContainerEntriesError>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteContainerEntriesResponse
 */
export type DeleteContainerEntriesResponse = Message<"obiente.cloud.deployments.v1.DeleteContainerEntriesResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: repeated string deleted_paths = 2;
     */
    deletedPaths: string[];
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.DeleteContainerEntriesError errors = 3;
     */
    errors: DeleteContainerEntriesError[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteContainerEntriesResponse.
 * Use `create(DeleteContainerEntriesResponseSchema)` to create a new message.
 */
export declare const DeleteContainerEntriesResponseSchema: GenMessage<DeleteContainerEntriesResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.RenameContainerEntryRequest
 */
export type RenameContainerEntryRequest = Message<"obiente.cloud.deployments.v1.RenameContainerEntryRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string source_path = 3;
     */
    sourcePath: string;
    /**
     * @generated from field: string target_path = 4;
     */
    targetPath: string;
    /**
     * @generated from field: optional string volume_name = 5;
     */
    volumeName?: string;
    /**
     * @generated from field: bool overwrite = 6;
     */
    overwrite: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RenameContainerEntryRequest.
 * Use `create(RenameContainerEntryRequestSchema)` to create a new message.
 */
export declare const RenameContainerEntryRequestSchema: GenMessage<RenameContainerEntryRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.RenameContainerEntryResponse
 */
export type RenameContainerEntryResponse = Message<"obiente.cloud.deployments.v1.RenameContainerEntryResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional obiente.cloud.deployments.v1.ContainerFile entry = 2;
     */
    entry?: ContainerFile;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RenameContainerEntryResponse.
 * Use `create(RenameContainerEntryResponseSchema)` to create a new message.
 */
export declare const RenameContainerEntryResponseSchema: GenMessage<RenameContainerEntryResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateContainerEntryRequest
 */
export type CreateContainerEntryRequest = Message<"obiente.cloud.deployments.v1.CreateContainerEntryRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string parent_path = 3;
     */
    parentPath: string;
    /**
     * @generated from field: string name = 4;
     */
    name: string;
    /**
     * @generated from field: obiente.cloud.deployments.v1.ContainerEntryType type = 5;
     */
    type: ContainerEntryType;
    /**
     * Optional template identifier for seeded content (required for symlinks - target path)
     *
     * @generated from field: optional string template = 6;
     */
    template?: string;
    /**
     * @generated from field: optional string volume_name = 7;
     */
    volumeName?: string;
    /**
     * @generated from field: optional uint32 mode_octal = 8;
     */
    modeOctal?: number;
    /**
     * Specific container to create entry in (for compose deployments)
     *
     * @generated from field: optional string container_id = 9;
     */
    containerId?: string;
    /**
     * Service name to create entry in (for compose deployments)
     *
     * @generated from field: optional string service_name = 10;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateContainerEntryRequest.
 * Use `create(CreateContainerEntryRequestSchema)` to create a new message.
 */
export declare const CreateContainerEntryRequestSchema: GenMessage<CreateContainerEntryRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateContainerEntryResponse
 */
export type CreateContainerEntryResponse = Message<"obiente.cloud.deployments.v1.CreateContainerEntryResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.ContainerFile entry = 1;
     */
    entry?: ContainerFile;
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateContainerEntryResponse.
 * Use `create(CreateContainerEntryResponseSchema)` to create a new message.
 */
export declare const CreateContainerEntryResponseSchema: GenMessage<CreateContainerEntryResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.WriteContainerFileRequest
 */
export type WriteContainerFileRequest = Message<"obiente.cloud.deployments.v1.WriteContainerFileRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string path = 3;
     */
    path: string;
    /**
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * @generated from field: string content = 5;
     */
    content: string;
    /**
     * "text" or "base64"
     *
     * @generated from field: string encoding = 6;
     */
    encoding: string;
    /**
     * @generated from field: bool create_if_missing = 7;
     */
    createIfMissing: boolean;
    /**
     * @generated from field: optional uint32 mode_octal = 8;
     */
    modeOctal?: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.WriteContainerFileRequest.
 * Use `create(WriteContainerFileRequestSchema)` to create a new message.
 */
export declare const WriteContainerFileRequestSchema: GenMessage<WriteContainerFileRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.WriteContainerFileResponse
 */
export type WriteContainerFileResponse = Message<"obiente.cloud.deployments.v1.WriteContainerFileResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional obiente.cloud.deployments.v1.ContainerFile entry = 2;
     */
    entry?: ContainerFile;
    /**
     * @generated from field: optional string error = 3;
     */
    error?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.WriteContainerFileResponse.
 * Use `create(WriteContainerFileResponseSchema)` to create a new message.
 */
export declare const WriteContainerFileResponseSchema: GenMessage<WriteContainerFileResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.ExtractDeploymentFileRequest
 */
export type ExtractDeploymentFileRequest = Message<"obiente.cloud.deployments.v1.ExtractDeploymentFileRequest"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Path to the zip file to extract (from common.ExtractServerFileRequest)
     *
     * @generated from field: string zip_path = 3;
     */
    zipPath: string;
    /**
     * Directory path where files should be extracted (from common.ExtractServerFileRequest)
     *
     * @generated from field: string destination_path = 4;
     */
    destinationPath: string;
    /**
     * If specified, extract to this volume instead of container filesystem (from common.ExtractServerFileRequest)
     *
     * @generated from field: optional string volume_name = 5;
     */
    volumeName?: string;
    /**
     * If specified, extract to this container
     *
     * @generated from field: optional string container_id = 6;
     */
    containerId?: string;
    /**
     * If specified, extract to container with this service name
     *
     * @generated from field: optional string service_name = 7;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ExtractDeploymentFileRequest.
 * Use `create(ExtractDeploymentFileRequestSchema)` to create a new message.
 */
export declare const ExtractDeploymentFileRequestSchema: GenMessage<ExtractDeploymentFileRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ExtractDeploymentFileResponse
 */
export type ExtractDeploymentFileResponse = Message<"obiente.cloud.deployments.v1.ExtractDeploymentFileResponse"> & {
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
 * Describes the message obiente.cloud.deployments.v1.ExtractDeploymentFileResponse.
 * Use `create(ExtractDeploymentFileResponseSchema)` to create a new message.
 */
export declare const ExtractDeploymentFileResponseSchema: GenMessage<ExtractDeploymentFileResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateDeploymentFileArchiveRequest
 */
export type CreateDeploymentFileArchiveRequest = Message<"obiente.cloud.deployments.v1.CreateDeploymentFileArchiveRequest"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Shared archive request
     *
     * @generated from field: obiente.cloud.common.v1.CreateServerFileArchiveRequest archive_request = 3;
     */
    archiveRequest?: CreateServerFileArchiveRequest;
    /**
     * If specified, create archive in this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 4;
     */
    volumeName?: string;
    /**
     * If specified, create archive in this container
     *
     * @generated from field: optional string container_id = 5;
     */
    containerId?: string;
    /**
     * If specified, create archive in container with this service name
     *
     * @generated from field: optional string service_name = 6;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateDeploymentFileArchiveRequest.
 * Use `create(CreateDeploymentFileArchiveRequestSchema)` to create a new message.
 */
export declare const CreateDeploymentFileArchiveRequestSchema: GenMessage<CreateDeploymentFileArchiveRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.CreateDeploymentFileArchiveResponse
 */
export type CreateDeploymentFileArchiveResponse = Message<"obiente.cloud.deployments.v1.CreateDeploymentFileArchiveResponse"> & {
    /**
     * Shared archive response
     *
     * @generated from field: obiente.cloud.common.v1.CreateServerFileArchiveResponse archive_response = 1;
     */
    archiveResponse?: CreateServerFileArchiveResponse;
};
/**
 * Describes the message obiente.cloud.deployments.v1.CreateDeploymentFileArchiveResponse.
 * Use `create(CreateDeploymentFileArchiveResponseSchema)` to create a new message.
 */
export declare const CreateDeploymentFileArchiveResponseSchema: GenMessage<CreateDeploymentFileArchiveResponse>;
/**
 * Routing configuration messages
 *
 * @generated from message obiente.cloud.deployments.v1.RoutingRule
 */
export type RoutingRule = Message<"obiente.cloud.deployments.v1.RoutingRule"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
    /**
     * Service name (e.g., "api", "web", "admin", "default")
     *
     * @generated from field: string service_name = 4;
     */
    serviceName: string;
    /**
     * Optional path prefix (e.g., "/api")
     *
     * @generated from field: string path_prefix = 5;
     */
    pathPrefix: string;
    /**
     * @generated from field: int32 target_port = 6;
     */
    targetPort: number;
    /**
     * "http", "https", "grpc"
     *
     * @generated from field: string protocol = 7;
     */
    protocol: string;
    /**
     * @generated from field: bool ssl_enabled = 8;
     */
    sslEnabled: boolean;
    /**
     * @generated from field: string ssl_cert_resolver = 9;
     */
    sslCertResolver: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RoutingRule.
 * Use `create(RoutingRuleSchema)` to create a new message.
 */
export declare const RoutingRuleSchema: GenMessage<RoutingRule>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentRoutingsRequest
 */
export type GetDeploymentRoutingsRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentRoutingsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentRoutingsRequest.
 * Use `create(GetDeploymentRoutingsRequestSchema)` to create a new message.
 */
export declare const GetDeploymentRoutingsRequestSchema: GenMessage<GetDeploymentRoutingsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentRoutingsResponse
 */
export type GetDeploymentRoutingsResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentRoutingsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.RoutingRule rules = 1;
     */
    rules: RoutingRule[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentRoutingsResponse.
 * Use `create(GetDeploymentRoutingsResponseSchema)` to create a new message.
 */
export declare const GetDeploymentRoutingsResponseSchema: GenMessage<GetDeploymentRoutingsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentRoutingsRequest
 */
export type UpdateDeploymentRoutingsRequest = Message<"obiente.cloud.deployments.v1.UpdateDeploymentRoutingsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Replace all existing rules with these
     *
     * @generated from field: repeated obiente.cloud.deployments.v1.RoutingRule rules = 3;
     */
    rules: RoutingRule[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentRoutingsRequest.
 * Use `create(UpdateDeploymentRoutingsRequestSchema)` to create a new message.
 */
export declare const UpdateDeploymentRoutingsRequestSchema: GenMessage<UpdateDeploymentRoutingsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.UpdateDeploymentRoutingsResponse
 */
export type UpdateDeploymentRoutingsResponse = Message<"obiente.cloud.deployments.v1.UpdateDeploymentRoutingsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.RoutingRule rules = 1;
     */
    rules: RoutingRule[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.UpdateDeploymentRoutingsResponse.
 * Use `create(UpdateDeploymentRoutingsResponseSchema)` to create a new message.
 */
export declare const UpdateDeploymentRoutingsResponseSchema: GenMessage<UpdateDeploymentRoutingsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentServiceNamesRequest
 */
export type GetDeploymentServiceNamesRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentServiceNamesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentServiceNamesRequest.
 * Use `create(GetDeploymentServiceNamesRequestSchema)` to create a new message.
 */
export declare const GetDeploymentServiceNamesRequestSchema: GenMessage<GetDeploymentServiceNamesRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentServiceNamesResponse
 */
export type GetDeploymentServiceNamesResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentServiceNamesResponse"> & {
    /**
     * @generated from field: repeated string service_names = 1;
     */
    serviceNames: string[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentServiceNamesResponse.
 * Use `create(GetDeploymentServiceNamesResponseSchema)` to create a new message.
 */
export declare const GetDeploymentServiceNamesResponseSchema: GenMessage<GetDeploymentServiceNamesResponse>;
/**
 * Domain verification messages
 *
 * @generated from message obiente.cloud.deployments.v1.GetDomainVerificationTokenRequest
 */
export type GetDomainVerificationTokenRequest = Message<"obiente.cloud.deployments.v1.GetDomainVerificationTokenRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDomainVerificationTokenRequest.
 * Use `create(GetDomainVerificationTokenRequestSchema)` to create a new message.
 */
export declare const GetDomainVerificationTokenRequestSchema: GenMessage<GetDomainVerificationTokenRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDomainVerificationTokenResponse
 */
export type GetDomainVerificationTokenResponse = Message<"obiente.cloud.deployments.v1.GetDomainVerificationTokenResponse"> & {
    /**
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * @generated from field: string token = 2;
     */
    token: string;
    /**
     * e.g., "_obiente-verification.example.com"
     *
     * @generated from field: string txt_record_name = 3;
     */
    txtRecordName: string;
    /**
     * e.g., "obiente-verification=abc123..."
     *
     * @generated from field: string txt_record_value = 4;
     */
    txtRecordValue: string;
    /**
     * "pending", "verified", "failed", "expired"
     *
     * @generated from field: string status = 5;
     */
    status: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDomainVerificationTokenResponse.
 * Use `create(GetDomainVerificationTokenResponseSchema)` to create a new message.
 */
export declare const GetDomainVerificationTokenResponseSchema: GenMessage<GetDomainVerificationTokenResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.VerifyDomainOwnershipRequest
 */
export type VerifyDomainOwnershipRequest = Message<"obiente.cloud.deployments.v1.VerifyDomainOwnershipRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.VerifyDomainOwnershipRequest.
 * Use `create(VerifyDomainOwnershipRequestSchema)` to create a new message.
 */
export declare const VerifyDomainOwnershipRequestSchema: GenMessage<VerifyDomainOwnershipRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.VerifyDomainOwnershipResponse
 */
export type VerifyDomainOwnershipResponse = Message<"obiente.cloud.deployments.v1.VerifyDomainOwnershipResponse"> & {
    /**
     * @generated from field: string domain = 1;
     */
    domain: string;
    /**
     * @generated from field: bool verified = 2;
     */
    verified: boolean;
    /**
     * "verified", "failed", "pending"
     *
     * @generated from field: string status = 3;
     */
    status: string;
    /**
     * Error message if verification failed
     *
     * @generated from field: optional string message = 4;
     */
    message?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.VerifyDomainOwnershipResponse.
 * Use `create(VerifyDomainOwnershipResponseSchema)` to create a new message.
 */
export declare const VerifyDomainOwnershipResponseSchema: GenMessage<VerifyDomainOwnershipResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentMetricsRequest
 */
export type GetDeploymentMetricsRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentMetricsRequest"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * Optional: time range for historical metrics
     *
     * @generated from field: optional google.protobuf.Timestamp start_time = 3;
     */
    startTime?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp end_time = 4;
     */
    endTime?: Timestamp;
    /**
     * If true, return only the latest metrics point
     *
     * @generated from field: optional bool latest_only = 5;
     */
    latestOnly?: boolean;
    /**
     * Optional: filter by container_id (if not specified, returns aggregated metrics across all containers)
     *
     * @generated from field: optional string container_id = 6;
     */
    containerId?: string;
    /**
     * Optional: filter by service_name (alternative to container_id, matches containers by service name)
     *
     * @generated from field: optional string service_name = 7;
     */
    serviceName?: string;
    /**
     * If true, aggregate metrics across all containers (sum for network/disk, avg for CPU/memory)
     * If false and container_id/service_name not specified, aggregates are returned
     *
     * @generated from field: optional bool aggregate = 8;
     */
    aggregate?: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentMetricsRequest.
 * Use `create(GetDeploymentMetricsRequestSchema)` to create a new message.
 */
export declare const GetDeploymentMetricsRequestSchema: GenMessage<GetDeploymentMetricsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentMetricsResponse
 */
export type GetDeploymentMetricsResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentMetricsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.DeploymentMetric metrics = 1;
     */
    metrics: DeploymentMetric[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentMetricsResponse.
 * Use `create(GetDeploymentMetricsResponseSchema)` to create a new message.
 */
export declare const GetDeploymentMetricsResponseSchema: GenMessage<GetDeploymentMetricsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamDeploymentMetricsRequest
 */
export type StreamDeploymentMetricsRequest = Message<"obiente.cloud.deployments.v1.StreamDeploymentMetricsRequest"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * How often to send updates (in seconds, default: 5)
     *
     * @generated from field: optional int32 interval_seconds = 3;
     */
    intervalSeconds?: number;
    /**
     * Optional: filter by container_id (if not specified, returns aggregated metrics across all containers)
     *
     * @generated from field: optional string container_id = 4;
     */
    containerId?: string;
    /**
     * Optional: filter by service_name (alternative to container_id, matches containers by service name)
     *
     * @generated from field: optional string service_name = 5;
     */
    serviceName?: string;
    /**
     * If true, aggregate metrics across all containers (sum for network/disk, avg for CPU/memory)
     * If false and container_id/service_name not specified, aggregates are returned
     *
     * @generated from field: optional bool aggregate = 6;
     */
    aggregate?: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamDeploymentMetricsRequest.
 * Use `create(StreamDeploymentMetricsRequestSchema)` to create a new message.
 */
export declare const StreamDeploymentMetricsRequestSchema: GenMessage<StreamDeploymentMetricsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeploymentMetric
 */
export type DeploymentMetric = Message<"obiente.cloud.deployments.v1.DeploymentMetric"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 2;
     */
    timestamp?: Timestamp;
    /**
     * CPU usage as percentage (0-100)
     *
     * @generated from field: double cpu_usage_percent = 3;
     */
    cpuUsagePercent: number;
    /**
     * Memory usage in bytes
     *
     * @generated from field: int64 memory_usage_bytes = 4;
     */
    memoryUsageBytes: bigint;
    /**
     * Network receive bytes (total since container start)
     *
     * @generated from field: int64 network_rx_bytes = 5;
     */
    networkRxBytes: bigint;
    /**
     * Network transmit bytes (total since container start)
     *
     * @generated from field: int64 network_tx_bytes = 6;
     */
    networkTxBytes: bigint;
    /**
     * Disk read bytes (total since container start)
     *
     * @generated from field: int64 disk_read_bytes = 7;
     */
    diskReadBytes: bigint;
    /**
     * Disk write bytes (total since container start)
     *
     * @generated from field: int64 disk_write_bytes = 8;
     */
    diskWriteBytes: bigint;
    /**
     * Request count (if tracked via middleware)
     *
     * @generated from field: optional int64 request_count = 9;
     */
    requestCount?: bigint;
    /**
     * Error count (if tracked via middleware)
     *
     * @generated from field: optional int64 error_count = 10;
     */
    errorCount?: bigint;
    /**
     * Container ID this metric is from (for multi-container deployments)
     *
     * @generated from field: optional string container_id = 11;
     */
    containerId?: string;
    /**
     * Service name this metric is from (for multi-container deployments)
     *
     * @generated from field: optional string service_name = 12;
     */
    serviceName?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeploymentMetric.
 * Use `create(DeploymentMetricSchema)` to create a new message.
 */
export declare const DeploymentMetricSchema: GenMessage<DeploymentMetric>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentUsageRequest
 */
export type GetDeploymentUsageRequest = Message<"obiente.cloud.deployments.v1.GetDeploymentUsageRequest"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
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
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentUsageRequest.
 * Use `create(GetDeploymentUsageRequestSchema)` to create a new message.
 */
export declare const GetDeploymentUsageRequestSchema: GenMessage<GetDeploymentUsageRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetDeploymentUsageResponse
 */
export type GetDeploymentUsageResponse = Message<"obiente.cloud.deployments.v1.GetDeploymentUsageResponse"> & {
    /**
     * @generated from field: string deployment_id = 1;
     */
    deploymentId: string;
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
     * @generated from field: obiente.cloud.deployments.v1.DeploymentUsageMetrics current = 4;
     */
    current?: DeploymentUsageMetrics;
    /**
     * @generated from field: obiente.cloud.deployments.v1.DeploymentUsageMetrics estimated_monthly = 5;
     */
    estimatedMonthly?: DeploymentUsageMetrics;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetDeploymentUsageResponse.
 * Use `create(GetDeploymentUsageResponseSchema)` to create a new message.
 */
export declare const GetDeploymentUsageResponseSchema: GenMessage<GetDeploymentUsageResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeploymentUsageMetrics
 */
export type DeploymentUsageMetrics = Message<"obiente.cloud.deployments.v1.DeploymentUsageMetrics"> & {
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
     * @generated from field: int64 request_count = 6;
     */
    requestCount: bigint;
    /**
     * @generated from field: int64 error_count = 7;
     */
    errorCount: bigint;
    /**
     * @generated from field: int64 uptime_seconds = 8;
     */
    uptimeSeconds: bigint;
    /**
     * Estimated cost in cents
     *
     * @generated from field: int64 estimated_cost_cents = 9;
     */
    estimatedCostCents: bigint;
    /**
     * Per-resource cost breakdown in cents
     *
     * @generated from field: optional int64 cpu_cost_cents = 10;
     */
    cpuCostCents?: bigint;
    /**
     * @generated from field: optional int64 memory_cost_cents = 11;
     */
    memoryCostCents?: bigint;
    /**
     * @generated from field: optional int64 bandwidth_cost_cents = 12;
     */
    bandwidthCostCents?: bigint;
    /**
     * @generated from field: optional int64 storage_cost_cents = 13;
     */
    storageCostCents?: bigint;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeploymentUsageMetrics.
 * Use `create(DeploymentUsageMetricsSchema)` to create a new message.
 */
export declare const DeploymentUsageMetricsSchema: GenMessage<DeploymentUsageMetrics>;
/**
 * @generated from message obiente.cloud.deployments.v1.Deployment
 */
export type Deployment = Message<"obiente.cloud.deployments.v1.Deployment"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string domain = 3;
     */
    domain: string;
    /**
     * @generated from field: repeated string custom_domains = 4;
     */
    customDomains: string[];
    /**
     * @generated from field: obiente.cloud.deployments.v1.DeploymentType type = 5;
     */
    type: DeploymentType;
    /**
     * build strategy enum
     *
     * @generated from field: obiente.cloud.deployments.v1.BuildStrategy build_strategy = 26;
     */
    buildStrategy: BuildStrategy;
    /**
     * @generated from field: optional string repository_url = 6;
     */
    repositoryUrl?: string;
    /**
     * @generated from field: string branch = 7;
     */
    branch: string;
    /**
     * @generated from field: optional string build_command = 8;
     */
    buildCommand?: string;
    /**
     * @generated from field: optional string install_command = 9;
     */
    installCommand?: string;
    /**
     * Start command for running the application
     *
     * @generated from field: optional string start_command = 29;
     */
    startCommand?: string;
    /**
     * Path to Dockerfile (relative to repo root, e.g., "Dockerfile", "backend/Dockerfile")
     *
     * @generated from field: optional string dockerfile_path = 27;
     */
    dockerfilePath?: string;
    /**
     * Path to compose file (relative to repo root, e.g., "docker-compose.yml", "compose/production.yml")
     *
     * @generated from field: optional string compose_file_path = 28;
     */
    composeFilePath?: string;
    /**
     * Working directory for build (relative to repo root, defaults to ".")
     *
     * @generated from field: optional string build_path = 34;
     */
    buildPath?: string;
    /**
     * Path to built output files (relative to repo root, auto-detected if empty)
     *
     * @generated from field: optional string build_output_path = 35;
     */
    buildOutputPath?: string;
    /**
     * Use nginx for static deployments
     *
     * @generated from field: optional bool use_nginx = 36;
     */
    useNginx?: boolean;
    /**
     * Custom nginx configuration (optional, uses default if empty)
     *
     * @generated from field: optional string nginx_config = 37;
     */
    nginxConfig?: string;
    /**
     * @generated from field: obiente.cloud.deployments.v1.DeploymentStatus status = 10;
     */
    status: DeploymentStatus;
    /**
     * @generated from field: string health_status = 11;
     */
    healthStatus: string;
    /**
     * @generated from field: google.protobuf.Timestamp last_deployed_at = 12;
     */
    lastDeployedAt?: Timestamp;
    /**
     * @generated from field: int64 bandwidth_usage = 13;
     */
    bandwidthUsage: bigint;
    /**
     * @generated from field: int64 storage_usage = 14;
     */
    storageUsage: bigint;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 15;
     */
    createdAt?: Timestamp;
    /**
     * Build time in seconds
     *
     * @generated from field: int32 build_time = 16;
     */
    buildTime: number;
    /**
     * Human-readable bundle size (e.g. "3.4MB")
     *
     * @generated from field: string size = 17;
     */
    size: string;
    /**
     * Environment enum (production/staging/development)
     *
     * @generated from field: obiente.cloud.deployments.v1.Environment environment = 18;
     */
    environment: Environment;
    /**
     * Optional groups/labels for organizing deployments
     *
     * @generated from field: repeated string groups = 32;
     */
    groups: string[];
    /**
     * Docker/runtime specific view
     *
     * @generated from field: optional string image = 19;
     */
    image?: string;
    /**
     * @generated from field: optional int32 port = 20;
     */
    port?: number;
    /**
     * @generated from field: optional int32 replicas = 21;
     */
    replicas?: number;
    /**
     * @generated from field: repeated string container_ids = 22;
     */
    containerIds: string[];
    /**
     * @generated from field: optional string node_id = 23;
     */
    nodeId?: string;
    /**
     * @generated from field: optional string node_hostname = 24;
     */
    nodeHostname?: string;
    /**
     * Environment variables
     *
     * @generated from field: map<string, string> env_vars = 25;
     */
    envVars: {
        [key: string]: string;
    };
    /**
     * GitHub integration ID used for this deployment
     *
     * @generated from field: optional string github_integration_id = 33;
     */
    githubIntegrationId?: string;
    /**
     * Container status (for compose deployments)
     *
     * Number of containers actually running (from Docker)
     *
     * @generated from field: optional int32 containers_running = 30;
     */
    containersRunning?: number;
    /**
     * Total number of containers for this deployment
     *
     * @generated from field: optional int32 containers_total = 31;
     */
    containersTotal?: number;
    /**
     * Per-deployment resource limits (optional overrides).
     * If unset (or set to 0), backend uses sane defaults capped by org plan.
     *
     * CPU limit in cores
     *
     * @generated from field: optional double cpu_limit = 38;
     */
    cpuLimit?: number;
    /**
     * Memory limit in MB
     *
     * @generated from field: optional int64 memory_limit = 39;
     */
    memoryLimit?: bigint;
    /**
     * Health check configuration
     *
     * Type of health check
     *
     * @generated from field: optional obiente.cloud.deployments.v1.HealthCheckType healthcheck_type = 40;
     */
    healthcheckType?: HealthCheckType;
    /**
     * Port to check (if different from main port)
     *
     * @generated from field: optional int32 healthcheck_port = 41;
     */
    healthcheckPort?: number;
    /**
     * HTTP path (default: "/", used with HEALTHCHECK_HTTP)
     *
     * @generated from field: optional string healthcheck_path = 42;
     */
    healthcheckPath?: string;
    /**
     * Expected HTTP status code (default: 200, used with HEALTHCHECK_HTTP)
     *
     * @generated from field: optional int32 healthcheck_expected_status = 43;
     */
    healthcheckExpectedStatus?: number;
    /**
     * Custom command (used with HEALTHCHECK_CUSTOM)
     *
     * @generated from field: optional string healthcheck_custom_command = 44;
     */
    healthcheckCustomCommand?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.Deployment.
 * Use `create(DeploymentSchema)` to create a new message.
 */
export declare const DeploymentSchema: GenMessage<Deployment>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListDeploymentContainersRequest
 */
export type ListDeploymentContainersRequest = Message<"obiente.cloud.deployments.v1.ListDeploymentContainersRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListDeploymentContainersRequest.
 * Use `create(ListDeploymentContainersRequestSchema)` to create a new message.
 */
export declare const ListDeploymentContainersRequestSchema: GenMessage<ListDeploymentContainersRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListDeploymentContainersResponse
 */
export type ListDeploymentContainersResponse = Message<"obiente.cloud.deployments.v1.ListDeploymentContainersResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.DeploymentContainer containers = 1;
     */
    containers: DeploymentContainer[];
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListDeploymentContainersResponse.
 * Use `create(ListDeploymentContainersResponseSchema)` to create a new message.
 */
export declare const ListDeploymentContainersResponseSchema: GenMessage<ListDeploymentContainersResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeploymentContainer
 */
export type DeploymentContainer = Message<"obiente.cloud.deployments.v1.DeploymentContainer"> & {
    /**
     * @generated from field: string container_id = 1;
     */
    containerId: string;
    /**
     * Service name from compose (e.g., "web", "api")
     *
     * @generated from field: optional string service_name = 2;
     */
    serviceName?: string;
    /**
     * running, stopped, exited, etc.
     *
     * @generated from field: string status = 3;
     */
    status: string;
    /**
     * @generated from field: optional string node_id = 4;
     */
    nodeId?: string;
    /**
     * @generated from field: optional string node_hostname = 5;
     */
    nodeHostname?: string;
    /**
     * @generated from field: optional int32 port = 6;
     */
    port?: number;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 7;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 8;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeploymentContainer.
 * Use `create(DeploymentContainerSchema)` to create a new message.
 */
export declare const DeploymentContainerSchema: GenMessage<DeploymentContainer>;
/**
 * @generated from message obiente.cloud.deployments.v1.StreamContainerLogsRequest
 */
export type StreamContainerLogsRequest = Message<"obiente.cloud.deployments.v1.StreamContainerLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Container ID to stream logs from
     *
     * @generated from field: string container_id = 3;
     */
    containerId: string;
    /**
     * Number of lines to tail (default: 200)
     *
     * @generated from field: int32 tail = 4;
     */
    tail: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StreamContainerLogsRequest.
 * Use `create(StreamContainerLogsRequestSchema)` to create a new message.
 */
export declare const StreamContainerLogsRequestSchema: GenMessage<StreamContainerLogsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StartContainerRequest
 */
export type StartContainerRequest = Message<"obiente.cloud.deployments.v1.StartContainerRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Container ID to start
     *
     * @generated from field: string container_id = 3;
     */
    containerId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StartContainerRequest.
 * Use `create(StartContainerRequestSchema)` to create a new message.
 */
export declare const StartContainerRequestSchema: GenMessage<StartContainerRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StartContainerResponse
 */
export type StartContainerResponse = Message<"obiente.cloud.deployments.v1.StartContainerResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StartContainerResponse.
 * Use `create(StartContainerResponseSchema)` to create a new message.
 */
export declare const StartContainerResponseSchema: GenMessage<StartContainerResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.StopContainerRequest
 */
export type StopContainerRequest = Message<"obiente.cloud.deployments.v1.StopContainerRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Container ID to stop
     *
     * @generated from field: string container_id = 3;
     */
    containerId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StopContainerRequest.
 * Use `create(StopContainerRequestSchema)` to create a new message.
 */
export declare const StopContainerRequestSchema: GenMessage<StopContainerRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.StopContainerResponse
 */
export type StopContainerResponse = Message<"obiente.cloud.deployments.v1.StopContainerResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.StopContainerResponse.
 * Use `create(StopContainerResponseSchema)` to create a new message.
 */
export declare const StopContainerResponseSchema: GenMessage<StopContainerResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.RestartContainerRequest
 */
export type RestartContainerRequest = Message<"obiente.cloud.deployments.v1.RestartContainerRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Container ID to restart
     *
     * @generated from field: string container_id = 3;
     */
    containerId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RestartContainerRequest.
 * Use `create(RestartContainerRequestSchema)` to create a new message.
 */
export declare const RestartContainerRequestSchema: GenMessage<RestartContainerRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.RestartContainerResponse
 */
export type RestartContainerResponse = Message<"obiente.cloud.deployments.v1.RestartContainerResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RestartContainerResponse.
 * Use `create(RestartContainerResponseSchema)` to create a new message.
 */
export declare const RestartContainerResponseSchema: GenMessage<RestartContainerResponse>;
/**
 * Build history messages
 *
 * @generated from message obiente.cloud.deployments.v1.ListBuildsRequest
 */
export type ListBuildsRequest = Message<"obiente.cloud.deployments.v1.ListBuildsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * Maximum number of builds to return (default: 50)
     *
     * @generated from field: optional int32 limit = 3;
     */
    limit?: number;
    /**
     * Offset for pagination
     *
     * @generated from field: optional int32 offset = 4;
     */
    offset?: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListBuildsRequest.
 * Use `create(ListBuildsRequestSchema)` to create a new message.
 */
export declare const ListBuildsRequestSchema: GenMessage<ListBuildsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.ListBuildsResponse
 */
export type ListBuildsResponse = Message<"obiente.cloud.deployments.v1.ListBuildsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.Build builds = 1;
     */
    builds: Build[];
    /**
     * Total number of builds
     *
     * @generated from field: int32 total = 2;
     */
    total: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.ListBuildsResponse.
 * Use `create(ListBuildsResponseSchema)` to create a new message.
 */
export declare const ListBuildsResponseSchema: GenMessage<ListBuildsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetBuildRequest
 */
export type GetBuildRequest = Message<"obiente.cloud.deployments.v1.GetBuildRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string build_id = 3;
     */
    buildId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetBuildRequest.
 * Use `create(GetBuildRequestSchema)` to create a new message.
 */
export declare const GetBuildRequestSchema: GenMessage<GetBuildRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetBuildResponse
 */
export type GetBuildResponse = Message<"obiente.cloud.deployments.v1.GetBuildResponse"> & {
    /**
     * @generated from field: obiente.cloud.deployments.v1.Build build = 1;
     */
    build?: Build;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetBuildResponse.
 * Use `create(GetBuildResponseSchema)` to create a new message.
 */
export declare const GetBuildResponseSchema: GenMessage<GetBuildResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetBuildLogsRequest
 */
export type GetBuildLogsRequest = Message<"obiente.cloud.deployments.v1.GetBuildLogsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string build_id = 3;
     */
    buildId: string;
    /**
     * Maximum number of log lines to return (default: all)
     *
     * @generated from field: optional int32 limit = 4;
     */
    limit?: number;
    /**
     * Offset for pagination (for very large logs)
     *
     * @generated from field: optional int32 offset = 5;
     */
    offset?: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetBuildLogsRequest.
 * Use `create(GetBuildLogsRequestSchema)` to create a new message.
 */
export declare const GetBuildLogsRequestSchema: GenMessage<GetBuildLogsRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.GetBuildLogsResponse
 */
export type GetBuildLogsResponse = Message<"obiente.cloud.deployments.v1.GetBuildLogsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.deployments.v1.DeploymentLogLine logs = 1;
     */
    logs: DeploymentLogLine[];
    /**
     * Total number of log lines
     *
     * @generated from field: int32 total = 2;
     */
    total: number;
};
/**
 * Describes the message obiente.cloud.deployments.v1.GetBuildLogsResponse.
 * Use `create(GetBuildLogsResponseSchema)` to create a new message.
 */
export declare const GetBuildLogsResponseSchema: GenMessage<GetBuildLogsResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.RevertToBuildRequest
 */
export type RevertToBuildRequest = Message<"obiente.cloud.deployments.v1.RevertToBuildRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string build_id = 3;
     */
    buildId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RevertToBuildRequest.
 * Use `create(RevertToBuildRequestSchema)` to create a new message.
 */
export declare const RevertToBuildRequestSchema: GenMessage<RevertToBuildRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.RevertToBuildResponse
 */
export type RevertToBuildResponse = Message<"obiente.cloud.deployments.v1.RevertToBuildResponse"> & {
    /**
     * Updated deployment after revert
     *
     * @generated from field: obiente.cloud.deployments.v1.Deployment deployment = 1;
     */
    deployment?: Deployment;
    /**
     * ID of the new build created from the revert
     *
     * @generated from field: string new_build_id = 2;
     */
    newBuildId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.RevertToBuildResponse.
 * Use `create(RevertToBuildResponseSchema)` to create a new message.
 */
export declare const RevertToBuildResponseSchema: GenMessage<RevertToBuildResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteBuildRequest
 */
export type DeleteBuildRequest = Message<"obiente.cloud.deployments.v1.DeleteBuildRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string build_id = 3;
     */
    buildId: string;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteBuildRequest.
 * Use `create(DeleteBuildRequestSchema)` to create a new message.
 */
export declare const DeleteBuildRequestSchema: GenMessage<DeleteBuildRequest>;
/**
 * @generated from message obiente.cloud.deployments.v1.DeleteBuildResponse
 */
export type DeleteBuildResponse = Message<"obiente.cloud.deployments.v1.DeleteBuildResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.deployments.v1.DeleteBuildResponse.
 * Use `create(DeleteBuildResponseSchema)` to create a new message.
 */
export declare const DeleteBuildResponseSchema: GenMessage<DeleteBuildResponse>;
/**
 * @generated from message obiente.cloud.deployments.v1.Build
 */
export type Build = Message<"obiente.cloud.deployments.v1.Build"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string deployment_id = 2;
     */
    deploymentId: string;
    /**
     * @generated from field: string organization_id = 3;
     */
    organizationId: string;
    /**
     * Sequential build number per deployment
     *
     * @generated from field: int32 build_number = 4;
     */
    buildNumber: number;
    /**
     * @generated from field: obiente.cloud.deployments.v1.BuildStatus status = 5;
     */
    status: BuildStatus;
    /**
     * @generated from field: google.protobuf.Timestamp started_at = 6;
     */
    startedAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp completed_at = 7;
     */
    completedAt?: Timestamp;
    /**
     * Duration in seconds
     *
     * @generated from field: int32 build_time = 8;
     */
    buildTime: number;
    /**
     * User ID who triggered the build
     *
     * @generated from field: string triggered_by = 9;
     */
    triggeredBy: string;
    /**
     * Build configuration snapshot (captured at build time)
     *
     * @generated from field: optional string repository_url = 10;
     */
    repositoryUrl?: string;
    /**
     * @generated from field: string branch = 11;
     */
    branch: string;
    /**
     * Git commit SHA if available
     *
     * @generated from field: optional string commit_sha = 12;
     */
    commitSha?: string;
    /**
     * @generated from field: optional string build_command = 13;
     */
    buildCommand?: string;
    /**
     * @generated from field: optional string install_command = 14;
     */
    installCommand?: string;
    /**
     * @generated from field: optional string start_command = 15;
     */
    startCommand?: string;
    /**
     * @generated from field: optional string dockerfile_path = 16;
     */
    dockerfilePath?: string;
    /**
     * @generated from field: optional string compose_file_path = 17;
     */
    composeFilePath?: string;
    /**
     * @generated from field: obiente.cloud.deployments.v1.BuildStrategy build_strategy = 18;
     */
    buildStrategy: BuildStrategy;
    /**
     * Build results
     *
     * Built image name (for single container)
     *
     * @generated from field: optional string image_name = 19;
     */
    imageName?: string;
    /**
     * Docker Compose YAML (for compose deployments)
     *
     * @generated from field: optional string compose_yaml = 20;
     */
    composeYaml?: string;
    /**
     * Human-readable bundle size
     *
     * @generated from field: optional string size = 21;
     */
    size?: string;
    /**
     * Error message if build failed
     *
     * @generated from field: optional string error = 22;
     */
    error?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 23;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 24;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.deployments.v1.Build.
 * Use `create(BuildSchema)` to create a new message.
 */
export declare const BuildSchema: GenMessage<Build>;
/**
 * Enumerations for typed fields used by the frontend
 * DeploymentType represents the runtime/language type of the application
 *
 * @generated from enum obiente.cloud.deployments.v1.DeploymentType
 */
export declare enum DeploymentType {
    /**
     * @generated from enum value: DEPLOYMENT_TYPE_UNSPECIFIED = 0;
     */
    DEPLOYMENT_TYPE_UNSPECIFIED = 0,
    /**
     * Docker container (from Dockerfile or compose)
     *
     * @generated from enum value: DOCKER = 1;
     */
    DOCKER = 1,
    /**
     * Static site hosting
     *
     * @generated from enum value: STATIC = 2;
     */
    STATIC = 2,
    /**
     * Node.js application
     *
     * @generated from enum value: NODE = 3;
     */
    NODE = 3,
    /**
     * Go application
     *
     * @generated from enum value: GO = 4;
     */
    GO = 4,
    /**
     * Python application
     *
     * @generated from enum value: PYTHON = 5;
     */
    PYTHON = 5,
    /**
     * Ruby/Rails application
     *
     * @generated from enum value: RUBY = 6;
     */
    RUBY = 6,
    /**
     * Rust application
     *
     * @generated from enum value: RUST = 7;
     */
    RUST = 7,
    /**
     * Java application
     *
     * @generated from enum value: JAVA = 8;
     */
    JAVA = 8,
    /**
     * PHP application
     *
     * @generated from enum value: PHP = 9;
     */
    PHP = 9,
    /**
     * Generic/unknown runtime (use when can't detect)
     *
     * @generated from enum value: GENERIC = 10;
     */
    GENERIC = 10
}
/**
 * Describes the enum obiente.cloud.deployments.v1.DeploymentType.
 */
export declare const DeploymentTypeSchema: GenEnum<DeploymentType>;
/**
 * Build strategy determines how the deployment is built and deployed
 *
 * @generated from enum obiente.cloud.deployments.v1.BuildStrategy
 */
export declare enum BuildStrategy {
    /**
     * @generated from enum value: BUILD_STRATEGY_UNSPECIFIED = 0;
     */
    BUILD_STRATEGY_UNSPECIFIED = 0,
    /**
     * Railpack (Nixpacks variant, can build any language including Rails)
     *
     * @generated from enum value: RAILPACK = 1;
     */
    RAILPACK = 1,
    /**
     * Nixpacks buildpacks (can build any language)
     *
     * @generated from enum value: NIXPACKS = 2;
     */
    NIXPACKS = 2,
    /**
     * Build from Dockerfile
     *
     * @generated from enum value: DOCKERFILE = 3;
     */
    DOCKERFILE = 3,
    /**
     * Plain Docker Compose (uses compose YAML from database, no repository)
     *
     * @generated from enum value: PLAIN_COMPOSE = 4;
     */
    PLAIN_COMPOSE = 4,
    /**
     * Compose from Repository (clones repo and uses compose file from repo)
     *
     * @generated from enum value: COMPOSE_REPO = 6;
     */
    COMPOSE_REPO = 6,
    /**
     * Static site hosting
     *
     * @generated from enum value: STATIC_SITE = 5;
     */
    STATIC_SITE = 5
}
/**
 * Describes the enum obiente.cloud.deployments.v1.BuildStrategy.
 */
export declare const BuildStrategySchema: GenEnum<BuildStrategy>;
/**
 * @generated from enum obiente.cloud.deployments.v1.Environment
 */
export declare enum Environment {
    /**
     * @generated from enum value: ENVIRONMENT_UNSPECIFIED = 0;
     */
    ENVIRONMENT_UNSPECIFIED = 0,
    /**
     * @generated from enum value: PRODUCTION = 1;
     */
    PRODUCTION = 1,
    /**
     * @generated from enum value: STAGING = 2;
     */
    STAGING = 2,
    /**
     * @generated from enum value: DEVELOPMENT = 3;
     */
    DEVELOPMENT = 3
}
/**
 * Describes the enum obiente.cloud.deployments.v1.Environment.
 */
export declare const EnvironmentSchema: GenEnum<Environment>;
/**
 * @generated from enum obiente.cloud.deployments.v1.DeploymentStatus
 */
export declare enum DeploymentStatus {
    /**
     * @generated from enum value: DEPLOYMENT_STATUS_UNSPECIFIED = 0;
     */
    DEPLOYMENT_STATUS_UNSPECIFIED = 0,
    /**
     * @generated from enum value: CREATED = 1;
     */
    CREATED = 1,
    /**
     * @generated from enum value: BUILDING = 2;
     */
    BUILDING = 2,
    /**
     * @generated from enum value: RUNNING = 3;
     */
    RUNNING = 3,
    /**
     * @generated from enum value: STOPPED = 4;
     */
    STOPPED = 4,
    /**
     * @generated from enum value: FAILED = 5;
     */
    FAILED = 5,
    /**
     * @generated from enum value: DEPLOYING = 6;
     */
    DEPLOYING = 6
}
/**
 * Describes the enum obiente.cloud.deployments.v1.DeploymentStatus.
 */
export declare const DeploymentStatusSchema: GenEnum<DeploymentStatus>;
/**
 * @generated from enum obiente.cloud.deployments.v1.BuildStatus
 */
export declare enum BuildStatus {
    /**
     * @generated from enum value: BUILD_STATUS_UNSPECIFIED = 0;
     */
    BUILD_STATUS_UNSPECIFIED = 0,
    /**
     * @generated from enum value: BUILD_PENDING = 1;
     */
    BUILD_PENDING = 1,
    /**
     * @generated from enum value: BUILD_BUILDING = 2;
     */
    BUILD_BUILDING = 2,
    /**
     * @generated from enum value: BUILD_SUCCESS = 3;
     */
    BUILD_SUCCESS = 3,
    /**
     * @generated from enum value: BUILD_FAILED = 4;
     */
    BUILD_FAILED = 4
}
/**
 * Describes the enum obiente.cloud.deployments.v1.BuildStatus.
 */
export declare const BuildStatusSchema: GenEnum<BuildStatus>;
/**
 * Health check type for deployments
 *
 * @generated from enum obiente.cloud.deployments.v1.HealthCheckType
 */
export declare enum HealthCheckType {
    /**
     * Auto-detect (TCP if routing exists, otherwise no healthcheck)
     *
     * @generated from enum value: HEALTHCHECK_TYPE_UNSPECIFIED = 0;
     */
    HEALTHCHECK_TYPE_UNSPECIFIED = 0,
    /**
     * Explicitly disabled
     *
     * @generated from enum value: HEALTHCHECK_DISABLED = 1;
     */
    HEALTHCHECK_DISABLED = 1,
    /**
     * TCP port check (nc)
     *
     * @generated from enum value: HEALTHCHECK_TCP = 2;
     */
    HEALTHCHECK_TCP = 2,
    /**
     * HTTP endpoint check
     *
     * @generated from enum value: HEALTHCHECK_HTTP = 3;
     */
    HEALTHCHECK_HTTP = 3,
    /**
     * Custom command
     *
     * @generated from enum value: HEALTHCHECK_CUSTOM = 4;
     */
    HEALTHCHECK_CUSTOM = 4
}
/**
 * Describes the enum obiente.cloud.deployments.v1.HealthCheckType.
 */
export declare const HealthCheckTypeSchema: GenEnum<HealthCheckType>;
/**
 * @generated from enum obiente.cloud.deployments.v1.ContainerEntryType
 */
export declare enum ContainerEntryType {
    /**
     * @generated from enum value: CONTAINER_ENTRY_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: CONTAINER_ENTRY_TYPE_FILE = 1;
     */
    FILE = 1,
    /**
     * @generated from enum value: CONTAINER_ENTRY_TYPE_DIRECTORY = 2;
     */
    DIRECTORY = 2,
    /**
     * @generated from enum value: CONTAINER_ENTRY_TYPE_SYMLINK = 3;
     */
    SYMLINK = 3
}
/**
 * Describes the enum obiente.cloud.deployments.v1.ContainerEntryType.
 */
export declare const ContainerEntryTypeSchema: GenEnum<ContainerEntryType>;
/**
 * @generated from service obiente.cloud.deployments.v1.DeploymentService
 */
export declare const DeploymentService: GenService<{
    /**
     * List organization deployments
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListDeployments
     */
    listDeployments: {
        methodKind: "unary";
        input: typeof ListDeploymentsRequestSchema;
        output: typeof ListDeploymentsResponseSchema;
    };
    /**
     * Create new deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.CreateDeployment
     */
    createDeployment: {
        methodKind: "unary";
        input: typeof CreateDeploymentRequestSchema;
        output: typeof CreateDeploymentResponseSchema;
    };
    /**
     * Get deployment details
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeployment
     */
    getDeployment: {
        methodKind: "unary";
        input: typeof GetDeploymentRequestSchema;
        output: typeof GetDeploymentResponseSchema;
    };
    /**
     * Update deployment configuration
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.UpdateDeployment
     */
    updateDeployment: {
        methodKind: "unary";
        input: typeof UpdateDeploymentRequestSchema;
        output: typeof UpdateDeploymentResponseSchema;
    };
    /**
     * Trigger deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.TriggerDeployment
     */
    triggerDeployment: {
        methodKind: "unary";
        input: typeof TriggerDeploymentRequestSchema;
        output: typeof TriggerDeploymentResponseSchema;
    };
    /**
     * Stream deployment status updates
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamDeploymentStatus
     */
    streamDeploymentStatus: {
        methodKind: "server_streaming";
        input: typeof StreamDeploymentStatusRequestSchema;
        output: typeof DeploymentStatusUpdateSchema;
    };
    /**
     * Get deployment logs
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentLogs
     */
    getDeploymentLogs: {
        methodKind: "unary";
        input: typeof GetDeploymentLogsRequestSchema;
        output: typeof GetDeploymentLogsResponseSchema;
    };
    /**
     * Stream deployment logs (tail/follow)
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamDeploymentLogs
     */
    streamDeploymentLogs: {
        methodKind: "server_streaming";
        input: typeof StreamDeploymentLogsRequestSchema;
        output: typeof DeploymentLogLineSchema;
    };
    /**
     * Stream build logs during deployment (live build output)
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamBuildLogs
     */
    streamBuildLogs: {
        methodKind: "server_streaming";
        input: typeof StreamBuildLogsRequestSchema;
        output: typeof DeploymentLogLineSchema;
    };
    /**
     * Get deployment metrics (real-time or historical)
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentMetrics
     */
    getDeploymentMetrics: {
        methodKind: "unary";
        input: typeof GetDeploymentMetricsRequestSchema;
        output: typeof GetDeploymentMetricsResponseSchema;
    };
    /**
     * Stream real-time deployment metrics
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamDeploymentMetrics
     */
    streamDeploymentMetrics: {
        methodKind: "server_streaming";
        input: typeof StreamDeploymentMetricsRequestSchema;
        output: typeof DeploymentMetricSchema;
    };
    /**
     * Get aggregated usage for a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentUsage
     */
    getDeploymentUsage: {
        methodKind: "unary";
        input: typeof GetDeploymentUsageRequestSchema;
        output: typeof GetDeploymentUsageResponseSchema;
    };
    /**
     * Start a stopped deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StartDeployment
     */
    startDeployment: {
        methodKind: "unary";
        input: typeof StartDeploymentRequestSchema;
        output: typeof StartDeploymentResponseSchema;
    };
    /**
     * Stop a running deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StopDeployment
     */
    stopDeployment: {
        methodKind: "unary";
        input: typeof StopDeploymentRequestSchema;
        output: typeof StopDeploymentResponseSchema;
    };
    /**
     * Delete a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.DeleteDeployment
     */
    deleteDeployment: {
        methodKind: "unary";
        input: typeof DeleteDeploymentRequestSchema;
        output: typeof DeleteDeploymentResponseSchema;
    };
    /**
     * Restart a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.RestartDeployment
     */
    restartDeployment: {
        methodKind: "unary";
        input: typeof RestartDeploymentRequestSchema;
        output: typeof RestartDeploymentResponseSchema;
    };
    /**
     * Scale a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ScaleDeployment
     */
    scaleDeployment: {
        methodKind: "unary";
        input: typeof ScaleDeploymentRequestSchema;
        output: typeof ScaleDeploymentResponseSchema;
    };
    /**
     * Get deployment environment variables
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentEnvVars
     */
    getDeploymentEnvVars: {
        methodKind: "unary";
        input: typeof GetDeploymentEnvVarsRequestSchema;
        output: typeof GetDeploymentEnvVarsResponseSchema;
    };
    /**
     * Update deployment environment variables
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.UpdateDeploymentEnvVars
     */
    updateDeploymentEnvVars: {
        methodKind: "unary";
        input: typeof UpdateDeploymentEnvVarsRequestSchema;
        output: typeof UpdateDeploymentEnvVarsResponseSchema;
    };
    /**
     * Get deployment docker compose configuration
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentCompose
     */
    getDeploymentCompose: {
        methodKind: "unary";
        input: typeof GetDeploymentComposeRequestSchema;
        output: typeof GetDeploymentComposeResponseSchema;
    };
    /**
     * Validate deployment docker compose configuration (validation only, no save)
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ValidateDeploymentCompose
     */
    validateDeploymentCompose: {
        methodKind: "unary";
        input: typeof ValidateDeploymentComposeRequestSchema;
        output: typeof ValidateDeploymentComposeResponseSchema;
    };
    /**
     * Update deployment docker compose configuration
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.UpdateDeploymentCompose
     */
    updateDeploymentCompose: {
        methodKind: "unary";
        input: typeof UpdateDeploymentComposeRequestSchema;
        output: typeof UpdateDeploymentComposeResponseSchema;
    };
    /**
     * GitHub integration
     * List GitHub repositories for the authenticated user
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListGitHubRepos
     */
    listGitHubRepos: {
        methodKind: "unary";
        input: typeof ListGitHubReposRequestSchema;
        output: typeof ListGitHubReposResponseSchema;
    };
    /**
     * Get branches for a GitHub repository
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetGitHubBranches
     */
    getGitHubBranches: {
        methodKind: "unary";
        input: typeof GetGitHubBranchesRequestSchema;
        output: typeof GetGitHubBranchesResponseSchema;
    };
    /**
     * Get file content from a GitHub repository
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetGitHubFile
     */
    getGitHubFile: {
        methodKind: "unary";
        input: typeof GetGitHubFileRequestSchema;
        output: typeof GetGitHubFileResponseSchema;
    };
    /**
     * Build history management
     * List builds for a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListBuilds
     */
    listBuilds: {
        methodKind: "unary";
        input: typeof ListBuildsRequestSchema;
        output: typeof ListBuildsResponseSchema;
    };
    /**
     * Get details of a specific build
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetBuild
     */
    getBuild: {
        methodKind: "unary";
        input: typeof GetBuildRequestSchema;
        output: typeof GetBuildResponseSchema;
    };
    /**
     * Get build logs for a specific build
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetBuildLogs
     */
    getBuildLogs: {
        methodKind: "unary";
        input: typeof GetBuildLogsRequestSchema;
        output: typeof GetBuildLogsResponseSchema;
    };
    /**
     * Revert deployment to a previous build
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.RevertToBuild
     */
    revertToBuild: {
        methodKind: "unary";
        input: typeof RevertToBuildRequestSchema;
        output: typeof RevertToBuildResponseSchema;
    };
    /**
     * Delete a build from history
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.DeleteBuild
     */
    deleteBuild: {
        methodKind: "unary";
        input: typeof DeleteBuildRequestSchema;
        output: typeof DeleteBuildResponseSchema;
    };
    /**
     * List all available GitHub integrations for the current user
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListAvailableGitHubIntegrations
     */
    listAvailableGitHubIntegrations: {
        methodKind: "unary";
        input: typeof ListAvailableGitHubIntegrationsRequestSchema;
        output: typeof ListAvailableGitHubIntegrationsResponseSchema;
    };
    /**
     * Terminal access
     * Stream terminal with bidirectional streaming (replaces StreamTerminalOutput + SendTerminalInput)
     * Use this for better input/output synchronization
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamTerminal
     */
    streamTerminal: {
        methodKind: "bidi_streaming";
        input: typeof TerminalInputSchema;
        output: typeof TerminalOutputSchema;
    };
    /**
     * DEPRECATED: Use StreamTerminal instead for bidirectional streaming
     * Stream terminal output from a deployment container
     * Input is sent via SendTerminalInput RPC
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamTerminalOutput
     */
    streamTerminalOutput: {
        methodKind: "server_streaming";
        input: typeof StreamTerminalOutputRequestSchema;
        output: typeof TerminalOutputSchema;
    };
    /**
     * DEPRECATED: Use StreamTerminal instead for bidirectional streaming
     * Send input to an active terminal session
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.SendTerminalInput
     */
    sendTerminalInput: {
        methodKind: "unary";
        input: typeof SendTerminalInputRequestSchema;
        output: typeof SendTerminalInputResponseSchema;
    };
    /**
     * File browser
     * List files in a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListContainerFiles
     */
    listContainerFiles: {
        methodKind: "unary";
        input: typeof ListContainerFilesRequestSchema;
        output: typeof ListContainerFilesResponseSchema;
    };
    /**
     * Get file content from a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetContainerFile
     */
    getContainerFile: {
        methodKind: "unary";
        input: typeof GetContainerFileRequestSchema;
        output: typeof GetContainerFileResponseSchema;
    };
    /**
     * Upload files to a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.UploadContainerFiles
     */
    uploadContainerFiles: {
        methodKind: "unary";
        input: typeof UploadContainerFilesRequestSchema;
        output: typeof UploadContainerFilesResponseSchema;
    };
    /**
     * Chunk-based file upload for deployment containers using shared payloads
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ChunkUploadContainerFiles
     */
    chunkUploadContainerFiles: {
        methodKind: "unary";
        input: typeof ChunkUploadContainerFilesRequestSchema;
        output: typeof ChunkUploadContainerFilesResponseSchema;
    };
    /**
     * Delete one or more files/directories in a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.DeleteContainerEntries
     */
    deleteContainerEntries: {
        methodKind: "unary";
        input: typeof DeleteContainerEntriesRequestSchema;
        output: typeof DeleteContainerEntriesResponseSchema;
    };
    /**
     * Rename or move a file/directory within a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.RenameContainerEntry
     */
    renameContainerEntry: {
        methodKind: "unary";
        input: typeof RenameContainerEntryRequestSchema;
        output: typeof RenameContainerEntryResponseSchema;
    };
    /**
     * Create an empty file or directory within a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.CreateContainerEntry
     */
    createContainerEntry: {
        methodKind: "unary";
        input: typeof CreateContainerEntryRequestSchema;
        output: typeof CreateContainerEntryResponseSchema;
    };
    /**
     * Write/update file contents in a deployment container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.WriteContainerFile
     */
    writeContainerFile: {
        methodKind: "unary";
        input: typeof WriteContainerFileRequestSchema;
        output: typeof WriteContainerFileResponseSchema;
    };
    /**
     * Extract a zip file to a destination directory
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ExtractDeploymentFile
     */
    extractDeploymentFile: {
        methodKind: "unary";
        input: typeof ExtractDeploymentFileRequestSchema;
        output: typeof ExtractDeploymentFileResponseSchema;
    };
    /**
     * Create a zip archive from files or folders
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.CreateDeploymentFileArchive
     */
    createDeploymentFileArchive: {
        methodKind: "unary";
        input: typeof CreateDeploymentFileArchiveRequestSchema;
        output: typeof CreateDeploymentFileArchiveResponseSchema;
    };
    /**
     * Routing configuration
     * Get all routing rules for a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentRoutings
     */
    getDeploymentRoutings: {
        methodKind: "unary";
        input: typeof GetDeploymentRoutingsRequestSchema;
        output: typeof GetDeploymentRoutingsResponseSchema;
    };
    /**
     * Update routing rules for a deployment (replaces all existing rules)
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.UpdateDeploymentRoutings
     */
    updateDeploymentRoutings: {
        methodKind: "unary";
        input: typeof UpdateDeploymentRoutingsRequestSchema;
        output: typeof UpdateDeploymentRoutingsResponseSchema;
    };
    /**
     * Get available service names from Docker Compose
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDeploymentServiceNames
     */
    getDeploymentServiceNames: {
        methodKind: "unary";
        input: typeof GetDeploymentServiceNamesRequestSchema;
        output: typeof GetDeploymentServiceNamesResponseSchema;
    };
    /**
     * Domain verification
     * Get verification token for a custom domain
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.GetDomainVerificationToken
     */
    getDomainVerificationToken: {
        methodKind: "unary";
        input: typeof GetDomainVerificationTokenRequestSchema;
        output: typeof GetDomainVerificationTokenResponseSchema;
    };
    /**
     * Verify domain ownership via DNS TXT record
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.VerifyDomainOwnership
     */
    verifyDomainOwnership: {
        methodKind: "unary";
        input: typeof VerifyDomainOwnershipRequestSchema;
        output: typeof VerifyDomainOwnershipResponseSchema;
    };
    /**
     * List all containers for a deployment
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.ListDeploymentContainers
     */
    listDeploymentContainers: {
        methodKind: "unary";
        input: typeof ListDeploymentContainersRequestSchema;
        output: typeof ListDeploymentContainersResponseSchema;
    };
    /**
     * Stream logs from a specific container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StreamContainerLogs
     */
    streamContainerLogs: {
        methodKind: "server_streaming";
        input: typeof StreamContainerLogsRequestSchema;
        output: typeof DeploymentLogLineSchema;
    };
    /**
     * Start a specific container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StartContainer
     */
    startContainer: {
        methodKind: "unary";
        input: typeof StartContainerRequestSchema;
        output: typeof StartContainerResponseSchema;
    };
    /**
     * Stop a specific container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.StopContainer
     */
    stopContainer: {
        methodKind: "unary";
        input: typeof StopContainerRequestSchema;
        output: typeof StopContainerResponseSchema;
    };
    /**
     * Restart a specific container
     *
     * @generated from rpc obiente.cloud.deployments.v1.DeploymentService.RestartContainer
     */
    restartContainer: {
        methodKind: "unary";
        input: typeof RestartContainerRequestSchema;
        output: typeof RestartContainerResponseSchema;
    };
}>;
