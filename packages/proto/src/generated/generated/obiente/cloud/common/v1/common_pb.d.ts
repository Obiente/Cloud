import type { GenEnum, GenFile, GenMessage } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/common/v1/common.proto.
 */
export declare const file_obiente_cloud_common_v1_common: GenFile;
/**
 * Pagination represents pagination information for list responses
 *
 * @generated from message obiente.cloud.common.v1.Pagination
 */
export type Pagination = Message<"obiente.cloud.common.v1.Pagination"> & {
    /**
     * Current page number (1-indexed)
     *
     * @generated from field: int32 page = 1;
     */
    page: number;
    /**
     * Number of items per page
     *
     * @generated from field: int32 per_page = 2;
     */
    perPage: number;
    /**
     * Total number of items
     *
     * @generated from field: int32 total = 3;
     */
    total: number;
    /**
     * Total number of pages
     *
     * @generated from field: int32 total_pages = 4;
     */
    totalPages: number;
};
/**
 * Describes the message obiente.cloud.common.v1.Pagination.
 * Use `create(PaginationSchema)` to create a new message.
 */
export declare const PaginationSchema: GenMessage<Pagination>;
/**
 * VPSSize represents a VPS instance size configuration
 * This is a shared type used across multiple services
 *
 * @generated from message obiente.cloud.common.v1.VPSSize
 */
export type VPSSize = Message<"obiente.cloud.common.v1.VPSSize"> & {
    /**
     * Size ID (e.g., "small", "medium", "cx11")
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
     * Optional description
     *
     * @generated from field: optional string description = 3;
     */
    description?: string;
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
    /**
     * @generated from field: optional google.protobuf.Timestamp created_at = 11;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: optional google.protobuf.Timestamp updated_at = 12;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.common.v1.VPSSize.
 * Use `create(VPSSizeSchema)` to create a new message.
 */
export declare const VPSSizeSchema: GenMessage<VPSSize>;
/**
 * ExtractServerFileRequest is a shared request message for extracting zip files
 * Resource-specific services should embed or reference this in their own request types
 *
 * @generated from message obiente.cloud.common.v1.ExtractServerFileRequest
 */
export type ExtractServerFileRequest = Message<"obiente.cloud.common.v1.ExtractServerFileRequest"> & {
    /**
     * Path to the zip file to extract
     *
     * @generated from field: string zip_path = 1;
     */
    zipPath: string;
    /**
     * Directory path where files should be extracted
     *
     * @generated from field: string destination_path = 2;
     */
    destinationPath: string;
    /**
     * If specified, extract to this volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 3;
     */
    volumeName?: string;
};
/**
 * Describes the message obiente.cloud.common.v1.ExtractServerFileRequest.
 * Use `create(ExtractServerFileRequestSchema)` to create a new message.
 */
export declare const ExtractServerFileRequestSchema: GenMessage<ExtractServerFileRequest>;
/**
 * ExtractServerFileResponse is a shared response message for extracting zip files
 *
 * @generated from message obiente.cloud.common.v1.ExtractServerFileResponse
 */
export type ExtractServerFileResponse = Message<"obiente.cloud.common.v1.ExtractServerFileResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
    /**
     * Number of files successfully extracted
     *
     * @generated from field: int32 files_extracted = 3;
     */
    filesExtracted: number;
};
/**
 * Describes the message obiente.cloud.common.v1.ExtractServerFileResponse.
 * Use `create(ExtractServerFileResponseSchema)` to create a new message.
 */
export declare const ExtractServerFileResponseSchema: GenMessage<ExtractServerFileResponse>;
/**
 * CreateServerFileArchiveRequest is a shared request message for creating zip archives
 * Resource-specific services should embed or reference this in their own request types
 *
 * @generated from message obiente.cloud.common.v1.CreateServerFileArchiveRequest
 */
export type CreateServerFileArchiveRequest = Message<"obiente.cloud.common.v1.CreateServerFileArchiveRequest"> & {
    /**
     * Paths to files/folders to zip
     *
     * @generated from field: repeated string source_paths = 1;
     */
    sourcePaths: string[];
    /**
     * Path where the zip file should be created
     *
     * @generated from field: string destination_path = 2;
     */
    destinationPath: string;
    /**
     * If true, zip includes the parent folder; if false, zip contains files directly
     *
     * @generated from field: bool include_parent_folder = 3;
     */
    includeParentFolder: boolean;
};
/**
 * Describes the message obiente.cloud.common.v1.CreateServerFileArchiveRequest.
 * Use `create(CreateServerFileArchiveRequestSchema)` to create a new message.
 */
