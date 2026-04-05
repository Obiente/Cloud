import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/support/v1/support_service.proto.
 */
export declare const file_obiente_cloud_support_v1_support_service: GenFile;
/**
 * Request/Response messages
 *
 * @generated from message obiente.cloud.support.v1.CreateTicketRequest
 */
export type CreateTicketRequest = Message<"obiente.cloud.support.v1.CreateTicketRequest"> & {
    /**
     * @generated from field: string subject = 1;
     */
    subject: string;
    /**
     * @generated from field: string description = 2;
     */
    description: string;
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicketCategory category = 3;
     */
    category: SupportTicketCategory;
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicketPriority priority = 4;
     */
    priority: SupportTicketPriority;
    /**
     * Optional organization context
     *
     * @generated from field: optional string organization_id = 5;
     */
    organizationId?: string;
};
/**
 * Describes the message obiente.cloud.support.v1.CreateTicketRequest.
 * Use `create(CreateTicketRequestSchema)` to create a new message.
 */
export declare const CreateTicketRequestSchema: GenMessage<CreateTicketRequest>;
/**
 * @generated from message obiente.cloud.support.v1.CreateTicketResponse
 */
export type CreateTicketResponse = Message<"obiente.cloud.support.v1.CreateTicketResponse"> & {
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicket ticket = 1;
     */
    ticket?: SupportTicket;
};
/**
 * Describes the message obiente.cloud.support.v1.CreateTicketResponse.
 * Use `create(CreateTicketResponseSchema)` to create a new message.
 */
export declare const CreateTicketResponseSchema: GenMessage<CreateTicketResponse>;
/**
 * @generated from message obiente.cloud.support.v1.ListTicketsRequest
 */
export type ListTicketsRequest = Message<"obiente.cloud.support.v1.ListTicketsRequest"> & {
    /**
     * @generated from field: optional obiente.cloud.support.v1.SupportTicketStatus status = 1;
     */
    status?: SupportTicketStatus;
    /**
     * @generated from field: optional obiente.cloud.support.v1.SupportTicketCategory category = 2;
     */
    category?: SupportTicketCategory;
    /**
     * @generated from field: optional obiente.cloud.support.v1.SupportTicketPriority priority = 3;
     */
    priority?: SupportTicketPriority;
    /**
     * @generated from field: optional string organization_id = 4;
     */
    organizationId?: string;
    /**
     * @generated from field: optional int32 page_size = 5;
     */
    pageSize?: number;
    /**
     * @generated from field: optional string page_token = 6;
     */
    pageToken?: string;
};
/**
 * Describes the message obiente.cloud.support.v1.ListTicketsRequest.
 * Use `create(ListTicketsRequestSchema)` to create a new message.
 */
export declare const ListTicketsRequestSchema: GenMessage<ListTicketsRequest>;
/**
 * @generated from message obiente.cloud.support.v1.ListTicketsResponse
 */
export type ListTicketsResponse = Message<"obiente.cloud.support.v1.ListTicketsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.support.v1.SupportTicket tickets = 1;
     */
    tickets: SupportTicket[];
    /**
     * @generated from field: optional string next_page_token = 2;
     */
    nextPageToken?: string;
};
/**
 * Describes the message obiente.cloud.support.v1.ListTicketsResponse.
 * Use `create(ListTicketsResponseSchema)` to create a new message.
 */
export declare const ListTicketsResponseSchema: GenMessage<ListTicketsResponse>;
/**
 * @generated from message obiente.cloud.support.v1.GetTicketRequest
 */
export type GetTicketRequest = Message<"obiente.cloud.support.v1.GetTicketRequest"> & {
    /**
     * @generated from field: string ticket_id = 1;
     */
    ticketId: string;
};
/**
 * Describes the message obiente.cloud.support.v1.GetTicketRequest.
 * Use `create(GetTicketRequestSchema)` to create a new message.
 */
export declare const GetTicketRequestSchema: GenMessage<GetTicketRequest>;
/**
 * @generated from message obiente.cloud.support.v1.GetTicketResponse
 */
