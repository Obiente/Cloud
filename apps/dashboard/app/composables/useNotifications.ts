import { ref, computed, readonly } from "vue";

export interface Notification {
  id: string;
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
}

const notifications = ref<Notification[]>([]);

let notificationIdCounter = 0;

export function useNotifications() {
  const addNotification = (notification: Omit<Notification, "id" | "timestamp" | "read">) => {
    const id = `notification-${++notificationIdCounter}-${Date.now()}`;
    const newNotification: Notification = {
      ...notification,
      id,
      timestamp: new Date(),
      read: false,
    };
    
    notifications.value.unshift(newNotification);
    
    // Keep only the last 100 notifications
    if (notifications.value.length > 100) {
      notifications.value = notifications.value.slice(0, 100);
    }
  };

  const removeNotification = (id: string) => {
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index !== -1) {
      notifications.value.splice(index, 1);
    }
  };

  const markAsRead = (id: string) => {
    const notification = notifications.value.find((n) => n.id === id);
    if (notification) {
      notification.read = true;
    }
  };

  const markAllAsRead = () => {
    notifications.value.forEach((n) => {
      n.read = true;
    });
  };

  const clearAll = () => {
    notifications.value = [];
  };

  const unreadCount = computed(() =>
    notifications.value.filter((n) => !n.read).length
  );

  return {
    notifications: readonly(notifications),
    addNotification,
    removeNotification,
    markAsRead,
    markAllAsRead,
    clearAll,
    unreadCount,
  };
}
