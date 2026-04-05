import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "../../../../google/protobuf/timestamp_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/audit/v1/audit_service.proto.
 */
export declare const file_obiente_cloud_audit_v1_audit_service: GenFile;
/**
 * Request/Response messages
 *
 * @generated from message obiente.cloud.audit.v1.ListAuditLogsRequest
 */
export type ListAuditLogsRequest = Message<"obiente.cloud.audit.v1.ListAuditLogsRequest"> & {
    /**
     * Filter by organization ID (optional)
     *
     * @generated from field: optional string organization_id = 1;
     */
    organizationId?: string;
    /**
     * Filter by resource type (optional, e.g., "deployment", "organization")
     *
     * @generated from field: optional string resource_type = 2;
     */
    resourceType?: string;
    /**
     * Filter by resource ID (optional)
     *
     * @generated from field: optional string resource_id = 3;
     */
    resourceId?: string;
    /**
     * Filter by user ID (optional)
     *
     * @generated from field: optional string user_id = 4;
     */
    userId?: string;
    /**
     * Filter by service name (optional, e.g., "DeploymentService")
     *
     * @generated from field: optional string service = 5;
     */
    service?: string;
    /**
     * Filter by action name (optional, e.g., "CreateDeployment")
     *
     * @generated from field: optional string action = 6;
     */
    action?: string;
    /**
     * Filter by time range (optional)
     *
     * @generated from field: optional google.protobuf.Timestamp start_time = 7;
     */
    startTime?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp end_time = 8;
     */
    endTime?: Timestamp;
    /**
     * Pagination
     *
     * Default: 50, Max: 1000
     *
     * @generated from field: optional int32 page_size = 9;
     */
    pageSize?: number;
    /**
     * Token from previous response for pagination
     *
     * @generated from field: optional string page_token = 10;
     */
    pageToken?: string;
};
/**
 * Describes the message obiente.cloud.audit.v1.ListAuditLogsRequest.
 * Use `create(ListAuditLogsRequestSchema)` to create a new message.
 */
export declare const ListAuditLogsRequestSchema: GenMessage<ListAuditLogsRequest>;
/**
 * @generated from message obiente.cloud.audit.v1.ListAuditLogsResponse
 */
export type ListAuditLogsResponse = Message<"obiente.cloud.audit.v1.ListAuditLogsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.audit.v1.AuditLogEntry audit_logs = 1;
     */
    auditLogs: AuditLogEntry[];
    /**
     * Token for next page, empty if no more pages
     *
     * @generated from field: string next_page_token = 2;
     */
    nextPageToken: string;
    /**
     * Total number of audit logs matching the filters (before pagination)
     *
     * @generated from field: int64 total_count = 3;
     */
    totalCount: bigint;
};
/**
 * Describes the message obiente.cloud.audit.v1.ListAuditLogsResponse.
 * Use `create(ListAuditLogsResponseSchema)` to create a new message.
 */
export declare const ListAuditLogsResponseSchema: GenMessage<ListAuditLogsResponse>;
/**
 * @generated from message obiente.cloud.audit.v1.GetAuditLogRequest
 */
export type GetAuditLogRequest = Message<"obiente.cloud.audit.v1.GetAuditLogRequest"> & {
    /**
     * @generated from field: string audit_log_id = 1;
     */
    auditLogId: string;
};
/**
 * Describes the message obiente.cloud.audit.v1.GetAuditLogRequest.
 * Use `create(GetAuditLogRequestSchema)` to create a new message.
 */
export declare const GetAuditLogRequestSchema: GenMessage<GetAuditLogRequest>;
/**
 * @generated from message obiente.cloud.audit.v1.GetAuditLogResponse
 */
export type GetAuditLogResponse = Message<"obiente.cloud.audit.v1.GetAuditLogResponse"> & {
    /**
     * @generated from field: obiente.cloud.audit.v1.AuditLogEntry audit_log = 1;
     */
    auditLog?: AuditLogEntry;
};
/**
 * Describes the message obiente.cloud.audit.v1.GetAuditLogResponse.
 * Use `create(GetAuditLogResponseSchema)` to create a new message.
 */
export declare const GetAuditLogResponseSchema: GenMessage<GetAuditLogResponse>;
/**
 * AuditLogEntry represents a single audit log entry
 *
 * @generated from message obiente.cloud.audit.v1.AuditLogEntry
 */
export type AuditLogEntry = Message<"obiente.cloud.audit.v1.AuditLogEntry"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string user_id = 2;
     */
    userId: string;
    /**
     * Resolved user name (for display)
     *
     * @generated from field: optional string user_name = 15;
     */
    userName?: string;
    /**
     * Resolved user email (for display)
     *
     * @generated from field: optional string user_email = 16;
     */
    userEmail?: string;
    /**
     * @generated from field: optional string organization_id = 3;
     */
    organizationId?: string;
    /**
     * RPC method name (e.g., "CreateDeployment")
     *
     * @generated from field: string action = 4;
     */
    action: string;
    /**
     * Service name (e.g., "DeploymentService")
     *
     * @generated from field: string service = 5;
     */
    service: string;
    /**
     * Type of resource affected
     *
     * @generated from field: optional string resource_type = 6;
     */
    resourceType?: string;
    /**
     * ID of the affected resource
     *
     * @generated from field: optional string resource_id = 7;
     */
    resourceId?: string;
    /**
     * @generated from field: string ip_address = 8;
     */
    ipAddress: string;
    /**
     * @generated from field: string user_agent = 9;
     */
    userAgent: string;
    /**
     * JSON-encoded request data (sanitized)
     *
     * @generated from field: string request_data = 10;
     */
    requestData: string;
    /**
     * HTTP/Connect status code
     *
     * @generated from field: int32 response_status = 11;
     */
    responseStatus: number;
    /**
     * Error message if action failed
     *
     * @generated from field: optional string error_message = 12;
     */
    errorMessage?: string;
    /**
     * Request duration in milliseconds
     *
     * @generated from field: int64 duration_ms = 13;
     */
    durationMs: bigint;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 14;
     */
    createdAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.audit.v1.AuditLogEntry.
 * Use `create(AuditLogEntrySchema)` to create a new message.
 */
export declare const AuditLogEntrySchema: GenMessage<AuditLogEntry>;
/**
 * @generated from service obiente.cloud.audit.v1.AuditService
 */
export declare const AuditService: GenService<{
    /**
     * List audit logs with filtering options
     *
     * @generated from rpc obiente.cloud.audit.v1.AuditService.ListAuditLogs
     */
    listAuditLogs: {
        methodKind: "unary";
        input: typeof ListAuditLogsRequestSchema;
        output: typeof ListAuditLogsResponseSchema;
    };
    /**
     * Get a specific audit log entry by ID
     *
     * @generated from rpc obiente.cloud.audit.v1.AuditService.GetAuditLog
     */
    getAuditLog: {
        methodKind: "unary";
        input: typeof GetAuditLogRequestSchema;
        output: typeof GetAuditLogResponseSchema;
    };
}>;