export type GetTicketResponse = Message<"obiente.cloud.support.v1.GetTicketResponse"> & {
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicket ticket = 1;
     */
    ticket?: SupportTicket;
};
/**
 * Describes the message obiente.cloud.support.v1.GetTicketResponse.
 * Use `create(GetTicketResponseSchema)` to create a new message.
 */
export declare const GetTicketResponseSchema: GenMessage<GetTicketResponse>;
/**
 * @generated from message obiente.cloud.support.v1.UpdateTicketRequest
 */
export type UpdateTicketRequest = Message<"obiente.cloud.support.v1.UpdateTicketRequest"> & {
    /**
     * @generated from field: string ticket_id = 1;
     */
    ticketId: string;
    /**
     * @generated from field: optional obiente.cloud.support.v1.SupportTicketStatus status = 2;
     */
    status?: SupportTicketStatus;
    /**
     * @generated from field: optional obiente.cloud.support.v1.SupportTicketPriority priority = 3;
     */
    priority?: SupportTicketPriority;
    /**
     * User ID of assignee (superadmin only)
     *
     * @generated from field: optional string assigned_to = 4;
     */
    assignedTo?: string;
};
/**
 * Describes the message obiente.cloud.support.v1.UpdateTicketRequest.
 * Use `create(UpdateTicketRequestSchema)` to create a new message.
 */
export declare const UpdateTicketRequestSchema: GenMessage<UpdateTicketRequest>;
/**
 * @generated from message obiente.cloud.support.v1.UpdateTicketResponse
 */
export type UpdateTicketResponse = Message<"obiente.cloud.support.v1.UpdateTicketResponse"> & {
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicket ticket = 1;
     */
    ticket?: SupportTicket;
};
/**
 * Describes the message obiente.cloud.support.v1.UpdateTicketResponse.
 * Use `create(UpdateTicketResponseSchema)` to create a new message.
 */
export declare const UpdateTicketResponseSchema: GenMessage<UpdateTicketResponse>;
/**
 * @generated from message obiente.cloud.support.v1.AddCommentRequest
 */
export type AddCommentRequest = Message<"obiente.cloud.support.v1.AddCommentRequest"> & {
    /**
     * @generated from field: string ticket_id = 1;
     */
    ticketId: string;
    /**
     * @generated from field: string content = 2;
     */
    content: string;
    /**
     * Internal comment (superadmin only, not visible to user)
     *
     * @generated from field: optional bool internal = 3;
     */
    internal?: boolean;
};
/**
 * Describes the message obiente.cloud.support.v1.AddCommentRequest.
 * Use `create(AddCommentRequestSchema)` to create a new message.
 */
export declare const AddCommentRequestSchema: GenMessage<AddCommentRequest>;
/**
 * @generated from message obiente.cloud.support.v1.AddCommentResponse
 */
export type AddCommentResponse = Message<"obiente.cloud.support.v1.AddCommentResponse"> & {
    /**
     * @generated from field: obiente.cloud.support.v1.TicketComment comment = 1;
     */
    comment?: TicketComment;
};
/**
 * Describes the message obiente.cloud.support.v1.AddCommentResponse.
 * Use `create(AddCommentResponseSchema)` to create a new message.
 */
export declare const AddCommentResponseSchema: GenMessage<AddCommentResponse>;
/**
 * @generated from message obiente.cloud.support.v1.ListCommentsRequest
 */
export type ListCommentsRequest = Message<"obiente.cloud.support.v1.ListCommentsRequest"> & {
    /**
     * @generated from field: string ticket_id = 1;
     */
    ticketId: string;
};
/**
 * Describes the message obiente.cloud.support.v1.ListCommentsRequest.
 * Use `create(ListCommentsRequestSchema)` to create a new message.
 */
export declare const ListCommentsRequestSchema: GenMessage<ListCommentsRequest>;
/**
 * @generated from message obiente.cloud.support.v1.ListCommentsResponse
 */
export type ListCommentsResponse = Message<"obiente.cloud.support.v1.ListCommentsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.support.v1.TicketComment comments = 1;
     */
    comments: TicketComment[];
};
/**
 * Describes the message obiente.cloud.support.v1.ListCommentsResponse.
 * Use `create(ListCommentsResponseSchema)` to create a new message.
 */