export declare const CreateServerFileArchiveRequestSchema: GenMessage<CreateServerFileArchiveRequest>;
/**
 * CreateServerFileArchiveResponse is a shared response message for creating zip archives
 *
 * @generated from message obiente.cloud.common.v1.CreateServerFileArchiveResponse
 */
export type CreateServerFileArchiveResponse = Message<"obiente.cloud.common.v1.CreateServerFileArchiveResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
    /**
     * Path to the created zip file
     *
     * @generated from field: string archive_path = 3;
     */
    archivePath: string;
    /**
     * Number of files archived
     *
     * @generated from field: int32 files_archived = 4;
     */
    filesArchived: number;
};
/**
 * Describes the message obiente.cloud.common.v1.CreateServerFileArchiveResponse.
 * Use `create(CreateServerFileArchiveResponseSchema)` to create a new message.
 */
export declare const CreateServerFileArchiveResponseSchema: GenMessage<CreateServerFileArchiveResponse>;
/**
 * ChunkedUploadPayload standardizes chunked file uploads across resources (gameservers, deployments, etc.)
 * Resource-specific services should wrap this in their own requests alongside resource identifiers.
 *
 * @generated from message obiente.cloud.common.v1.ChunkedUploadPayload
 */
export type ChunkedUploadPayload = Message<"obiente.cloud.common.v1.ChunkedUploadPayload"> & {
    /**
     * Directory path where files should be extracted (default: "/")
     *
     * @generated from field: string destination_path = 1;
     */
    destinationPath: string;
    /**
     * Optional target volume instead of container filesystem
     *
     * @generated from field: optional string volume_name = 2;
     */
    volumeName?: string;
    /**
     * Name of the file being uploaded
     *
     * @generated from field: string file_name = 3;
     */
    fileName: string;
    /**
     * Total size of the complete file (for validation)
     *
     * @generated from field: int64 file_size = 4;
     */
    fileSize: bigint;
    /**
     * 0-based index of this chunk
     *
     * @generated from field: int32 chunk_index = 5;
     */
    chunkIndex: number;
    /**
     * Total number of chunks for this file
     *
     * @generated from field: int32 total_chunks = 6;
     */
    totalChunks: number;
    /**
     * Raw file data for this chunk
     *
     * @generated from field: bytes chunk_data = 7;
     */
    chunkData: Uint8Array;
    /**
     * File permissions (e.g., "0644"), optional
     *
     * @generated from field: optional string file_mode = 8;
     */
    fileMode?: string;
};
/**
 * Describes the message obiente.cloud.common.v1.ChunkedUploadPayload.
 * Use `create(ChunkedUploadPayloadSchema)` to create a new message.
 */
export declare const ChunkedUploadPayloadSchema: GenMessage<ChunkedUploadPayload>;
/**
 * ChunkedUploadResponsePayload represents the result of processing a single chunk.
 *
 * @generated from message obiente.cloud.common.v1.ChunkedUploadResponsePayload
 */
export type ChunkedUploadResponsePayload = Message<"obiente.cloud.common.v1.ChunkedUploadResponsePayload"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
    /**
     * @generated from field: optional string error = 2;
     */
    error?: string;
    /**
     * Name of the uploaded file
     *
     * @generated from field: string file_name = 3;
     */
    fileName: string;
    /**
     * Total bytes received for this file so far
     *
     * @generated from field: int64 bytes_received = 4;
     */
    bytesReceived: bigint;
};
/**
 * Describes the message obiente.cloud.common.v1.ChunkedUploadResponsePayload.
 * Use `create(ChunkedUploadResponsePayloadSchema)` to create a new message.
 */
export declare const ChunkedUploadResponsePayloadSchema: GenMessage<ChunkedUploadResponsePayload>;
/**
 * LogLevel represents the severity/type of a log line
 *
 * @generated from enum obiente.cloud.common.v1.LogLevel
 */
export declare enum LogLevel {
    /**
     * @generated from enum value: LOG_LEVEL_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * Trace-level debugging
     *
     * @generated from enum value: LOG_LEVEL_TRACE = 1;
     */
    TRACE = 1,
    /**
     * Debug information
     *
     * @generated from enum value: LOG_LEVEL_DEBUG = 2;
     */
    DEBUG = 2,
    /**
     * Informational messages (default)
     *
     * @generated from enum value: LOG_LEVEL_INFO = 3;
     */
    INFO = 3,
    /**
     * Warning messages
     *
     * @generated from enum value: LOG_LEVEL_WARN = 4;
     */
    WARN = 4,
    /**
     * Error messages
     *
     * @generated from enum value: LOG_LEVEL_ERROR = 5;
     */
    ERROR = 5
}
/**
 * Describes the enum obiente.cloud.common.v1.LogLevel.
 */
export declare const LogLevelSchema: GenEnum<LogLevel>;
