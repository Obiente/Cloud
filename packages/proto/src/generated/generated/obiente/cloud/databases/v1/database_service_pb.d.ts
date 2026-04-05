import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Pagination } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/databases/v1/database_service.proto.
 */
export declare const file_obiente_cloud_databases_v1_database_service: GenFile;
/**
 * @generated from message obiente.cloud.databases.v1.ListDatabasesRequest
 */
export type ListDatabasesRequest = Message<"obiente.cloud.databases.v1.ListDatabasesRequest"> & {
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
     * @generated from field: optional obiente.cloud.databases.v1.DatabaseStatus status = 4;
     */
    status?: DatabaseStatus;
    /**
     * Optional: filter by type
     *
     * @generated from field: optional obiente.cloud.databases.v1.DatabaseType type = 5;
     */
    type?: DatabaseType;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListDatabasesRequest.
 * Use `create(ListDatabasesRequestSchema)` to create a new message.
 */
export declare const ListDatabasesRequestSchema: GenMessage<ListDatabasesRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ListDatabasesResponse
 */
export type ListDatabasesResponse = Message<"obiente.cloud.databases.v1.ListDatabasesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.DatabaseInstance databases = 1;
     */
    databases: DatabaseInstance[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListDatabasesResponse.
 * Use `create(ListDatabasesResponseSchema)` to create a new message.
 */
export declare const ListDatabasesResponseSchema: GenMessage<ListDatabasesResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateDatabaseRequest
 */
export type CreateDatabaseRequest = Message<"obiente.cloud.databases.v1.CreateDatabaseRequest"> & {
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
     * @generated from field: obiente.cloud.databases.v1.DatabaseType type = 4;
     */
    type: DatabaseType;
    /**
     * Database size/spec (e.g., "small", "medium", "large")
     *
     * @generated from field: string size = 5;
     */
    size: string;
    /**
     * Database version (e.g., "15", "8.0", "7.0")
     *
     * @generated from field: optional string version = 6;
     */
    version?: string;
    /**
     * Additional metadata/tags
     *
     * @generated from field: map<string, string> metadata = 7;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * Initial username (if not provided, auto-generated)
     *
     * @generated from field: optional string initial_username = 9;
     */
    initialUsername?: string;
    /**
     * Initial password (if not provided, auto-generated)
     *
     * @generated from field: optional string initial_password = 10;
     */
    initialPassword?: string;
    /**
     * Auto-sleep after inactivity (0 = disabled)
     *
     * @generated from field: optional int32 auto_sleep_seconds = 11;
     */
    autoSleepSeconds?: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateDatabaseRequest.
 * Use `create(CreateDatabaseRequestSchema)` to create a new message.
 */
export declare const CreateDatabaseRequestSchema: GenMessage<CreateDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateDatabaseResponse
 */
export type CreateDatabaseResponse = Message<"obiente.cloud.databases.v1.CreateDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
    /**
     * Connection info returned only on creation
     *
     * @generated from field: obiente.cloud.databases.v1.DatabaseConnectionInfo connection_info = 2;
     */
    connectionInfo?: DatabaseConnectionInfo;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateDatabaseResponse.
 * Use `create(CreateDatabaseResponseSchema)` to create a new message.
 */
export declare const CreateDatabaseResponseSchema: GenMessage<CreateDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseRequest
 */
export type GetDatabaseRequest = Message<"obiente.cloud.databases.v1.GetDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseRequest.
 * Use `create(GetDatabaseRequestSchema)` to create a new message.
 */
export declare const GetDatabaseRequestSchema: GenMessage<GetDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseResponse
 */
export type GetDatabaseResponse = Message<"obiente.cloud.databases.v1.GetDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseResponse.
 * Use `create(GetDatabaseResponseSchema)` to create a new message.
 */
export declare const GetDatabaseResponseSchema: GenMessage<GetDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.UpdateDatabaseRequest
 */
export type UpdateDatabaseRequest = Message<"obiente.cloud.databases.v1.UpdateDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
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
    /**
     * Auto-sleep after inactivity (0 = disabled)
     *
     * @generated from field: optional int32 auto_sleep_seconds = 6;
     */
    autoSleepSeconds?: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.UpdateDatabaseRequest.
 * Use `create(UpdateDatabaseRequestSchema)` to create a new message.
 */
export declare const UpdateDatabaseRequestSchema: GenMessage<UpdateDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.UpdateDatabaseResponse
 */
export type UpdateDatabaseResponse = Message<"obiente.cloud.databases.v1.UpdateDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.UpdateDatabaseResponse.
 * Use `create(UpdateDatabaseResponseSchema)` to create a new message.
 */
export declare const UpdateDatabaseResponseSchema: GenMessage<UpdateDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteDatabaseRequest
 */
export type DeleteDatabaseRequest = Message<"obiente.cloud.databases.v1.DeleteDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * If true, force delete even if database is running
     *
     * @generated from field: bool force = 3;
     */
    force: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteDatabaseRequest.
 * Use `create(DeleteDatabaseRequestSchema)` to create a new message.
 */
export declare const DeleteDatabaseRequestSchema: GenMessage<DeleteDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteDatabaseResponse
 */
export type DeleteDatabaseResponse = Message<"obiente.cloud.databases.v1.DeleteDatabaseResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteDatabaseResponse.
 * Use `create(DeleteDatabaseResponseSchema)` to create a new message.
 */
export declare const DeleteDatabaseResponseSchema: GenMessage<DeleteDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.StartDatabaseRequest
 */
export type StartDatabaseRequest = Message<"obiente.cloud.databases.v1.StartDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.StartDatabaseRequest.
 * Use `create(StartDatabaseRequestSchema)` to create a new message.
 */
export declare const StartDatabaseRequestSchema: GenMessage<StartDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.StartDatabaseResponse
 */
export type StartDatabaseResponse = Message<"obiente.cloud.databases.v1.StartDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.StartDatabaseResponse.
 * Use `create(StartDatabaseResponseSchema)` to create a new message.
 */
export declare const StartDatabaseResponseSchema: GenMessage<StartDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.StopDatabaseRequest
 */
export type StopDatabaseRequest = Message<"obiente.cloud.databases.v1.StopDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.StopDatabaseRequest.
 * Use `create(StopDatabaseRequestSchema)` to create a new message.
 */
export declare const StopDatabaseRequestSchema: GenMessage<StopDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.StopDatabaseResponse
 */
export type StopDatabaseResponse = Message<"obiente.cloud.databases.v1.StopDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.StopDatabaseResponse.
 * Use `create(StopDatabaseResponseSchema)` to create a new message.
 */
export declare const StopDatabaseResponseSchema: GenMessage<StopDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.SleepDatabaseRequest
 */
export type SleepDatabaseRequest = Message<"obiente.cloud.databases.v1.SleepDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.SleepDatabaseRequest.
 * Use `create(SleepDatabaseRequestSchema)` to create a new message.
 */
export declare const SleepDatabaseRequestSchema: GenMessage<SleepDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.SleepDatabaseResponse
 */
export type SleepDatabaseResponse = Message<"obiente.cloud.databases.v1.SleepDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.SleepDatabaseResponse.
 * Use `create(SleepDatabaseResponseSchema)` to create a new message.
 */
export declare const SleepDatabaseResponseSchema: GenMessage<SleepDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.RestartDatabaseRequest
 */
export type RestartDatabaseRequest = Message<"obiente.cloud.databases.v1.RestartDatabaseRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.RestartDatabaseRequest.
 * Use `create(RestartDatabaseRequestSchema)` to create a new message.
 */
export declare const RestartDatabaseRequestSchema: GenMessage<RestartDatabaseRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.RestartDatabaseResponse
 */
export type RestartDatabaseResponse = Message<"obiente.cloud.databases.v1.RestartDatabaseResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
};
/**
 * Describes the message obiente.cloud.databases.v1.RestartDatabaseResponse.
 * Use `create(RestartDatabaseResponseSchema)` to create a new message.
 */
export declare const RestartDatabaseResponseSchema: GenMessage<RestartDatabaseResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.StreamDatabaseStatusRequest
 */
export type StreamDatabaseStatusRequest = Message<"obiente.cloud.databases.v1.StreamDatabaseStatusRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.StreamDatabaseStatusRequest.
 * Use `create(StreamDatabaseStatusRequestSchema)` to create a new message.
 */
export declare const StreamDatabaseStatusRequestSchema: GenMessage<StreamDatabaseStatusRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseStatusUpdate
 */
export type DatabaseStatusUpdate = Message<"obiente.cloud.databases.v1.DatabaseStatusUpdate"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseStatus status = 2;
     */
    status: DatabaseStatus;
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
 * Describes the message obiente.cloud.databases.v1.DatabaseStatusUpdate.
 * Use `create(DatabaseStatusUpdateSchema)` to create a new message.
 */
export declare const DatabaseStatusUpdateSchema: GenMessage<DatabaseStatusUpdate>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseConnectionInfoRequest
 */
export type GetDatabaseConnectionInfoRequest = Message<"obiente.cloud.databases.v1.GetDatabaseConnectionInfoRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseConnectionInfoRequest.
 * Use `create(GetDatabaseConnectionInfoRequestSchema)` to create a new message.
 */
export declare const GetDatabaseConnectionInfoRequestSchema: GenMessage<GetDatabaseConnectionInfoRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseConnectionInfoResponse
 */
export type GetDatabaseConnectionInfoResponse = Message<"obiente.cloud.databases.v1.GetDatabaseConnectionInfoResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseConnectionInfo connection_info = 1;
     */
    connectionInfo?: DatabaseConnectionInfo;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseConnectionInfoResponse.
 * Use `create(GetDatabaseConnectionInfoResponseSchema)` to create a new message.
 */
export declare const GetDatabaseConnectionInfoResponseSchema: GenMessage<GetDatabaseConnectionInfoResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseConnectionInfo
 */
export type DatabaseConnectionInfo = Message<"obiente.cloud.databases.v1.DatabaseConnectionInfo"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: string host = 2;
     */
    host: string;
    /**
     * @generated from field: int32 port = 3;
     */
    port: number;
    /**
     * @generated from field: string database_name = 4;
     */
    databaseName: string;
    /**
     * @generated from field: string username = 5;
     */
    username: string;
    /**
     * Only returned on creation or password reset
     *
     * @generated from field: string password = 6;
     */
    password: string;
    /**
     * Connection strings for different languages/frameworks
     *
     * postgresql://user:pass@host:port/dbname
     *
     * @generated from field: string postgresql_url = 7;
     */
    postgresqlUrl: string;
    /**
     * mysql://user:pass@host:port/dbname
     *
     * @generated from field: string mysql_url = 8;
     */
    mysqlUrl: string;
    /**
     * mongodb://user:pass@host:port/dbname
     *
     * @generated from field: string mongodb_url = 9;
     */
    mongodbUrl: string;
    /**
     * redis://:pass@host:port
     *
     * @generated from field: string redis_url = 10;
     */
    redisUrl: string;
    /**
     * Additional connection info
     *
     * @generated from field: bool ssl_required = 11;
     */
    sslRequired: boolean;
    /**
     * CA certificate if needed
     *
     * @generated from field: optional string ssl_certificate = 12;
     */
    sslCertificate?: string;
    /**
     * Human-readable instructions
     *
     * @generated from field: string connection_instructions = 13;
     */
    connectionInstructions: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.DatabaseConnectionInfo.
 * Use `create(DatabaseConnectionInfoSchema)` to create a new message.
 */
export declare const DatabaseConnectionInfoSchema: GenMessage<DatabaseConnectionInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ResetDatabasePasswordRequest
 */
export type ResetDatabasePasswordRequest = Message<"obiente.cloud.databases.v1.ResetDatabasePasswordRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * If not provided, resets root/admin password
     *
     * @generated from field: optional string username = 3;
     */
    username?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ResetDatabasePasswordRequest.
 * Use `create(ResetDatabasePasswordRequestSchema)` to create a new message.
 */
export declare const ResetDatabasePasswordRequestSchema: GenMessage<ResetDatabasePasswordRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ResetDatabasePasswordResponse
 */
export type ResetDatabasePasswordResponse = Message<"obiente.cloud.databases.v1.ResetDatabasePasswordResponse"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: string username = 2;
     */
    username: string;
    /**
     * New password (shown once, never stored)
     *
     * @generated from field: string new_password = 3;
     */
    newPassword: string;
    /**
     * Instructions for applying the password change
     *
     * @generated from field: string message = 4;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ResetDatabasePasswordResponse.
 * Use `create(ResetDatabasePasswordResponseSchema)` to create a new message.
 */
export declare const ResetDatabasePasswordResponseSchema: GenMessage<ResetDatabasePasswordResponse>;
/**
 * Schema introspection messages
 *
 * @generated from message obiente.cloud.databases.v1.GetDatabaseSchemaRequest
 */
export type GetDatabaseSchemaRequest = Message<"obiente.cloud.databases.v1.GetDatabaseSchemaRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * Specific database name (for multi-database engines)
     *
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseSchemaRequest.
 * Use `create(GetDatabaseSchemaRequestSchema)` to create a new message.
 */
export declare const GetDatabaseSchemaRequestSchema: GenMessage<GetDatabaseSchemaRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseSchemaResponse
 */
export type GetDatabaseSchemaResponse = Message<"obiente.cloud.databases.v1.GetDatabaseSchemaResponse"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 2;
     */
    databaseName?: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.TableInfo tables = 3;
     */
    tables: TableInfo[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ViewInfo views = 4;
     */
    views: ViewInfo[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.FunctionInfo functions = 5;
     */
    functions: FunctionInfo[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ProcedureInfo procedures = 6;
     */
    procedures: ProcedureInfo[];
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseSchemaResponse.
 * Use `create(GetDatabaseSchemaResponseSchema)` to create a new message.
 */
export declare const GetDatabaseSchemaResponseSchema: GenMessage<GetDatabaseSchemaResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.ListTablesRequest
 */
export type ListTablesRequest = Message<"obiente.cloud.databases.v1.ListTablesRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * Specific database name
     *
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: int32 page = 4;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 5;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListTablesRequest.
 * Use `create(ListTablesRequestSchema)` to create a new message.
 */
export declare const ListTablesRequestSchema: GenMessage<ListTablesRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ListTablesResponse
 */
export type ListTablesResponse = Message<"obiente.cloud.databases.v1.ListTablesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.TableInfo tables = 1;
     */
    tables: TableInfo[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListTablesResponse.
 * Use `create(ListTablesResponseSchema)` to create a new message.
 */
export declare const ListTablesResponseSchema: GenMessage<ListTablesResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.GetTableStructureRequest
 */
export type GetTableStructureRequest = Message<"obiente.cloud.databases.v1.GetTableStructureRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string table_name = 3;
     */
    tableName: string;
    /**
     * Specific database name
     *
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableStructureRequest.
 * Use `create(GetTableStructureRequestSchema)` to create a new message.
 */
export declare const GetTableStructureRequestSchema: GenMessage<GetTableStructureRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetTableStructureResponse
 */
export type GetTableStructureResponse = Message<"obiente.cloud.databases.v1.GetTableStructureResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.TableInfo table = 1;
     */
    table?: TableInfo;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableStructureResponse.
 * Use `create(GetTableStructureResponseSchema)` to create a new message.
 */
export declare const GetTableStructureResponseSchema: GenMessage<GetTableStructureResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.TableInfo
 */
export type TableInfo = Message<"obiente.cloud.databases.v1.TableInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * Schema/database name
     *
     * @generated from field: string schema = 2;
     */
    schema: string;
    /**
     * "table", "view", etc.
     *
     * @generated from field: string type = 3;
     */
    type: string;
    /**
     * Approximate row count
     *
     * @generated from field: int64 row_count = 4;
     */
    rowCount: bigint;
    /**
     * Table size in bytes
     *
     * @generated from field: int64 size_bytes = 5;
     */
    sizeBytes: bigint;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ColumnInfo columns = 6;
     */
    columns: ColumnInfo[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.IndexInfo indexes = 7;
     */
    indexes: IndexInfo[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ForeignKeyInfo foreign_keys = 8;
     */
    foreignKeys: ForeignKeyInfo[];
};
/**
 * Describes the message obiente.cloud.databases.v1.TableInfo.
 * Use `create(TableInfoSchema)` to create a new message.
 */
export declare const TableInfoSchema: GenMessage<TableInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ColumnInfo
 */
export type ColumnInfo = Message<"obiente.cloud.databases.v1.ColumnInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * e.g., "varchar(255)", "int", "timestamp"
     *
     * @generated from field: string data_type = 2;
     */
    dataType: string;
    /**
     * @generated from field: bool is_nullable = 3;
     */
    isNullable: boolean;
    /**
     * @generated from field: optional string default_value = 4;
     */
    defaultValue?: string;
    /**
     * @generated from field: bool is_primary_key = 5;
     */
    isPrimaryKey: boolean;
    /**
     * @generated from field: bool is_unique = 6;
     */
    isUnique: boolean;
    /**
     * @generated from field: optional string comment = 7;
     */
    comment?: string;
    /**
     * @generated from field: int32 ordinal_position = 8;
     */
    ordinalPosition: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.ColumnInfo.
 * Use `create(ColumnInfoSchema)` to create a new message.
 */
export declare const ColumnInfoSchema: GenMessage<ColumnInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.IndexInfo
 */
export type IndexInfo = Message<"obiente.cloud.databases.v1.IndexInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: bool is_unique = 2;
     */
    isUnique: boolean;
    /**
     * @generated from field: bool is_primary = 3;
     */
    isPrimary: boolean;
    /**
     * @generated from field: repeated string column_names = 4;
     */
    columnNames: string[];
    /**
     * "btree", "hash", etc.
     *
     * @generated from field: optional string type = 5;
     */
    type?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.IndexInfo.
 * Use `create(IndexInfoSchema)` to create a new message.
 */
export declare const IndexInfoSchema: GenMessage<IndexInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ForeignKeyInfo
 */
export type ForeignKeyInfo = Message<"obiente.cloud.databases.v1.ForeignKeyInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string from_table = 2;
     */
    fromTable: string;
    /**
     * @generated from field: repeated string from_columns = 3;
     */
    fromColumns: string[];
    /**
     * @generated from field: string to_table = 4;
     */
    toTable: string;
    /**
     * @generated from field: repeated string to_columns = 5;
     */
    toColumns: string[];
    /**
     * "CASCADE", "RESTRICT", etc.
     *
     * @generated from field: optional string on_delete = 6;
     */
    onDelete?: string;
    /**
     * @generated from field: optional string on_update = 7;
     */
    onUpdate?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ForeignKeyInfo.
 * Use `create(ForeignKeyInfoSchema)` to create a new message.
 */
export declare const ForeignKeyInfoSchema: GenMessage<ForeignKeyInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ViewInfo
 */
export type ViewInfo = Message<"obiente.cloud.databases.v1.ViewInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string schema = 2;
     */
    schema: string;
    /**
     * SQL definition
     *
     * @generated from field: string definition = 3;
     */
    definition: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ViewInfo.
 * Use `create(ViewInfoSchema)` to create a new message.
 */
export declare const ViewInfoSchema: GenMessage<ViewInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.FunctionInfo
 */
export type FunctionInfo = Message<"obiente.cloud.databases.v1.FunctionInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string schema = 2;
     */
    schema: string;
    /**
     * @generated from field: string return_type = 3;
     */
    returnType: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ParameterInfo parameters = 4;
     */
    parameters: ParameterInfo[];
    /**
     * Function body/definition
     *
     * @generated from field: string definition = 5;
     */
    definition: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.FunctionInfo.
 * Use `create(FunctionInfoSchema)` to create a new message.
 */
export declare const FunctionInfoSchema: GenMessage<FunctionInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ProcedureInfo
 */
export type ProcedureInfo = Message<"obiente.cloud.databases.v1.ProcedureInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string schema = 2;
     */
    schema: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ParameterInfo parameters = 3;
     */
    parameters: ParameterInfo[];
    /**
     * Procedure body/definition
     *
     * @generated from field: string definition = 5;
     */
    definition: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ProcedureInfo.
 * Use `create(ProcedureInfoSchema)` to create a new message.
 */
export declare const ProcedureInfoSchema: GenMessage<ProcedureInfo>;
/**
 * @generated from message obiente.cloud.databases.v1.ParameterInfo
 */
export type ParameterInfo = Message<"obiente.cloud.databases.v1.ParameterInfo"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string data_type = 2;
     */
    dataType: string;
    /**
     * "IN", "OUT", "INOUT"
     *
     * @generated from field: string mode = 3;
     */
    mode: string;
    /**
     * @generated from field: int32 ordinal_position = 4;
     */
    ordinalPosition: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.ParameterInfo.
 * Use `create(ParameterInfoSchema)` to create a new message.
 */
export declare const ParameterInfoSchema: GenMessage<ParameterInfo>;
/**
 * Query execution messages
 *
 * @generated from message obiente.cloud.databases.v1.ExecuteQueryRequest
 */
export type ExecuteQueryRequest = Message<"obiente.cloud.databases.v1.ExecuteQueryRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string query = 3;
     */
    query: string;
    /**
     * Specific database name
     *
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * Maximum rows to return (default: 1000)
     *
     * @generated from field: optional int32 max_rows = 5;
     */
    maxRows?: number;
    /**
     * Query timeout in seconds (default: 30)
     *
     * @generated from field: optional int32 timeout_seconds = 6;
     */
    timeoutSeconds?: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.ExecuteQueryRequest.
 * Use `create(ExecuteQueryRequestSchema)` to create a new message.
 */
export declare const ExecuteQueryRequestSchema: GenMessage<ExecuteQueryRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ExecuteQueryResponse
 */
export type ExecuteQueryResponse = Message<"obiente.cloud.databases.v1.ExecuteQueryResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultColumn columns = 1;
     */
    columns: QueryResultColumn[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultRow rows = 2;
     */
    rows: QueryResultRow[];
    /**
     * @generated from field: int32 row_count = 3;
     */
    rowCount: number;
    /**
     * For INSERT/UPDATE/DELETE
     *
     * @generated from field: optional int32 affected_rows = 4;
     */
    affectedRows?: number;
    /**
     * "SELECT", "INSERT", "UPDATE", "DELETE", etc.
     *
     * @generated from field: optional string query_type = 5;
     */
    queryType?: string;
    /**
     * Whether results were truncated due to max_rows
     *
     * @generated from field: bool truncated = 6;
     */
    truncated: boolean;
    /**
     * @generated from field: google.protobuf.Timestamp executed_at = 7;
     */
    executedAt?: Timestamp;
    /**
     * Query execution time in milliseconds
     *
     * @generated from field: int32 execution_time_ms = 8;
     */
    executionTimeMs: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.ExecuteQueryResponse.
 * Use `create(ExecuteQueryResponseSchema)` to create a new message.
 */
export declare const ExecuteQueryResponseSchema: GenMessage<ExecuteQueryResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.QueryResultColumn
 */
export type QueryResultColumn = Message<"obiente.cloud.databases.v1.QueryResultColumn"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: string data_type = 2;
     */
    dataType: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.QueryResultColumn.
 * Use `create(QueryResultColumnSchema)` to create a new message.
 */
export declare const QueryResultColumnSchema: GenMessage<QueryResultColumn>;
/**
 * @generated from message obiente.cloud.databases.v1.QueryResultRow
 */
export type QueryResultRow = Message<"obiente.cloud.databases.v1.QueryResultRow"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultCell cells = 1;
     */
    cells: QueryResultCell[];
};
/**
 * Describes the message obiente.cloud.databases.v1.QueryResultRow.
 * Use `create(QueryResultRowSchema)` to create a new message.
 */
export declare const QueryResultRowSchema: GenMessage<QueryResultRow>;
/**
 * @generated from message obiente.cloud.databases.v1.QueryResultCell
 */
export type QueryResultCell = Message<"obiente.cloud.databases.v1.QueryResultCell"> & {
    /**
     * @generated from field: string column_name = 1;
     */
    columnName: string;
    /**
     * JSON-encoded value (null for NULL values)
     *
     * @generated from field: optional string value = 2;
     */
    value?: string;
    /**
     * @generated from field: bool is_null = 3;
     */
    isNull: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.QueryResultCell.
 * Use `create(QueryResultCellSchema)` to create a new message.
 */
export declare const QueryResultCellSchema: GenMessage<QueryResultCell>;
/**
 * @generated from message obiente.cloud.databases.v1.StreamQueryRequest
 */
export type StreamQueryRequest = Message<"obiente.cloud.databases.v1.StreamQueryRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string query = 3;
     */
    query: string;
    /**
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * @generated from field: optional int32 timeout_seconds = 5;
     */
    timeoutSeconds?: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.StreamQueryRequest.
 * Use `create(StreamQueryRequestSchema)` to create a new message.
 */
export declare const StreamQueryRequestSchema: GenMessage<StreamQueryRequest>;
/**
 * Table data browsing messages
 *
 * @generated from message obiente.cloud.databases.v1.GetTableDataRequest
 */
export type GetTableDataRequest = Message<"obiente.cloud.databases.v1.GetTableDataRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string table_name = 3;
     */
    tableName: string;
    /**
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * @generated from field: int32 page = 5;
     */
    page: number;
    /**
     * default 50
     *
     * @generated from field: int32 per_page = 6;
     */
    perPage: number;
    /**
     * @generated from field: optional string sort_column = 7;
     */
    sortColumn?: string;
    /**
     * "ASC" or "DESC"
     *
     * @generated from field: optional string sort_direction = 8;
     */
    sortDirection?: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ColumnFilter filters = 9;
     */
    filters: ColumnFilter[];
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableDataRequest.
 * Use `create(GetTableDataRequestSchema)` to create a new message.
 */
export declare const GetTableDataRequestSchema: GenMessage<GetTableDataRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ColumnFilter
 */
export type ColumnFilter = Message<"obiente.cloud.databases.v1.ColumnFilter"> & {
    /**
     * @generated from field: string column_name = 1;
     */
    columnName: string;
    /**
     * "=", "!=", "LIKE", "IS NULL", "IS NOT NULL", ">", "<", etc.
     *
     * @generated from field: string operator = 2;
     */
    operator: string;
    /**
     * @generated from field: optional string value = 3;
     */
    value?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ColumnFilter.
 * Use `create(ColumnFilterSchema)` to create a new message.
 */
export declare const ColumnFilterSchema: GenMessage<ColumnFilter>;
/**
 * @generated from message obiente.cloud.databases.v1.GetTableDataResponse
 */
export type GetTableDataResponse = Message<"obiente.cloud.databases.v1.GetTableDataResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultColumn columns = 1;
     */
    columns: QueryResultColumn[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultRow rows = 2;
     */
    rows: QueryResultRow[];
    /**
     * @generated from field: int32 total_rows = 3;
     */
    totalRows: number;
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 4;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableDataResponse.
 * Use `create(GetTableDataResponseSchema)` to create a new message.
 */
export declare const GetTableDataResponseSchema: GenMessage<GetTableDataResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.UpdateTableRowRequest
 */
export type UpdateTableRowRequest = Message<"obiente.cloud.databases.v1.UpdateTableRowRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string table_name = 3;
     */
    tableName: string;
    /**
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * PK columns to identify the row
     *
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultCell where_cells = 5;
     */
    whereCells: QueryResultCell[];
    /**
     * Columns to update
     *
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultCell set_cells = 6;
     */
    setCells: QueryResultCell[];
};
/**
 * Describes the message obiente.cloud.databases.v1.UpdateTableRowRequest.
 * Use `create(UpdateTableRowRequestSchema)` to create a new message.
 */
export declare const UpdateTableRowRequestSchema: GenMessage<UpdateTableRowRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.UpdateTableRowResponse
 */
export type UpdateTableRowResponse = Message<"obiente.cloud.databases.v1.UpdateTableRowResponse"> & {
    /**
     * @generated from field: int32 affected_rows = 1;
     */
    affectedRows: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.UpdateTableRowResponse.
 * Use `create(UpdateTableRowResponseSchema)` to create a new message.
 */
export declare const UpdateTableRowResponseSchema: GenMessage<UpdateTableRowResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.InsertTableRowRequest
 */
export type InsertTableRowRequest = Message<"obiente.cloud.databases.v1.InsertTableRowRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string table_name = 3;
     */
    tableName: string;
    /**
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultCell cells = 5;
     */
    cells: QueryResultCell[];
};
/**
 * Describes the message obiente.cloud.databases.v1.InsertTableRowRequest.
 * Use `create(InsertTableRowRequestSchema)` to create a new message.
 */
export declare const InsertTableRowRequestSchema: GenMessage<InsertTableRowRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.InsertTableRowResponse
 */
export type InsertTableRowResponse = Message<"obiente.cloud.databases.v1.InsertTableRowResponse"> & {
    /**
     * @generated from field: int32 affected_rows = 1;
     */
    affectedRows: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.InsertTableRowResponse.
 * Use `create(InsertTableRowResponseSchema)` to create a new message.
 */
export declare const InsertTableRowResponseSchema: GenMessage<InsertTableRowResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteTableRowsRequest
 */
export type DeleteTableRowsRequest = Message<"obiente.cloud.databases.v1.DeleteTableRowsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string table_name = 3;
     */
    tableName: string;
    /**
     * @generated from field: optional string database_name = 4;
     */
    databaseName?: string;
    /**
     * PK columns to identify the row
     *
     * @generated from field: repeated obiente.cloud.databases.v1.QueryResultCell where_cells = 5;
     */
    whereCells: QueryResultCell[];
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteTableRowsRequest.
 * Use `create(DeleteTableRowsRequestSchema)` to create a new message.
 */
export declare const DeleteTableRowsRequestSchema: GenMessage<DeleteTableRowsRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteTableRowsResponse
 */
export type DeleteTableRowsResponse = Message<"obiente.cloud.databases.v1.DeleteTableRowsResponse"> & {
    /**
     * @generated from field: int32 affected_rows = 1;
     */
    affectedRows: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteTableRowsResponse.
 * Use `create(DeleteTableRowsResponseSchema)` to create a new message.
 */
export declare const DeleteTableRowsResponseSchema: GenMessage<DeleteTableRowsResponse>;
/**
 * Backup management messages
 *
 * @generated from message obiente.cloud.databases.v1.ListBackupsRequest
 */
export type ListBackupsRequest = Message<"obiente.cloud.databases.v1.ListBackupsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
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
 * Describes the message obiente.cloud.databases.v1.ListBackupsRequest.
 * Use `create(ListBackupsRequestSchema)` to create a new message.
 */
export declare const ListBackupsRequestSchema: GenMessage<ListBackupsRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ListBackupsResponse
 */
export type ListBackupsResponse = Message<"obiente.cloud.databases.v1.ListBackupsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.DatabaseBackup backups = 1;
     */
    backups: DatabaseBackup[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListBackupsResponse.
 * Use `create(ListBackupsResponseSchema)` to create a new message.
 */
export declare const ListBackupsResponseSchema: GenMessage<ListBackupsResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateBackupRequest
 */
export type CreateBackupRequest = Message<"obiente.cloud.databases.v1.CreateBackupRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * Optional backup name (auto-generated if not provided)
     *
     * @generated from field: optional string name = 3;
     */
    name?: string;
    /**
     * @generated from field: optional string description = 4;
     */
    description?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateBackupRequest.
 * Use `create(CreateBackupRequestSchema)` to create a new message.
 */
export declare const CreateBackupRequestSchema: GenMessage<CreateBackupRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateBackupResponse
 */
export type CreateBackupResponse = Message<"obiente.cloud.databases.v1.CreateBackupResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseBackup backup = 1;
     */
    backup?: DatabaseBackup;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateBackupResponse.
 * Use `create(CreateBackupResponseSchema)` to create a new message.
 */
export declare const CreateBackupResponseSchema: GenMessage<CreateBackupResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.GetBackupRequest
 */
export type GetBackupRequest = Message<"obiente.cloud.databases.v1.GetBackupRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string backup_id = 3;
     */
    backupId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetBackupRequest.
 * Use `create(GetBackupRequestSchema)` to create a new message.
 */
export declare const GetBackupRequestSchema: GenMessage<GetBackupRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetBackupResponse
 */
export type GetBackupResponse = Message<"obiente.cloud.databases.v1.GetBackupResponse"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseBackup backup = 1;
     */
    backup?: DatabaseBackup;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetBackupResponse.
 * Use `create(GetBackupResponseSchema)` to create a new message.
 */
export declare const GetBackupResponseSchema: GenMessage<GetBackupResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteBackupRequest
 */
export type DeleteBackupRequest = Message<"obiente.cloud.databases.v1.DeleteBackupRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string backup_id = 3;
     */
    backupId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteBackupRequest.
 * Use `create(DeleteBackupRequestSchema)` to create a new message.
 */
export declare const DeleteBackupRequestSchema: GenMessage<DeleteBackupRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DeleteBackupResponse
 */
export type DeleteBackupResponse = Message<"obiente.cloud.databases.v1.DeleteBackupResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DeleteBackupResponse.
 * Use `create(DeleteBackupResponseSchema)` to create a new message.
 */
export declare const DeleteBackupResponseSchema: GenMessage<DeleteBackupResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.RestoreBackupRequest
 */
export type RestoreBackupRequest = Message<"obiente.cloud.databases.v1.RestoreBackupRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string backup_id = 3;
     */
    backupId: string;
    /**
     * Optional: restore to different database name
     *
     * @generated from field: optional string target_database_name = 4;
     */
    targetDatabaseName?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.RestoreBackupRequest.
 * Use `create(RestoreBackupRequestSchema)` to create a new message.
 */
export declare const RestoreBackupRequestSchema: GenMessage<RestoreBackupRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.RestoreBackupResponse
 */
export type RestoreBackupResponse = Message<"obiente.cloud.databases.v1.RestoreBackupResponse"> & {
    /**
     * Restored database instance
     *
     * @generated from field: obiente.cloud.databases.v1.DatabaseInstance database = 1;
     */
    database?: DatabaseInstance;
    /**
     * @generated from field: string message = 2;
     */
    message: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.RestoreBackupResponse.
 * Use `create(RestoreBackupResponseSchema)` to create a new message.
 */
export declare const RestoreBackupResponseSchema: GenMessage<RestoreBackupResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseBackup
 */
export type DatabaseBackup = Message<"obiente.cloud.databases.v1.DatabaseBackup"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: string name = 3;
     */
    name: string;
    /**
     * @generated from field: optional string description = 4;
     */
    description?: string;
    /**
     * @generated from field: int64 size_bytes = 5;
     */
    sizeBytes: bigint;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseBackupStatus status = 6;
     */
    status: DatabaseBackupStatus;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 7;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp completed_at = 8;
     */
    completedAt?: Timestamp;
    /**
     * @generated from field: optional string error_message = 9;
     */
    errorMessage?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.DatabaseBackup.
 * Use `create(DatabaseBackupSchema)` to create a new message.
 */
export declare const DatabaseBackupSchema: GenMessage<DatabaseBackup>;
/**
 * Metrics messages
 *
 * @generated from message obiente.cloud.databases.v1.GetDatabaseMetricsRequest
 */
export type GetDatabaseMetricsRequest = Message<"obiente.cloud.databases.v1.GetDatabaseMetricsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: google.protobuf.Timestamp start_time = 3;
     */
    startTime?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp end_time = 4;
     */
    endTime?: Timestamp;
    /**
     * Aggregation interval (e.g., "1m", "5m", "1h")
     *
     * @generated from field: optional string interval = 5;
     */
    interval?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseMetricsRequest.
 * Use `create(GetDatabaseMetricsRequestSchema)` to create a new message.
 */
export declare const GetDatabaseMetricsRequestSchema: GenMessage<GetDatabaseMetricsRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseMetricsResponse
 */
export type GetDatabaseMetricsResponse = Message<"obiente.cloud.databases.v1.GetDatabaseMetricsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.DatabaseMetric metrics = 1;
     */
    metrics: DatabaseMetric[];
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseMetricsResponse.
 * Use `create(GetDatabaseMetricsResponseSchema)` to create a new message.
 */
export declare const GetDatabaseMetricsResponseSchema: GenMessage<GetDatabaseMetricsResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.StreamDatabaseMetricsRequest
 */
export type StreamDatabaseMetricsRequest = Message<"obiente.cloud.databases.v1.StreamDatabaseMetricsRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.StreamDatabaseMetricsRequest.
 * Use `create(StreamDatabaseMetricsRequestSchema)` to create a new message.
 */
export declare const StreamDatabaseMetricsRequestSchema: GenMessage<StreamDatabaseMetricsRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseMetric
 */
export type DatabaseMetric = Message<"obiente.cloud.databases.v1.DatabaseMetric"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: google.protobuf.Timestamp timestamp = 2;
     */
    timestamp?: Timestamp;
    /**
     * @generated from field: double cpu_usage_percent = 3;
     */
    cpuUsagePercent: number;
    /**
     * @generated from field: int64 memory_used_bytes = 4;
     */
    memoryUsedBytes: bigint;
    /**
     * @generated from field: int64 memory_total_bytes = 5;
     */
    memoryTotalBytes: bigint;
    /**
     * @generated from field: int64 disk_used_bytes = 6;
     */
    diskUsedBytes: bigint;
    /**
     * @generated from field: int64 disk_total_bytes = 7;
     */
    diskTotalBytes: bigint;
    /**
     * @generated from field: int64 connections_active = 8;
     */
    connectionsActive: bigint;
    /**
     * @generated from field: int64 connections_max = 9;
     */
    connectionsMax: bigint;
    /**
     * @generated from field: int64 queries_per_second = 10;
     */
    queriesPerSecond: bigint;
    /**
     * @generated from field: int64 slow_queries = 11;
     */
    slowQueries: bigint;
    /**
     * Percentage (0-100)
     *
     * @generated from field: int64 cache_hit_rate = 12;
     */
    cacheHitRate: bigint;
};
/**
 * Describes the message obiente.cloud.databases.v1.DatabaseMetric.
 * Use `create(DatabaseMetricSchema)` to create a new message.
 */
export declare const DatabaseMetricSchema: GenMessage<DatabaseMetric>;
/**
 * @generated from message obiente.cloud.databases.v1.ListDatabaseSizesRequest
 */
export type ListDatabaseSizesRequest = Message<"obiente.cloud.databases.v1.ListDatabaseSizesRequest"> & {
    /**
     * Optional: filter by database type
     *
     * @generated from field: optional obiente.cloud.databases.v1.DatabaseType type = 1;
     */
    type?: DatabaseType;
};
/**
 * Describes the message obiente.cloud.databases.v1.ListDatabaseSizesRequest.
 * Use `create(ListDatabaseSizesRequestSchema)` to create a new message.
 */
export declare const ListDatabaseSizesRequestSchema: GenMessage<ListDatabaseSizesRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ListDatabaseSizesResponse
 */
export type ListDatabaseSizesResponse = Message<"obiente.cloud.databases.v1.ListDatabaseSizesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.DatabaseSize sizes = 1;
     */
    sizes: DatabaseSize[];
};
/**
 * Describes the message obiente.cloud.databases.v1.ListDatabaseSizesResponse.
 * Use `create(ListDatabaseSizesResponseSchema)` to create a new message.
 */
export declare const ListDatabaseSizesResponseSchema: GenMessage<ListDatabaseSizesResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseSize
 */
export type DatabaseSize = Message<"obiente.cloud.databases.v1.DatabaseSize"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseType type = 3;
     */
    type: DatabaseType;
    /**
     * @generated from field: int32 cpu_cores = 4;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 5;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int64 disk_bytes = 6;
     */
    diskBytes: bigint;
    /**
     * @generated from field: int64 max_connections = 7;
     */
    maxConnections: bigint;
    /**
     * Pricing (in cents per month)
     *
     * @generated from field: int64 price_cents_per_month = 8;
     */
    priceCentsPerMonth: bigint;
};
/**
 * Describes the message obiente.cloud.databases.v1.DatabaseSize.
 * Use `create(DatabaseSizeSchema)` to create a new message.
 */
export declare const DatabaseSizeSchema: GenMessage<DatabaseSize>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseInstance
 */
export type DatabaseInstance = Message<"obiente.cloud.databases.v1.DatabaseInstance"> & {
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
     * @generated from field: obiente.cloud.databases.v1.DatabaseStatus status = 4;
     */
    status: DatabaseStatus;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseType type = 5;
     */
    type: DatabaseType;
    /**
     * @generated from field: optional string version = 6;
     */
    version?: string;
    /**
     * @generated from field: string size = 7;
     */
    size: string;
    /**
     * Resource specifications
     *
     * @generated from field: int32 cpu_cores = 8;
     */
    cpuCores: number;
    /**
     * @generated from field: int64 memory_bytes = 9;
     */
    memoryBytes: bigint;
    /**
     * @generated from field: int64 disk_bytes = 10;
     */
    diskBytes: bigint;
    /**
     * @generated from field: int64 disk_used_bytes = 11;
     */
    diskUsedBytes: bigint;
    /**
     * @generated from field: int64 max_connections = 12;
     */
    maxConnections: bigint;
    /**
     * Network information
     *
     * @generated from field: optional string host = 13;
     */
    host?: string;
    /**
     * @generated from field: optional int32 port = 14;
     */
    port?: number;
    /**
     * Infrastructure information
     *
     * Internal instance ID
     *
     * @generated from field: optional string instance_id = 15;
     */
    instanceId?: string;
    /**
     * Docker Swarm node ID where database is running
     *
     * @generated from field: optional string node_id = 16;
     */
    nodeId?: string;
    /**
     * Metadata and tags
     *
     * @generated from field: map<string, string> metadata = 17;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * Timestamps
     *
     * @generated from field: google.protobuf.Timestamp created_at = 18;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 19;
     */
    updatedAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp last_started_at = 20;
     */
    lastStartedAt?: Timestamp;
    /**
     * Soft delete
     *
     * @generated from field: optional google.protobuf.Timestamp deleted_at = 21;
     */
    deletedAt?: Timestamp;
    /**
     * Organization and ownership
     *
     * @generated from field: string organization_id = 22;
     */
    organizationId: string;
    /**
     * @generated from field: string created_by = 23;
     */
    createdBy: string;
    /**
     * Current metrics (if available)
     *
     * @generated from field: optional obiente.cloud.databases.v1.DatabaseMetric current_metrics = 24;
     */
    currentMetrics?: DatabaseMetric;
    /**
     * Auto-sleep configuration (0 = disabled)
     *
     * @generated from field: optional int32 auto_sleep_seconds = 25;
     */
    autoSleepSeconds?: number;
};
/**
 * Describes the message obiente.cloud.databases.v1.DatabaseInstance.
 * Use `create(DatabaseInstanceSchema)` to create a new message.
 */
export declare const DatabaseInstanceSchema: GenMessage<DatabaseInstance>;
/**
 * Usage/billing messages
 *
 * @generated from message obiente.cloud.databases.v1.GetDatabaseUsageRequest
 */
export type GetDatabaseUsageRequest = Message<"obiente.cloud.databases.v1.GetDatabaseUsageRequest"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * YYYY-MM format, defaults to current month
     *
     * @generated from field: optional string month = 3;
     */
    month?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseUsageRequest.
 * Use `create(GetDatabaseUsageRequestSchema)` to create a new message.
 */
export declare const GetDatabaseUsageRequestSchema: GenMessage<GetDatabaseUsageRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetDatabaseUsageResponse
 */
export type GetDatabaseUsageResponse = Message<"obiente.cloud.databases.v1.GetDatabaseUsageResponse"> & {
    /**
     * @generated from field: string database_id = 1;
     */
    databaseId: string;
    /**
     * @generated from field: string organization_id = 2;
     */
    organizationId: string;
    /**
     * @generated from field: string month = 3;
     */
    month: string;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseUsageMetrics current = 4;
     */
    current?: DatabaseUsageMetrics;
    /**
     * @generated from field: obiente.cloud.databases.v1.DatabaseUsageMetrics estimated_monthly = 5;
     */
    estimatedMonthly?: DatabaseUsageMetrics;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetDatabaseUsageResponse.
 * Use `create(GetDatabaseUsageResponseSchema)` to create a new message.
 */
export declare const GetDatabaseUsageResponseSchema: GenMessage<GetDatabaseUsageResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DatabaseUsageMetrics
 */
export type DatabaseUsageMetrics = Message<"obiente.cloud.databases.v1.DatabaseUsageMetrics"> & {
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
     * @generated from field: int64 estimated_cost_cents = 7;
     */
    estimatedCostCents: bigint;
    /**
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
 * Describes the message obiente.cloud.databases.v1.DatabaseUsageMetrics.
 * Use `create(DatabaseUsageMetricsSchema)` to create a new message.
 */
export declare const DatabaseUsageMetricsSchema: GenMessage<DatabaseUsageMetrics>;
/**
 * DDL operation messages
 *
 * @generated from message obiente.cloud.databases.v1.CreateTableRequest
 */
export type CreateTableRequest = Message<"obiente.cloud.databases.v1.CreateTableRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ColumnDefinition columns = 5;
     */
    columns: ColumnDefinition[];
    /**
     * @generated from field: optional obiente.cloud.databases.v1.PrimaryKeyDefinition primary_key = 6;
     */
    primaryKey?: PrimaryKeyDefinition;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.ForeignKeyDefinition foreign_keys = 7;
     */
    foreignKeys: ForeignKeyDefinition[];
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.IndexDefinition indexes = 8;
     */
    indexes: IndexDefinition[];
    /**
     * @generated from field: optional string comment = 9;
     */
    comment?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateTableRequest.
 * Use `create(CreateTableRequestSchema)` to create a new message.
 */
export declare const CreateTableRequestSchema: GenMessage<CreateTableRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.ColumnDefinition
 */
export type ColumnDefinition = Message<"obiente.cloud.databases.v1.ColumnDefinition"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * e.g., "varchar(255)", "integer", "timestamp"
     *
     * @generated from field: string data_type = 2;
     */
    dataType: string;
    /**
     * @generated from field: bool is_nullable = 3;
     */
    isNullable: boolean;
    /**
     * @generated from field: optional string default_value = 4;
     */
    defaultValue?: string;
    /**
     * @generated from field: bool is_unique = 5;
     */
    isUnique: boolean;
    /**
     * @generated from field: optional string comment = 6;
     */
    comment?: string;
    /**
     * PostgreSQL: SERIAL, MySQL: AUTO_INCREMENT
     *
     * @generated from field: bool auto_increment = 7;
     */
    autoIncrement: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.ColumnDefinition.
 * Use `create(ColumnDefinitionSchema)` to create a new message.
 */
export declare const ColumnDefinitionSchema: GenMessage<ColumnDefinition>;
/**
 * @generated from message obiente.cloud.databases.v1.PrimaryKeyDefinition
 */
export type PrimaryKeyDefinition = Message<"obiente.cloud.databases.v1.PrimaryKeyDefinition"> & {
    /**
     * @generated from field: repeated string column_names = 1;
     */
    columnNames: string[];
    /**
     * @generated from field: optional string name = 2;
     */
    name?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.PrimaryKeyDefinition.
 * Use `create(PrimaryKeyDefinitionSchema)` to create a new message.
 */
export declare const PrimaryKeyDefinitionSchema: GenMessage<PrimaryKeyDefinition>;
/**
 * @generated from message obiente.cloud.databases.v1.IndexDefinition
 */
export type IndexDefinition = Message<"obiente.cloud.databases.v1.IndexDefinition"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: repeated string column_names = 2;
     */
    columnNames: string[];
    /**
     * @generated from field: bool is_unique = 3;
     */
    isUnique: boolean;
    /**
     * btree, hash, gin, gist
     *
     * @generated from field: optional string type = 4;
     */
    type?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.IndexDefinition.
 * Use `create(IndexDefinitionSchema)` to create a new message.
 */
export declare const IndexDefinitionSchema: GenMessage<IndexDefinition>;
/**
 * @generated from message obiente.cloud.databases.v1.ForeignKeyDefinition
 */
export type ForeignKeyDefinition = Message<"obiente.cloud.databases.v1.ForeignKeyDefinition"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: repeated string from_columns = 2;
     */
    fromColumns: string[];
    /**
     * @generated from field: string to_table = 3;
     */
    toTable: string;
    /**
     * @generated from field: repeated string to_columns = 4;
     */
    toColumns: string[];
    /**
     * CASCADE, RESTRICT, SET NULL, NO ACTION
     *
     * @generated from field: string on_delete = 5;
     */
    onDelete: string;
    /**
     * @generated from field: string on_update = 6;
     */
    onUpdate: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.ForeignKeyDefinition.
 * Use `create(ForeignKeyDefinitionSchema)` to create a new message.
 */
export declare const ForeignKeyDefinitionSchema: GenMessage<ForeignKeyDefinition>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateTableResponse
 */
export type CreateTableResponse = Message<"obiente.cloud.databases.v1.CreateTableResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: obiente.cloud.databases.v1.TableInfo table = 2;
     */
    table?: TableInfo;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateTableResponse.
 * Use `create(CreateTableResponseSchema)` to create a new message.
 */
export declare const CreateTableResponseSchema: GenMessage<CreateTableResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.AlterTableRequest
 */
export type AlterTableRequest = Message<"obiente.cloud.databases.v1.AlterTableRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
    /**
     * @generated from field: repeated obiente.cloud.databases.v1.AlterTableOperation operations = 5;
     */
    operations: AlterTableOperation[];
};
/**
 * Describes the message obiente.cloud.databases.v1.AlterTableRequest.
 * Use `create(AlterTableRequestSchema)` to create a new message.
 */
export declare const AlterTableRequestSchema: GenMessage<AlterTableRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.AlterTableOperation
 */
export type AlterTableOperation = Message<"obiente.cloud.databases.v1.AlterTableOperation"> & {
    /**
     * @generated from oneof obiente.cloud.databases.v1.AlterTableOperation.operation
     */
    operation: {
        /**
         * @generated from field: obiente.cloud.databases.v1.AddColumnOperation add_column = 1;
         */
        value: AddColumnOperation;
        case: "addColumn";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.DropColumnOperation drop_column = 2;
         */
        value: DropColumnOperation;
        case: "dropColumn";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.ModifyColumnOperation modify_column = 3;
         */
        value: ModifyColumnOperation;
        case: "modifyColumn";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.RenameColumnOperation rename_column = 4;
         */
        value: RenameColumnOperation;
        case: "renameColumn";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.AddForeignKeyOperation add_foreign_key = 5;
         */
        value: AddForeignKeyOperation;
        case: "addForeignKey";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.DropForeignKeyOperation drop_foreign_key = 6;
         */
        value: DropForeignKeyOperation;
        case: "dropForeignKey";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.AddUniqueConstraintOperation add_unique = 7;
         */
        value: AddUniqueConstraintOperation;
        case: "addUnique";
    } | {
        /**
         * @generated from field: obiente.cloud.databases.v1.DropConstraintOperation drop_constraint = 8;
         */
        value: DropConstraintOperation;
        case: "dropConstraint";
    } | {
        case: undefined;
        value?: undefined;
    };
};
/**
 * Describes the message obiente.cloud.databases.v1.AlterTableOperation.
 * Use `create(AlterTableOperationSchema)` to create a new message.
 */
export declare const AlterTableOperationSchema: GenMessage<AlterTableOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.AddColumnOperation
 */
export type AddColumnOperation = Message<"obiente.cloud.databases.v1.AddColumnOperation"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.ColumnDefinition column = 1;
     */
    column?: ColumnDefinition;
    /**
     * MySQL only
     *
     * @generated from field: optional string after_column = 2;
     */
    afterColumn?: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.AddColumnOperation.
 * Use `create(AddColumnOperationSchema)` to create a new message.
 */
export declare const AddColumnOperationSchema: GenMessage<AddColumnOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.DropColumnOperation
 */
export type DropColumnOperation = Message<"obiente.cloud.databases.v1.DropColumnOperation"> & {
    /**
     * @generated from field: string column_name = 1;
     */
    columnName: string;
    /**
     * @generated from field: bool cascade = 2;
     */
    cascade: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropColumnOperation.
 * Use `create(DropColumnOperationSchema)` to create a new message.
 */
export declare const DropColumnOperationSchema: GenMessage<DropColumnOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.ModifyColumnOperation
 */
export type ModifyColumnOperation = Message<"obiente.cloud.databases.v1.ModifyColumnOperation"> & {
    /**
     * @generated from field: string column_name = 1;
     */
    columnName: string;
    /**
     * @generated from field: optional string new_data_type = 2;
     */
    newDataType?: string;
    /**
     * @generated from field: optional bool is_nullable = 3;
     */
    isNullable?: boolean;
    /**
     * @generated from field: optional string default_value = 4;
     */
    defaultValue?: string;
    /**
     * @generated from field: bool drop_default = 5;
     */
    dropDefault: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.ModifyColumnOperation.
 * Use `create(ModifyColumnOperationSchema)` to create a new message.
 */
export declare const ModifyColumnOperationSchema: GenMessage<ModifyColumnOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.RenameColumnOperation
 */
export type RenameColumnOperation = Message<"obiente.cloud.databases.v1.RenameColumnOperation"> & {
    /**
     * @generated from field: string old_name = 1;
     */
    oldName: string;
    /**
     * @generated from field: string new_name = 2;
     */
    newName: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.RenameColumnOperation.
 * Use `create(RenameColumnOperationSchema)` to create a new message.
 */
export declare const RenameColumnOperationSchema: GenMessage<RenameColumnOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.AddForeignKeyOperation
 */
export type AddForeignKeyOperation = Message<"obiente.cloud.databases.v1.AddForeignKeyOperation"> & {
    /**
     * @generated from field: obiente.cloud.databases.v1.ForeignKeyDefinition foreign_key = 1;
     */
    foreignKey?: ForeignKeyDefinition;
};
/**
 * Describes the message obiente.cloud.databases.v1.AddForeignKeyOperation.
 * Use `create(AddForeignKeyOperationSchema)` to create a new message.
 */
export declare const AddForeignKeyOperationSchema: GenMessage<AddForeignKeyOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.DropForeignKeyOperation
 */
export type DropForeignKeyOperation = Message<"obiente.cloud.databases.v1.DropForeignKeyOperation"> & {
    /**
     * @generated from field: string constraint_name = 1;
     */
    constraintName: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropForeignKeyOperation.
 * Use `create(DropForeignKeyOperationSchema)` to create a new message.
 */
export declare const DropForeignKeyOperationSchema: GenMessage<DropForeignKeyOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.AddUniqueConstraintOperation
 */
export type AddUniqueConstraintOperation = Message<"obiente.cloud.databases.v1.AddUniqueConstraintOperation"> & {
    /**
     * @generated from field: string name = 1;
     */
    name: string;
    /**
     * @generated from field: repeated string column_names = 2;
     */
    columnNames: string[];
};
/**
 * Describes the message obiente.cloud.databases.v1.AddUniqueConstraintOperation.
 * Use `create(AddUniqueConstraintOperationSchema)` to create a new message.
 */
export declare const AddUniqueConstraintOperationSchema: GenMessage<AddUniqueConstraintOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.DropConstraintOperation
 */
export type DropConstraintOperation = Message<"obiente.cloud.databases.v1.DropConstraintOperation"> & {
    /**
     * @generated from field: string constraint_name = 1;
     */
    constraintName: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropConstraintOperation.
 * Use `create(DropConstraintOperationSchema)` to create a new message.
 */
export declare const DropConstraintOperationSchema: GenMessage<DropConstraintOperation>;
/**
 * @generated from message obiente.cloud.databases.v1.AlterTableResponse
 */
export type AlterTableResponse = Message<"obiente.cloud.databases.v1.AlterTableResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: obiente.cloud.databases.v1.TableInfo table = 2;
     */
    table?: TableInfo;
};
/**
 * Describes the message obiente.cloud.databases.v1.AlterTableResponse.
 * Use `create(AlterTableResponseSchema)` to create a new message.
 */
export declare const AlterTableResponseSchema: GenMessage<AlterTableResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DropTableRequest
 */
export type DropTableRequest = Message<"obiente.cloud.databases.v1.DropTableRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
    /**
     * @generated from field: bool cascade = 5;
     */
    cascade: boolean;
    /**
     * @generated from field: bool if_exists = 6;
     */
    ifExists: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropTableRequest.
 * Use `create(DropTableRequestSchema)` to create a new message.
 */
export declare const DropTableRequestSchema: GenMessage<DropTableRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DropTableResponse
 */
export type DropTableResponse = Message<"obiente.cloud.databases.v1.DropTableResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropTableResponse.
 * Use `create(DropTableResponseSchema)` to create a new message.
 */
export declare const DropTableResponseSchema: GenMessage<DropTableResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.RenameTableRequest
 */
export type RenameTableRequest = Message<"obiente.cloud.databases.v1.RenameTableRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string old_name = 4;
     */
    oldName: string;
    /**
     * @generated from field: string new_name = 5;
     */
    newName: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.RenameTableRequest.
 * Use `create(RenameTableRequestSchema)` to create a new message.
 */
export declare const RenameTableRequestSchema: GenMessage<RenameTableRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.RenameTableResponse
 */
export type RenameTableResponse = Message<"obiente.cloud.databases.v1.RenameTableResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.RenameTableResponse.
 * Use `create(RenameTableResponseSchema)` to create a new message.
 */
export declare const RenameTableResponseSchema: GenMessage<RenameTableResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.TruncateTableRequest
 */
export type TruncateTableRequest = Message<"obiente.cloud.databases.v1.TruncateTableRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
    /**
     * @generated from field: bool cascade = 5;
     */
    cascade: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.TruncateTableRequest.
 * Use `create(TruncateTableRequestSchema)` to create a new message.
 */
export declare const TruncateTableRequestSchema: GenMessage<TruncateTableRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.TruncateTableResponse
 */
export type TruncateTableResponse = Message<"obiente.cloud.databases.v1.TruncateTableResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: int64 rows_deleted = 2;
     */
    rowsDeleted: bigint;
};
/**
 * Describes the message obiente.cloud.databases.v1.TruncateTableResponse.
 * Use `create(TruncateTableResponseSchema)` to create a new message.
 */
export declare const TruncateTableResponseSchema: GenMessage<TruncateTableResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateIndexRequest
 */
export type CreateIndexRequest = Message<"obiente.cloud.databases.v1.CreateIndexRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
    /**
     * @generated from field: obiente.cloud.databases.v1.IndexDefinition index = 5;
     */
    index?: IndexDefinition;
    /**
     * @generated from field: bool if_not_exists = 6;
     */
    ifNotExists: boolean;
    /**
     * PostgreSQL only
     *
     * @generated from field: bool concurrently = 7;
     */
    concurrently: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateIndexRequest.
 * Use `create(CreateIndexRequestSchema)` to create a new message.
 */
export declare const CreateIndexRequestSchema: GenMessage<CreateIndexRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.CreateIndexResponse
 */
export type CreateIndexResponse = Message<"obiente.cloud.databases.v1.CreateIndexResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.CreateIndexResponse.
 * Use `create(CreateIndexResponseSchema)` to create a new message.
 */
export declare const CreateIndexResponseSchema: GenMessage<CreateIndexResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.DropIndexRequest
 */
export type DropIndexRequest = Message<"obiente.cloud.databases.v1.DropIndexRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string index_name = 4;
     */
    indexName: string;
    /**
     * @generated from field: bool cascade = 5;
     */
    cascade: boolean;
    /**
     * @generated from field: bool if_exists = 6;
     */
    ifExists: boolean;
    /**
     * PostgreSQL only
     *
     * @generated from field: bool concurrently = 7;
     */
    concurrently: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropIndexRequest.
 * Use `create(DropIndexRequestSchema)` to create a new message.
 */
export declare const DropIndexRequestSchema: GenMessage<DropIndexRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.DropIndexResponse
 */
export type DropIndexResponse = Message<"obiente.cloud.databases.v1.DropIndexResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.databases.v1.DropIndexResponse.
 * Use `create(DropIndexResponseSchema)` to create a new message.
 */
export declare const DropIndexResponseSchema: GenMessage<DropIndexResponse>;
/**
 * @generated from message obiente.cloud.databases.v1.GetTableDDLRequest
 */
export type GetTableDDLRequest = Message<"obiente.cloud.databases.v1.GetTableDDLRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: string database_id = 2;
     */
    databaseId: string;
    /**
     * @generated from field: optional string database_name = 3;
     */
    databaseName?: string;
    /**
     * @generated from field: string table_name = 4;
     */
    tableName: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableDDLRequest.
 * Use `create(GetTableDDLRequestSchema)` to create a new message.
 */
export declare const GetTableDDLRequestSchema: GenMessage<GetTableDDLRequest>;
/**
 * @generated from message obiente.cloud.databases.v1.GetTableDDLResponse
 */
export type GetTableDDLResponse = Message<"obiente.cloud.databases.v1.GetTableDDLResponse"> & {
    /**
     * @generated from field: string ddl = 1;
     */
    ddl: string;
};
/**
 * Describes the message obiente.cloud.databases.v1.GetTableDDLResponse.
 * Use `create(GetTableDDLResponseSchema)` to create a new message.
 */
export declare const GetTableDDLResponseSchema: GenMessage<GetTableDDLResponse>;
/**
 * DatabaseType represents the type of database engine
 *
 * @generated from enum obiente.cloud.databases.v1.DatabaseType
 */
export declare enum DatabaseType {
    /**
     * @generated from enum value: DATABASE_TYPE_UNSPECIFIED = 0;
     */
    DATABASE_TYPE_UNSPECIFIED = 0,
    /**
     * @generated from enum value: POSTGRESQL = 1;
     */
    POSTGRESQL = 1,
    /**
     * @generated from enum value: MYSQL = 2;
     */
    MYSQL = 2,
    /**
     * @generated from enum value: MONGODB = 3;
     */
    MONGODB = 3,
    /**
     * @generated from enum value: REDIS = 4;
     */
    REDIS = 4,
    /**
     * @generated from enum value: MARIADB = 5;
     */
    MARIADB = 5
}
/**
 * Describes the enum obiente.cloud.databases.v1.DatabaseType.
 */
export declare const DatabaseTypeSchema: GenEnum<DatabaseType>;
/**
 * DatabaseStatus represents the current status of a database instance
 *
 * @generated from enum obiente.cloud.databases.v1.DatabaseStatus
 */
export declare enum DatabaseStatus {
    /**
     * @generated from enum value: DATABASE_STATUS_UNSPECIFIED = 0;
     */
    DATABASE_STATUS_UNSPECIFIED = 0,
    /**
     * Database is being provisioned
     *
     * @generated from enum value: CREATING = 1;
     */
    CREATING = 1,
    /**
     * Database is starting up
     *
     * @generated from enum value: STARTING = 2;
     */
    STARTING = 2,
    /**
     * Database is running
     *
     * @generated from enum value: RUNNING = 3;
     */
    RUNNING = 3,
    /**
     * Database is stopping
     *
     * @generated from enum value: STOPPING = 4;
     */
    STOPPING = 4,
    /**
     * Database is stopped
     *
     * @generated from enum value: STOPPED = 5;
     */
    STOPPED = 5,
    /**
     * Database is being backed up
     *
     * @generated from enum value: BACKING_UP = 6;
     */
    BACKING_UP = 6,
    /**
     * Database is being restored
     *
     * @generated from enum value: RESTORING = 7;
     */
    RESTORING = 7,
    /**
     * Database provisioning or operation failed
     *
     * @generated from enum value: FAILED = 8;
     */
    FAILED = 8,
    /**
     * Database is being deleted
     *
     * @generated from enum value: DELETING = 9;
     */
    DELETING = 9,
    /**
     * Database has been deleted (soft delete)
     *
     * @generated from enum value: DELETED = 10;
     */
    DELETED = 10,
    /**
     * Database is suspended
     *
     * @generated from enum value: SUSPENDED = 11;
     */
    SUSPENDED = 11,
    /**
     * Database is sleeping (auto-wakes on connection)
     *
     * @generated from enum value: SLEEPING = 12;
     */
    SLEEPING = 12
}
/**
 * Describes the enum obiente.cloud.databases.v1.DatabaseStatus.
 */
export declare const DatabaseStatusSchema: GenEnum<DatabaseStatus>;
/**
 * @generated from enum obiente.cloud.databases.v1.DatabaseBackupStatus
 */
export declare enum DatabaseBackupStatus {
    /**
     * @generated from enum value: DATABASE_BACKUP_STATUS_UNSPECIFIED = 0;
     */
    DATABASE_BACKUP_STATUS_UNSPECIFIED = 0,
    /**
     * @generated from enum value: BACKUP_CREATING = 1;
     */
    BACKUP_CREATING = 1,
    /**
     * @generated from enum value: BACKUP_COMPLETED = 2;
     */
    BACKUP_COMPLETED = 2,
    /**
     * @generated from enum value: BACKUP_FAILED = 3;
     */
    BACKUP_FAILED = 3,
    /**
     * @generated from enum value: BACKUP_DELETING = 4;
     */
    BACKUP_DELETING = 4,
    /**
     * @generated from enum value: BACKUP_DELETED = 5;
     */
    BACKUP_DELETED = 5
}
/**
 * Describes the enum obiente.cloud.databases.v1.DatabaseBackupStatus.
 */
export declare const DatabaseBackupStatusSchema: GenEnum<DatabaseBackupStatus>;
/**
 * @generated from service obiente.cloud.databases.v1.DatabaseService
 */
export declare const DatabaseService: GenService<{
    /**
     * List organization databases
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ListDatabases
     */
    listDatabases: {
        methodKind: "unary";
        input: typeof ListDatabasesRequestSchema;
        output: typeof ListDatabasesResponseSchema;
    };
    /**
     * Create new database instance
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.CreateDatabase
     */
    createDatabase: {
        methodKind: "unary";
        input: typeof CreateDatabaseRequestSchema;
        output: typeof CreateDatabaseResponseSchema;
    };
    /**
     * Get database instance details
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetDatabase
     */
    getDatabase: {
        methodKind: "unary";
        input: typeof GetDatabaseRequestSchema;
        output: typeof GetDatabaseResponseSchema;
    };
    /**
     * Update database instance configuration
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.UpdateDatabase
     */
    updateDatabase: {
        methodKind: "unary";
        input: typeof UpdateDatabaseRequestSchema;
        output: typeof UpdateDatabaseResponseSchema;
    };
    /**
     * Delete database instance
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.DeleteDatabase
     */
    deleteDatabase: {
        methodKind: "unary";
        input: typeof DeleteDatabaseRequestSchema;
        output: typeof DeleteDatabaseResponseSchema;
    };
    /**
     * Start a stopped database instance
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.StartDatabase
     */
    startDatabase: {
        methodKind: "unary";
        input: typeof StartDatabaseRequestSchema;
        output: typeof StartDatabaseResponseSchema;
    };
    /**
     * Stop a running database instance
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.StopDatabase
     */
    stopDatabase: {
        methodKind: "unary";
        input: typeof StopDatabaseRequestSchema;
        output: typeof StopDatabaseResponseSchema;
    };
    /**
     * Put a database to sleep (auto-wakes on connection)
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.SleepDatabase
     */
    sleepDatabase: {
        methodKind: "unary";
        input: typeof SleepDatabaseRequestSchema;
        output: typeof SleepDatabaseResponseSchema;
    };
    /**
     * Restart a database instance
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.RestartDatabase
     */
    restartDatabase: {
        methodKind: "unary";
        input: typeof RestartDatabaseRequestSchema;
        output: typeof RestartDatabaseResponseSchema;
    };
    /**
     * Stream database status updates
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.StreamDatabaseStatus
     */
    streamDatabaseStatus: {
        methodKind: "server_streaming";
        input: typeof StreamDatabaseStatusRequestSchema;
        output: typeof DatabaseStatusUpdateSchema;
    };
    /**
     * Get database connection info (credentials, connection strings)
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetDatabaseConnectionInfo
     */
    getDatabaseConnectionInfo: {
        methodKind: "unary";
        input: typeof GetDatabaseConnectionInfoRequestSchema;
        output: typeof GetDatabaseConnectionInfoResponseSchema;
    };
    /**
     * Reset database password
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ResetDatabasePassword
     */
    resetDatabasePassword: {
        methodKind: "unary";
        input: typeof ResetDatabasePasswordRequestSchema;
        output: typeof ResetDatabasePasswordResponseSchema;
    };
    /**
     * Database introspection - get schema information
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetDatabaseSchema
     */
    getDatabaseSchema: {
        methodKind: "unary";
        input: typeof GetDatabaseSchemaRequestSchema;
        output: typeof GetDatabaseSchemaResponseSchema;
    };
    /**
     * List tables in a database
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ListTables
     */
    listTables: {
        methodKind: "unary";
        input: typeof ListTablesRequestSchema;
        output: typeof ListTablesResponseSchema;
    };
    /**
     * Get table structure/details
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetTableStructure
     */
    getTableStructure: {
        methodKind: "unary";
        input: typeof GetTableStructureRequestSchema;
        output: typeof GetTableStructureResponseSchema;
    };
    /**
     * Execute query on database
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ExecuteQuery
     */
    executeQuery: {
        methodKind: "unary";
        input: typeof ExecuteQueryRequestSchema;
        output: typeof ExecuteQueryResponseSchema;
    };
    /**
     * Stream query results (for large result sets)
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.StreamQuery
     */
    streamQuery: {
        methodKind: "server_streaming";
        input: typeof StreamQueryRequestSchema;
        output: typeof QueryResultRowSchema;
    };
    /**
     * Table data browsing and editing
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetTableData
     */
    getTableData: {
        methodKind: "unary";
        input: typeof GetTableDataRequestSchema;
        output: typeof GetTableDataResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.UpdateTableRow
     */
    updateTableRow: {
        methodKind: "unary";
        input: typeof UpdateTableRowRequestSchema;
        output: typeof UpdateTableRowResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.InsertTableRow
     */
    insertTableRow: {
        methodKind: "unary";
        input: typeof InsertTableRowRequestSchema;
        output: typeof InsertTableRowResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.DeleteTableRows
     */
    deleteTableRows: {
        methodKind: "unary";
        input: typeof DeleteTableRowsRequestSchema;
        output: typeof DeleteTableRowsResponseSchema;
    };
    /**
     * Backup management
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ListBackups
     */
    listBackups: {
        methodKind: "unary";
        input: typeof ListBackupsRequestSchema;
        output: typeof ListBackupsResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.CreateBackup
     */
    createBackup: {
        methodKind: "unary";
        input: typeof CreateBackupRequestSchema;
        output: typeof CreateBackupResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetBackup
     */
    getBackup: {
        methodKind: "unary";
        input: typeof GetBackupRequestSchema;
        output: typeof GetBackupResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.DeleteBackup
     */
    deleteBackup: {
        methodKind: "unary";
        input: typeof DeleteBackupRequestSchema;
        output: typeof DeleteBackupResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.RestoreBackup
     */
    restoreBackup: {
        methodKind: "unary";
        input: typeof RestoreBackupRequestSchema;
        output: typeof RestoreBackupResponseSchema;
    };
    /**
     * Get database metrics
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetDatabaseMetrics
     */
    getDatabaseMetrics: {
        methodKind: "unary";
        input: typeof GetDatabaseMetricsRequestSchema;
        output: typeof GetDatabaseMetricsResponseSchema;
    };
    /**
     * Stream real-time database metrics
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.StreamDatabaseMetrics
     */
    streamDatabaseMetrics: {
        methodKind: "server_streaming";
        input: typeof StreamDatabaseMetricsRequestSchema;
        output: typeof DatabaseMetricSchema;
    };
    /**
     * Get available database sizes/pricing
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.ListDatabaseSizes
     */
    listDatabaseSizes: {
        methodKind: "unary";
        input: typeof ListDatabaseSizesRequestSchema;
        output: typeof ListDatabaseSizesResponseSchema;
    };
    /**
     * Get database usage and billing information
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetDatabaseUsage
     */
    getDatabaseUsage: {
        methodKind: "unary";
        input: typeof GetDatabaseUsageRequestSchema;
        output: typeof GetDatabaseUsageResponseSchema;
    };
    /**
     * Schema DDL operations
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.CreateTable
     */
    createTable: {
        methodKind: "unary";
        input: typeof CreateTableRequestSchema;
        output: typeof CreateTableResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.AlterTable
     */
    alterTable: {
        methodKind: "unary";
        input: typeof AlterTableRequestSchema;
        output: typeof AlterTableResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.DropTable
     */
    dropTable: {
        methodKind: "unary";
        input: typeof DropTableRequestSchema;
        output: typeof DropTableResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.RenameTable
     */
    renameTable: {
        methodKind: "unary";
        input: typeof RenameTableRequestSchema;
        output: typeof RenameTableResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.TruncateTable
     */
    truncateTable: {
        methodKind: "unary";
        input: typeof TruncateTableRequestSchema;
        output: typeof TruncateTableResponseSchema;
    };
    /**
     * Index management
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.CreateIndex
     */
    createIndex: {
        methodKind: "unary";
        input: typeof CreateIndexRequestSchema;
        output: typeof CreateIndexResponseSchema;
    };
    /**
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.DropIndex
     */
    dropIndex: {
        methodKind: "unary";
        input: typeof DropIndexRequestSchema;
        output: typeof DropIndexResponseSchema;
    };
    /**
     * Get DDL statement for a table
     *
     * @generated from rpc obiente.cloud.databases.v1.DatabaseService.GetTableDDL
     */
    getTableDDL: {
        methodKind: "unary";
        input: typeof GetTableDDLRequestSchema;
        output: typeof GetTableDDLResponseSchema;
    };
}>;