export declare const ListCommentsResponseSchema: GenMessage<ListCommentsResponse>;
/**
 * SupportTicket represents a support ticket
 *
 * @generated from message obiente.cloud.support.v1.SupportTicket
 */
export type SupportTicket = Message<"obiente.cloud.support.v1.SupportTicket"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string subject = 2;
     */
    subject: string;
    /**
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicketStatus status = 4;
     */
    status: SupportTicketStatus;
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicketPriority priority = 5;
     */
    priority: SupportTicketPriority;
    /**
     * @generated from field: obiente.cloud.support.v1.SupportTicketCategory category = 6;
     */
    category: SupportTicketCategory;
    /**
     * User ID who created the ticket
     *
     * @generated from field: string created_by = 7;
     */
    createdBy: string;
    /**
     * User ID of assignee (superadmin)
     *
     * @generated from field: optional string assigned_to = 8;
     */
    assignedTo?: string;
    /**
     * @generated from field: optional string organization_id = 9;
     */
    organizationId?: string;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 10;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 11;
     */
    updatedAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp resolved_at = 12;
     */
    resolvedAt?: Timestamp;
    /**
     * Number of comments on this ticket
     *
     * @generated from field: int32 comment_count = 13;
     */
    commentCount: number;
    /**
     * Name of the user who created the ticket
     *
     * @generated from field: optional string created_by_name = 14;
     */
    createdByName?: string;
    /**
     * Email of the user who created the ticket
     *
     * @generated from field: optional string created_by_email = 15;
     */
    createdByEmail?: string;
    /**
     * Name of the user assigned to the ticket
     *
     * @generated from field: optional string assigned_to_name = 16;
     */
    assignedToName?: string;
    /**
     * Email of the user assigned to the ticket
     *
     * @generated from field: optional string assigned_to_email = 17;
     */
    assignedToEmail?: string;
};
/**
 * Describes the message obiente.cloud.support.v1.SupportTicket.
 * Use `create(SupportTicketSchema)` to create a new message.
 */
export declare const SupportTicketSchema: GenMessage<SupportTicket>;
/**
 * TicketComment represents a comment/reply on a support ticket
 *
 * @generated from message obiente.cloud.support.v1.TicketComment
 */
export type TicketComment = Message<"obiente.cloud.support.v1.TicketComment"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string ticket_id = 2;
     */
    ticketId: string;
    /**
     * @generated from field: string content = 3;
     */
    content: string;
    /**
     * User ID who created the comment
     *
     * @generated from field: string created_by = 4;
     */
    createdBy: string;
    /**
     * Internal comment (not visible to user)
     *
     * @generated from field: bool internal = 5;
     */
    internal: boolean;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 6;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 7;
     */
    updatedAt?: Timestamp;
    /**
     * Name of the user who created the comment
     *
     * @generated from field: optional string created_by_name = 8;
     */
    createdByName?: string;
    /**
     * Email of the user who created the comment
     *
     * @generated from field: optional string created_by_email = 9;
     */
    createdByEmail?: string;
    /**
     * Whether the comment author is a superadmin
     *
     * @generated from field: bool is_superadmin = 10;
     */
    isSuperadmin: boolean;
};
/**
 * Describes the message obiente.cloud.support.v1.TicketComment.
 * Use `create(TicketCommentSchema)` to create a new message.
 */
export declare const TicketCommentSchema: GenMessage<TicketComment>;
/**
 * SupportTicketStatus represents the current status of a support ticket
 *
 * @generated from enum obiente.cloud.support.v1.SupportTicketStatus
 */
export declare enum SupportTicketStatus {
    /**
     * @generated from enum value: SUPPORT_TICKET_STATUS_UNSPECIFIED = 0;
     */
    SUPPORT_TICKET_STATUS_UNSPECIFIED = 0,
    /**
     * Ticket is open and awaiting response
     *
     * @generated from enum value: OPEN = 1;
     */
    OPEN = 1,
    /**
     * Ticket is being worked on
     *
     * @generated from enum value: IN_PROGRESS = 2;
     */
    IN_PROGRESS = 2,
    /**
     * Waiting for user response
     *
     * @generated from enum value: WAITING_FOR_USER = 3;
     */
    WAITING_FOR_USER = 3,
    /**
     * Ticket has been resolved
     *
     * @generated from enum value: RESOLVED = 4;
     */
    RESOLVED = 4,
    /**
     * Ticket has been closed
     *
     * @generated from enum value: CLOSED = 5;
     */
    CLOSED = 5
}
/**
 * Describes the enum obiente.cloud.support.v1.SupportTicketStatus.
 */
