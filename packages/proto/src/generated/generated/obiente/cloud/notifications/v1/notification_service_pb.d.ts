import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { Pagination } from "../../common/v1/common_pb";
import type { Message } from "@bufbuild/protobuf";
/**
 * Describes the file obiente/cloud/notifications/v1/notification_service.proto.
 */
export declare const file_obiente_cloud_notifications_v1_notification_service: GenFile;
/**
 * @generated from message obiente.cloud.notifications.v1.ListNotificationsRequest
 */
export type ListNotificationsRequest = Message<"obiente.cloud.notifications.v1.ListNotificationsRequest"> & {
    /**
     * Optional filters
     *
     * @generated from field: optional bool unread_only = 1;
     */
    unreadOnly?: boolean;
    /**
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationType type = 2;
     */
    type?: NotificationType;
    /**
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationSeverity severity = 3;
     */
    severity?: NotificationSeverity;
    /**
     * Pagination
     *
     * @generated from field: int32 page = 4;
     */
    page: number;
    /**
     * @generated from field: int32 per_page = 5;
     */
    perPage: number;
};
/**
 * Describes the message obiente.cloud.notifications.v1.ListNotificationsRequest.
 * Use `create(ListNotificationsRequestSchema)` to create a new message.
 */
export declare const ListNotificationsRequestSchema: GenMessage<ListNotificationsRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.ListNotificationsResponse
 */
export type ListNotificationsResponse = Message<"obiente.cloud.notifications.v1.ListNotificationsResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.Notification notifications = 1;
     */
    notifications: Notification[];
    /**
     * @generated from field: obiente.cloud.common.v1.Pagination pagination = 2;
     */
    pagination?: Pagination;
};
/**
 * Describes the message obiente.cloud.notifications.v1.ListNotificationsResponse.
 * Use `create(ListNotificationsResponseSchema)` to create a new message.
 */
export declare const ListNotificationsResponseSchema: GenMessage<ListNotificationsResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationRequest
 */
