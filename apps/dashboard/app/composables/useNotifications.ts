import { ref, computed, readonly, onMounted, onUnmounted, watch, toRef } from "vue";
import { NotificationService, type Notification as ProtoNotification } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

export interface Notification {
  id: string;
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
  type?: string;
  severity?: string;
  actionUrl?: string;
  actionLabel?: string;
  metadata?: Record<string, string>;
  clientOnly?: boolean;
}

const notifications = ref<Notification[]>([]);
const isLoading = ref(false);
const lastSyncTime = ref<Date | null>(null);

let notificationIdCounter = 0;
let syncInterval: ReturnType<typeof setInterval> | null = null;

// Convert proto notification to our interface
function protoToNotification(proto: ProtoNotification): Notification {
  let timestamp = new Date();
  if (proto.createdAt) {
    // Convert bigint seconds to number, and handle nanos
    const seconds = typeof proto.createdAt.seconds === 'bigint' 
      ? Number(proto.createdAt.seconds) 
      : proto.createdAt.seconds;
    const nanos = proto.createdAt.nanos || 0;
    timestamp = new Date(seconds * 1000 + nanos / 1000000);
  }
  
  return {
    id: proto.id,
    title: proto.title,
    message: proto.message,
    timestamp,
    read: proto.read,
    type: proto.type?.toString(),
    severity: proto.severity?.toString(),
    actionUrl: proto.actionUrl || undefined,
    actionLabel: proto.actionLabel || undefined,
    metadata: proto.metadata || undefined,
    clientOnly: proto.clientOnly || false,
  };
}

export function useNotifications() {
  const client = useConnectClient(NotificationService);
  const auth = useAuth();
  // Use toRef to preserve the computed ref type from the reactive object
  const isAuthenticated = toRef(auth, 'isAuthenticated');

  // Fetch notifications from server
  const fetchNotifications = async () => {
    if (!import.meta.client || !isAuthenticated.value) {
      return;
    }

    try {
      isLoading.value = true;
      const response = await client.listNotifications({
        page: 1,
        perPage: 100,
      });

      // Merge server notifications with client-only notifications
      const serverNotifications = (response.notifications || []).map(protoToNotification);
      const clientOnlyNotifications = notifications.value.filter((n) => n.clientOnly);

      // Combine and deduplicate by ID
      const allNotifications = [...serverNotifications, ...clientOnlyNotifications];
      const uniqueNotifications = new Map<string, Notification>();
      
      for (const notif of allNotifications) {
        if (!uniqueNotifications.has(notif.id)) {
          uniqueNotifications.set(notif.id, notif);
        }
      }

      notifications.value = Array.from(uniqueNotifications.values())
        .sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime())
        .slice(0, 100); // Keep only the last 100

      lastSyncTime.value = new Date();
    } catch (error) {
      console.error("[Notifications] Failed to fetch notifications:", error);
      // Don't clear existing notifications on error
    } finally {
      isLoading.value = false;
    }
  };

  // Get unread count from server
  const fetchUnreadCount = async (): Promise<number> => {
    if (!import.meta.client || !isAuthenticated.value) {
      return 0;
    }

    try {
      const response = await client.getUnreadCount({});
      return response.count || 0;
    } catch (error) {
      console.error("[Notifications] Failed to fetch unread count:", error);
      return 0;
    }
  };

  // Add a notification (client-side only by default, or server-side if specified)
  const addNotification = async (notification: Omit<Notification, "id" | "timestamp" | "read">, clientOnly: boolean = true) => {
    const id = `notification-${++notificationIdCounter}-${Date.now()}`;
    const newNotification: Notification = {
      ...notification,
      id,
      timestamp: new Date(),
      read: false,
      clientOnly,
    };
    
    notifications.value.unshift(newNotification);
    
    // Keep only the last 100 notifications
    if (notifications.value.length > 100) {
      notifications.value = notifications.value.slice(0, 100);
    }

    // If not client-only, also create on server (for future persistence)
    // Note: This requires proper permissions - typically only system can create server-side notifications
    // For now, we'll just store client-side and let the server sync handle it
  };

  const removeNotification = async (id: string) => {
    const notification = notifications.value.find((n) => n.id === id);
    if (!notification) return;

    // If it's a server notification, delete from server
    if (!notification.clientOnly && import.meta.client && isAuthenticated.value) {
      try {
        await client.deleteNotification({ notificationId: id });
      } catch (error) {
        console.error("[Notifications] Failed to delete notification from server:", error);
        // Continue with local removal anyway
      }
    }

    // Remove from local list
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }
  };

  const markAsRead = async (id: string) => {
    const notification = notifications.value.find((n) => n.id === id);
    if (!notification) return;

    // Optimistically update
    notification.read = true;

    // If it's a server notification, mark as read on server
    if (!notification.clientOnly && import.meta.client && isAuthenticated.value) {
      try {
        await client.markAsRead({ notificationId: id });
      } catch (error) {
        console.error("[Notifications] Failed to mark notification as read on server:", error);
        // Revert optimistic update
        notification.read = false;
      }
    }
  };

  const markAllAsRead = async () => {
    // Optimistically update all
    notifications.value.forEach((n) => {
      n.read = true;
    });

    // Mark all server notifications as read
    if (import.meta.client && isAuthenticated.value) {
      try {
        await client.markAllAsRead({});
        // Refresh to get updated state
        await fetchNotifications();
      } catch (error) {
        console.error("[Notifications] Failed to mark all as read on server:", error);
        // Refresh to revert optimistic updates
        await fetchNotifications();
      }
    }
  };

  const clearAll = async () => {
    // Delete all server notifications
    if (import.meta.client && isAuthenticated.value) {
      try {
        await client.deleteAllNotifications({});
      } catch (error) {
        console.error("[Notifications] Failed to delete all notifications from server:", error);
      }
    }

    // Clear local list (including client-only)
    notifications.value = [];
  };

  const unreadCount = computed(() =>
    notifications.value.filter((n) => !n.read).length
  );

  // Start periodic sync when authenticated
  const startSync = () => {
    if (syncInterval) return; // Already syncing
    
    // Initial fetch
    fetchNotifications();
    
    // Sync every 30 seconds
    syncInterval = setInterval(() => {
      if (isAuthenticated.value) {
        fetchNotifications();
      }
    }, 30000);
  };

  // Stop periodic sync
  const stopSync = () => {
    if (syncInterval) {
      clearInterval(syncInterval);
      syncInterval = null;
    }
  };

  // Auto-start sync when composable is used and user is authenticated
  if (import.meta.client) {
    onMounted(() => {
      if (isAuthenticated.value) {
        startSync();
      }
      
      // Watch for auth changes
      const unwatch = watch(isAuthenticated, (isAuth) => {
        if (isAuth) {
          startSync();
        } else {
          stopSync();
          notifications.value = notifications.value.filter((n) => n.clientOnly); // Keep only client-only
        }
      });
      
      onUnmounted(() => {
        stopSync();
        unwatch();
      });
    });
  }

  return {
    notifications: readonly(notifications),
    isLoading: readonly(isLoading),
    lastSyncTime: readonly(lastSyncTime),
    addNotification,
    removeNotification,
    markAsRead,
    markAllAsRead,
    clearAll,
    unreadCount,
    fetchNotifications,
    fetchUnreadCount,
    startSync,
    stopSync,
  };
}