export declare const SupportTicketStatusSchema: GenEnum<SupportTicketStatus>;
/**
 * SupportTicketPriority represents the priority level of a support ticket
 *
 * @generated from enum obiente.cloud.support.v1.SupportTicketPriority
 */
export declare enum SupportTicketPriority {
    /**
     * @generated from enum value: SUPPORT_TICKET_PRIORITY_UNSPECIFIED = 0;
     */
    SUPPORT_TICKET_PRIORITY_UNSPECIFIED = 0,
    /**
     * @generated from enum value: LOW = 1;
     */
    LOW = 1,
    /**
     * @generated from enum value: MEDIUM = 2;
     */
    MEDIUM = 2,
    /**
     * @generated from enum value: HIGH = 3;
     */
    HIGH = 3,
    /**
     * @generated from enum value: URGENT = 4;
     */
    URGENT = 4
}
/**
 * Describes the enum obiente.cloud.support.v1.SupportTicketPriority.
 */
export declare const SupportTicketPrioritySchema: GenEnum<SupportTicketPriority>;
/**
 * SupportTicketCategory represents the category of a support ticket
 *
 * @generated from enum obiente.cloud.support.v1.SupportTicketCategory
 */
export declare enum SupportTicketCategory {
    /**
     * @generated from enum value: SUPPORT_TICKET_CATEGORY_UNSPECIFIED = 0;
     */
    SUPPORT_TICKET_CATEGORY_UNSPECIFIED = 0,
    /**
     * @generated from enum value: TECHNICAL = 1;
     */
    TECHNICAL = 1,
    /**
     * @generated from enum value: BILLING = 2;
     */
    BILLING = 2,
    /**
     * @generated from enum value: FEATURE_REQUEST = 3;
     */
    FEATURE_REQUEST = 3,
    /**
     * @generated from enum value: BUG_REPORT = 4;
     */
    BUG_REPORT = 4,
    /**
     * @generated from enum value: ACCOUNT = 5;
     */
    ACCOUNT = 5,
    /**
     * @generated from enum value: OTHER = 99;
     */
    OTHER = 99
}
/**
 * Describes the enum obiente.cloud.support.v1.SupportTicketCategory.
 */
export declare const SupportTicketCategorySchema: GenEnum<SupportTicketCategory>;
/**
 * @generated from service obiente.cloud.support.v1.SupportService
 */
export declare const SupportService: GenService<{
    /**
     * Create a new support ticket
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.CreateTicket
     */
    createTicket: {
        methodKind: "unary";
        input: typeof CreateTicketRequestSchema;
        output: typeof CreateTicketResponseSchema;
    };
    /**
     * List support tickets (users see their own, superadmins see all)
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.ListTickets
     */
    listTickets: {
        methodKind: "unary";
        input: typeof ListTicketsRequestSchema;
        output: typeof ListTicketsResponseSchema;
    };
    /**
     * Get a specific ticket by ID
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.GetTicket
     */
    getTicket: {
        methodKind: "unary";
        input: typeof GetTicketRequestSchema;
        output: typeof GetTicketResponseSchema;
    };
    /**
     * Update a ticket (status, priority, etc.)
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.UpdateTicket
     */
    updateTicket: {
        methodKind: "unary";
        input: typeof UpdateTicketRequestSchema;
        output: typeof UpdateTicketResponseSchema;
    };
    /**
     * Add a comment/reply to a ticket
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.AddComment
     */
    addComment: {
        methodKind: "unary";
        input: typeof AddCommentRequestSchema;
        output: typeof AddCommentResponseSchema;
    };
    /**
     * List comments for a ticket
     *
     * @generated from rpc obiente.cloud.support.v1.SupportService.ListComments
     */
    listComments: {
        methodKind: "unary";
        input: typeof ListCommentsRequestSchema;
        output: typeof ListCommentsResponseSchema;
    };
}>;