export type GetNotificationRequest = Message<"obiente.cloud.notifications.v1.GetNotificationRequest"> & {
    /**
     * @generated from field: string notification_id = 1;
     */
    notificationId: string;
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationRequest.
 * Use `create(GetNotificationRequestSchema)` to create a new message.
 */
export declare const GetNotificationRequestSchema: GenMessage<GetNotificationRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationResponse
 */
export type GetNotificationResponse = Message<"obiente.cloud.notifications.v1.GetNotificationResponse"> & {
    /**
     * @generated from field: obiente.cloud.notifications.v1.Notification notification = 1;
     */
    notification?: Notification;
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationResponse.
 * Use `create(GetNotificationResponseSchema)` to create a new message.
 */
export declare const GetNotificationResponseSchema: GenMessage<GetNotificationResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.MarkAsReadRequest
 */
export type MarkAsReadRequest = Message<"obiente.cloud.notifications.v1.MarkAsReadRequest"> & {
    /**
     * @generated from field: string notification_id = 1;
     */
    notificationId: string;
};
/**
 * Describes the message obiente.cloud.notifications.v1.MarkAsReadRequest.
 * Use `create(MarkAsReadRequestSchema)` to create a new message.
 */
export declare const MarkAsReadRequestSchema: GenMessage<MarkAsReadRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.MarkAsReadResponse
 */
export type MarkAsReadResponse = Message<"obiente.cloud.notifications.v1.MarkAsReadResponse"> & {
    /**
     * @generated from field: obiente.cloud.notifications.v1.Notification notification = 1;
     */
    notification?: Notification;
};
/**
 * Describes the message obiente.cloud.notifications.v1.MarkAsReadResponse.
 * Use `create(MarkAsReadResponseSchema)` to create a new message.
 */
export declare const MarkAsReadResponseSchema: GenMessage<MarkAsReadResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.MarkAllAsReadRequest
 */
export type MarkAllAsReadRequest = Message<"obiente.cloud.notifications.v1.MarkAllAsReadRequest"> & {
    /**
     * Optional filters
     *
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationType type = 1;
     */
    type?: NotificationType;
    /**
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationSeverity severity = 2;
     */
    severity?: NotificationSeverity;
};
/**
 * Describes the message obiente.cloud.notifications.v1.MarkAllAsReadRequest.
 * Use `create(MarkAllAsReadRequestSchema)` to create a new message.
 */
export declare const MarkAllAsReadRequestSchema: GenMessage<MarkAllAsReadRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.MarkAllAsReadResponse
 */
export type MarkAllAsReadResponse = Message<"obiente.cloud.notifications.v1.MarkAllAsReadResponse"> & {
    /**
     * @generated from field: int32 marked_count = 1;
     */
    markedCount: number;
};
/**
 * Describes the message obiente.cloud.notifications.v1.MarkAllAsReadResponse.
 * Use `create(MarkAllAsReadResponseSchema)` to create a new message.
 */
export declare const MarkAllAsReadResponseSchema: GenMessage<MarkAllAsReadResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.DeleteNotificationRequest
 */
export type DeleteNotificationRequest = Message<"obiente.cloud.notifications.v1.DeleteNotificationRequest"> & {
    /**
     * @generated from field: string notification_id = 1;
     */
    notificationId: string;
};
/**
 * Describes the message obiente.cloud.notifications.v1.DeleteNotificationRequest.
 * Use `create(DeleteNotificationRequestSchema)` to create a new message.
 */
export declare const DeleteNotificationRequestSchema: GenMessage<DeleteNotificationRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.DeleteNotificationResponse
 */
export type DeleteNotificationResponse = Message<"obiente.cloud.notifications.v1.DeleteNotificationResponse"> & {
    /**
     * @generated from field: bool success = 1;
     */
    success: boolean;
};
/**
 * Describes the message obiente.cloud.notifications.v1.DeleteNotificationResponse.
 * Use `create(DeleteNotificationResponseSchema)` to create a new message.
 */
export declare const DeleteNotificationResponseSchema: GenMessage<DeleteNotificationResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.DeleteAllNotificationsRequest
 */
export type DeleteAllNotificationsRequest = Message<"obiente.cloud.notifications.v1.DeleteAllNotificationsRequest"> & {
    /**
     * Optional filters
     *
     * @generated from field: optional bool read_only = 1;
     */
    readOnly?: boolean;
    /**
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationType type = 2;
     */
    type?: NotificationType;
};
/**
 * Describes the message obiente.cloud.notifications.v1.DeleteAllNotificationsRequest.
 * Use `create(DeleteAllNotificationsRequestSchema)` to create a new message.
 */
export declare const DeleteAllNotificationsRequestSchema: GenMessage<DeleteAllNotificationsRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.DeleteAllNotificationsResponse
 */
export type DeleteAllNotificationsResponse = Message<"obiente.cloud.notifications.v1.DeleteAllNotificationsResponse"> & {
    /**
     * @generated from field: int32 deleted_count = 1;
     */
    deletedCount: number;
};
/**
 * Describes the message obiente.cloud.notifications.v1.DeleteAllNotificationsResponse.
 * Use `create(DeleteAllNotificationsResponseSchema)` to create a new message.
 */
export declare const DeleteAllNotificationsResponseSchema: GenMessage<DeleteAllNotificationsResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetUnreadCountRequest
 */
export type GetUnreadCountRequest = Message<"obiente.cloud.notifications.v1.GetUnreadCountRequest"> & {
    /**
     * Optional filters
     *
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationType type = 1;
     */
    type?: NotificationType;
    /**
     * @generated from field: optional obiente.cloud.notifications.v1.NotificationSeverity min_severity = 2;
     */
    minSeverity?: NotificationSeverity;
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetUnreadCountRequest.
 * Use `create(GetUnreadCountRequestSchema)` to create a new message.
 */
export declare const GetUnreadCountRequestSchema: GenMessage<GetUnreadCountRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetUnreadCountResponse
 */
export type GetUnreadCountResponse = Message<"obiente.cloud.notifications.v1.GetUnreadCountResponse"> & {
    /**
     * @generated from field: int32 count = 1;
     */
    count: number;
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetUnreadCountResponse.
 * Use `create(GetUnreadCountResponseSchema)` to create a new message.
 */
export declare const GetUnreadCountResponseSchema: GenMessage<GetUnreadCountResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.CreateNotificationRequest
 */
export type CreateNotificationRequest = Message<"obiente.cloud.notifications.v1.CreateNotificationRequest"> & {
    /**
     * @generated from field: string user_id = 1;
     */
    userId: string;
    /**
     * @generated from field: optional string organization_id = 2;
     */
    organizationId?: string;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationType type = 3;
     */
    type: NotificationType;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationSeverity severity = 4;
     */
    severity: NotificationSeverity;
    /**
     * @generated from field: string title = 5;
     */
    title: string;
    /**
     * @generated from field: string message = 6;
     */
    message: string;
    /**
     * @generated from field: optional string action_url = 7;
     */
    actionUrl?: string;
    /**
     * @generated from field: optional string action_label = 8;
     */
    actionLabel?: string;
    /**
     * @generated from field: map<string, string> metadata = 9;
     */
    metadata: {
        [key: string]: string;
    };
};
/**
 * Describes the message obiente.cloud.notifications.v1.CreateNotificationRequest.
 * Use `create(CreateNotificationRequestSchema)` to create a new message.
 */
export declare const CreateNotificationRequestSchema: GenMessage<CreateNotificationRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.CreateNotificationResponse
 */
export type CreateNotificationResponse = Message<"obiente.cloud.notifications.v1.CreateNotificationResponse"> & {
    /**
     * @generated from field: obiente.cloud.notifications.v1.Notification notification = 1;
     */
    notification?: Notification;
};
/**
 * Describes the message obiente.cloud.notifications.v1.CreateNotificationResponse.
 * Use `create(CreateNotificationResponseSchema)` to create a new message.
 */
export declare const CreateNotificationResponseSchema: GenMessage<CreateNotificationResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.CreateOrganizationNotificationRequest
 */
export type CreateOrganizationNotificationRequest = Message<"obiente.cloud.notifications.v1.CreateOrganizationNotificationRequest"> & {
    /**
     * @generated from field: string organization_id = 1;
     */
    organizationId: string;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationType type = 2;
     */
    type: NotificationType;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationSeverity severity = 3;
     */
    severity: NotificationSeverity;
    /**
     * @generated from field: string title = 4;
     */
    title: string;
    /**
     * @generated from field: string message = 5;
     */
    message: string;
    /**
     * @generated from field: optional string action_url = 6;
     */
    actionUrl?: string;
    /**
     * @generated from field: optional string action_label = 7;
     */
    actionLabel?: string;
    /**
     * @generated from field: map<string, string> metadata = 8;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * Optional: only notify specific roles (empty = all members)
     *
     * @generated from field: repeated string roles = 9;
     */
    roles: string[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.CreateOrganizationNotificationRequest.
 * Use `create(CreateOrganizationNotificationRequestSchema)` to create a new message.
 */
export declare const CreateOrganizationNotificationRequestSchema: GenMessage<CreateOrganizationNotificationRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.CreateOrganizationNotificationResponse
 */
export type CreateOrganizationNotificationResponse = Message<"obiente.cloud.notifications.v1.CreateOrganizationNotificationResponse"> & {
    /**
     * @generated from field: int32 created_count = 1;
     */
    createdCount: number;
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.Notification notifications = 2;
     */
    notifications: Notification[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.CreateOrganizationNotificationResponse.
 * Use `create(CreateOrganizationNotificationResponseSchema)` to create a new message.
 */
export declare const CreateOrganizationNotificationResponseSchema: GenMessage<CreateOrganizationNotificationResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.Notification
 */
export type Notification = Message<"obiente.cloud.notifications.v1.Notification"> & {
    /**
     * @generated from field: string id = 1;
     */
    id: string;
    /**
     * @generated from field: string user_id = 2;
     */
    userId: string;
    /**
     * @generated from field: optional string organization_id = 3;
     */
    organizationId?: string;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationType type = 4;
     */
    type: NotificationType;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationSeverity severity = 5;
     */
    severity: NotificationSeverity;
    /**
     * @generated from field: string title = 6;
     */
    title: string;
    /**
     * @generated from field: string message = 7;
     */
    message: string;
    /**
     * @generated from field: bool read = 8;
     */
    read: boolean;
    /**
     * @generated from field: optional google.protobuf.Timestamp read_at = 9;
     */
    readAt?: Timestamp;
    /**
     * @generated from field: optional string action_url = 10;
     */
    actionUrl?: string;
    /**
     * @generated from field: optional string action_label = 11;
     */
    actionLabel?: string;
    /**
     * @generated from field: map<string, string> metadata = 12;
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * @generated from field: bool client_only = 13;
     */
    clientOnly: boolean;
    /**
     * @generated from field: google.protobuf.Timestamp created_at = 14;
     */
    createdAt?: Timestamp;
    /**
     * @generated from field: google.protobuf.Timestamp updated_at = 15;
     */
    updatedAt?: Timestamp;
};
/**
 * Describes the message obiente.cloud.notifications.v1.Notification.
 * Use `create(NotificationSchema)` to create a new message.
 */
export declare const NotificationSchema: GenMessage<Notification>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationTypesRequest
 */
export type GetNotificationTypesRequest = Message<"obiente.cloud.notifications.v1.GetNotificationTypesRequest"> & {};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationTypesRequest.
 * Use `create(GetNotificationTypesRequestSchema)` to create a new message.
 */
export declare const GetNotificationTypesRequestSchema: GenMessage<GetNotificationTypesRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationTypesResponse
 */
export type GetNotificationTypesResponse = Message<"obiente.cloud.notifications.v1.GetNotificationTypesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.NotificationTypeInfo types = 1;
     */
    types: NotificationTypeInfo[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationTypesResponse.
 * Use `create(GetNotificationTypesResponseSchema)` to create a new message.
 */
export declare const GetNotificationTypesResponseSchema: GenMessage<GetNotificationTypesResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.NotificationTypeInfo
 */
export type NotificationTypeInfo = Message<"obiente.cloud.notifications.v1.NotificationTypeInfo"> & {
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationType type = 1;
     */
    type: NotificationType;
    /**
     * @generated from field: string name = 2;
     */
    name: string;
    /**
     * @generated from field: string description = 3;
     */
    description: string;
    /**
     * @generated from field: bool default_email_enabled = 4;
     */
    defaultEmailEnabled: boolean;
    /**
     * @generated from field: bool default_in_app_enabled = 5;
     */
    defaultInAppEnabled: boolean;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationSeverity default_min_severity = 6;
     */
    defaultMinSeverity: NotificationSeverity;
};
/**
 * Describes the message obiente.cloud.notifications.v1.NotificationTypeInfo.
 * Use `create(NotificationTypeInfoSchema)` to create a new message.
 */
export declare const NotificationTypeInfoSchema: GenMessage<NotificationTypeInfo>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationPreferencesRequest
 */
export type GetNotificationPreferencesRequest = Message<"obiente.cloud.notifications.v1.GetNotificationPreferencesRequest"> & {};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationPreferencesRequest.
 * Use `create(GetNotificationPreferencesRequestSchema)` to create a new message.
 */
export declare const GetNotificationPreferencesRequestSchema: GenMessage<GetNotificationPreferencesRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.GetNotificationPreferencesResponse
 */
export type GetNotificationPreferencesResponse = Message<"obiente.cloud.notifications.v1.GetNotificationPreferencesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.NotificationPreference preferences = 1;
     */
    preferences: NotificationPreference[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.GetNotificationPreferencesResponse.
 * Use `create(GetNotificationPreferencesResponseSchema)` to create a new message.
 */
export declare const GetNotificationPreferencesResponseSchema: GenMessage<GetNotificationPreferencesResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.UpdateNotificationPreferencesRequest
 */
export type UpdateNotificationPreferencesRequest = Message<"obiente.cloud.notifications.v1.UpdateNotificationPreferencesRequest"> & {
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.NotificationPreference preferences = 1;
     */
    preferences: NotificationPreference[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.UpdateNotificationPreferencesRequest.
 * Use `create(UpdateNotificationPreferencesRequestSchema)` to create a new message.
 */
export declare const UpdateNotificationPreferencesRequestSchema: GenMessage<UpdateNotificationPreferencesRequest>;
/**
 * @generated from message obiente.cloud.notifications.v1.UpdateNotificationPreferencesResponse
 */
export type UpdateNotificationPreferencesResponse = Message<"obiente.cloud.notifications.v1.UpdateNotificationPreferencesResponse"> & {
    /**
     * @generated from field: repeated obiente.cloud.notifications.v1.NotificationPreference preferences = 1;
     */
    preferences: NotificationPreference[];
};
/**
 * Describes the message obiente.cloud.notifications.v1.UpdateNotificationPreferencesResponse.
 * Use `create(UpdateNotificationPreferencesResponseSchema)` to create a new message.
 */
export declare const UpdateNotificationPreferencesResponseSchema: GenMessage<UpdateNotificationPreferencesResponse>;
/**
 * @generated from message obiente.cloud.notifications.v1.NotificationPreference
 */
export type NotificationPreference = Message<"obiente.cloud.notifications.v1.NotificationPreference"> & {
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationType notification_type = 1;
     */
    notificationType: NotificationType;
    /**
     * @generated from field: bool email_enabled = 2;
     */
    emailEnabled: boolean;
    /**
     * @generated from field: bool in_app_enabled = 3;
     */
    inAppEnabled: boolean;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationFrequency frequency = 4;
     */
    frequency: NotificationFrequency;
    /**
     * @generated from field: obiente.cloud.notifications.v1.NotificationSeverity min_severity = 5;
     */
    minSeverity: NotificationSeverity;
};
/**
 * Describes the message obiente.cloud.notifications.v1.NotificationPreference.
 * Use `create(NotificationPreferenceSchema)` to create a new message.
 */
export declare const NotificationPreferenceSchema: GenMessage<NotificationPreference>;
/**
 * @generated from enum obiente.cloud.notifications.v1.NotificationType
 */
export declare enum NotificationType {
    /**
     * @generated from enum value: NOTIFICATION_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_INFO = 1;
     */
    INFO = 1,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_SUCCESS = 2;
     */
    SUCCESS = 2,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_WARNING = 3;
     */
    WARNING = 3,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_ERROR = 4;
     */
    ERROR = 4,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_DEPLOYMENT = 5;
     */
    DEPLOYMENT = 5,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_BILLING = 6;
     */
    BILLING = 6,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_QUOTA = 7;
     */
    QUOTA = 7,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_INVITE = 8;
     */
    INVITE = 8,
    /**
     * @generated from enum value: NOTIFICATION_TYPE_SYSTEM = 9;
     */
    SYSTEM = 9
}
/**
 * Describes the enum obiente.cloud.notifications.v1.NotificationType.
 */
export declare const NotificationTypeSchema: GenEnum<NotificationType>;
/**
 * @generated from enum obiente.cloud.notifications.v1.NotificationSeverity
 */
export declare enum NotificationSeverity {
    /**
     * @generated from enum value: NOTIFICATION_SEVERITY_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: NOTIFICATION_SEVERITY_LOW = 1;
     */
    LOW = 1,
    /**
     * @generated from enum value: NOTIFICATION_SEVERITY_MEDIUM = 2;
     */
    MEDIUM = 2,
    /**
     * @generated from enum value: NOTIFICATION_SEVERITY_HIGH = 3;
     */
    HIGH = 3,
    /**
     * @generated from enum value: NOTIFICATION_SEVERITY_CRITICAL = 4;
     */
    CRITICAL = 4
}
/**
 * Describes the enum obiente.cloud.notifications.v1.NotificationSeverity.
 */
export declare const NotificationSeveritySchema: GenEnum<NotificationSeverity>;
/**
 * @generated from enum obiente.cloud.notifications.v1.NotificationFrequency
 */
export declare enum NotificationFrequency {
    /**
     * @generated from enum value: NOTIFICATION_FREQUENCY_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from enum value: NOTIFICATION_FREQUENCY_IMMEDIATE = 1;
     */
    IMMEDIATE = 1,
    /**
     * @generated from enum value: NOTIFICATION_FREQUENCY_DAILY = 2;
     */
    DAILY = 2,
    /**
     * @generated from enum value: NOTIFICATION_FREQUENCY_WEEKLY = 3;
     */
    WEEKLY = 3,
    /**
     * @generated from enum value: NOTIFICATION_FREQUENCY_NEVER = 4;
     */
    NEVER = 4
}
/**
 * Describes the enum obiente.cloud.notifications.v1.NotificationFrequency.
 */
export declare const NotificationFrequencySchema: GenEnum<NotificationFrequency>;
/**
 * @generated from service obiente.cloud.notifications.v1.NotificationService
 */
export declare const NotificationService: GenService<{
    /**
     * List notifications for the current user
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.ListNotifications
     */
    listNotifications: {
        methodKind: "unary";
        input: typeof ListNotificationsRequestSchema;
        output: typeof ListNotificationsResponseSchema;
    };
    /**
     * Get a specific notification
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.GetNotification
     */
    getNotification: {
        methodKind: "unary";
        input: typeof GetNotificationRequestSchema;
        output: typeof GetNotificationResponseSchema;
    };
    /**
     * Mark notification as read
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.MarkAsRead
     */
    markAsRead: {
        methodKind: "unary";
        input: typeof MarkAsReadRequestSchema;
        output: typeof MarkAsReadResponseSchema;
    };
    /**
     * Mark all notifications as read
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.MarkAllAsRead
     */
    markAllAsRead: {
        methodKind: "unary";
        input: typeof MarkAllAsReadRequestSchema;
        output: typeof MarkAllAsReadResponseSchema;
    };
    /**
     * Delete a notification
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.DeleteNotification
     */
    deleteNotification: {
        methodKind: "unary";
        input: typeof DeleteNotificationRequestSchema;
        output: typeof DeleteNotificationResponseSchema;
    };
    /**
     * Delete all notifications
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.DeleteAllNotifications
     */
    deleteAllNotifications: {
        methodKind: "unary";
        input: typeof DeleteAllNotificationsRequestSchema;
        output: typeof DeleteAllNotificationsResponseSchema;
    };
    /**
     * Get unread count
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.GetUnreadCount
     */
    getUnreadCount: {
        methodKind: "unary";
        input: typeof GetUnreadCountRequestSchema;
        output: typeof GetUnreadCountResponseSchema;
    };
    /**
     * Create a notification (internal/admin use)
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.CreateNotification
     */
    createNotification: {
        methodKind: "unary";
        input: typeof CreateNotificationRequestSchema;
        output: typeof CreateNotificationResponseSchema;
    };
    /**
     * Create notifications for organization members
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.CreateOrganizationNotification
     */
    createOrganizationNotification: {
        methodKind: "unary";
        input: typeof CreateOrganizationNotificationRequestSchema;
        output: typeof CreateOrganizationNotificationResponseSchema;
    };
    /**
     * Get available notification types
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.GetNotificationTypes
     */
    getNotificationTypes: {
        methodKind: "unary";
        input: typeof GetNotificationTypesRequestSchema;
        output: typeof GetNotificationTypesResponseSchema;
    };
    /**
     * Get user's notification preferences
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.GetNotificationPreferences
     */
    getNotificationPreferences: {
        methodKind: "unary";
        input: typeof GetNotificationPreferencesRequestSchema;
        output: typeof GetNotificationPreferencesResponseSchema;
    };
    /**
     * Update user's notification preferences
     *
     * @generated from rpc obiente.cloud.notifications.v1.NotificationService.UpdateNotificationPreferences
     */
    updateNotificationPreferences: {
        methodKind: "unary";
        input: typeof UpdateNotificationPreferencesRequestSchema;
        output: typeof UpdateNotificationPreferencesResponseSchema;
    };
}>;
