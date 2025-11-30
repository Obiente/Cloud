<template>
  <ClientOnly>
    <OuiFloatingPanel
      v-model="open"
      title="Notifications"
      :description="description || `${unreadCount} unread`"
      :default-position="clientPosition"
      :persist-rect="true"
      content-class="max-w-[800px] w-full h-[85vh] max-h-[900px] min-h-[600px] flex flex-col"
      body-class="flex-1 flex flex-col min-h-0 overflow-hidden p-0"
      role="dialog"
      aria-labelledby="notifications-title"
      aria-describedby="notifications-description"
      @close="handleClose"
    >
      <div class="flex flex-col h-full min-h-0" role="list" aria-label="Notifications list">
        <!-- Header with filters and actions -->
        <div class="bg-surface-base border-b border-border-muted px-4 md:px-6 pt-4 pb-3 flex-shrink-0">
          <OuiStack gap="sm">
            <!-- Filter tabs -->
            <OuiFlex
              gap="xs"
              class="flex-1 flex-wrap md:flex-nowrap overflow-x-auto min-w-0"
            >
              <OuiButton
                v-for="filter in filters"
                :key="filter.key"
                :variant="activeFilter === filter.key ? 'soft' : 'ghost'"
                :color="activeFilter === filter.key ? 'primary' : 'neutral'"
                size="sm"
                @click="activeFilter = filter.key"
                class="whitespace-nowrap flex-shrink-0"
              >
                {{ filter.label }}
                <OuiBox
                  v-if="filter.count !== undefined && filter.count > 0"
                  class="ml-1.5 px-1.5 py-0.5 rounded-full bg-primary/20 text-primary text-xs font-medium min-w-[1.25rem] text-center"
                >
                  {{ filter.count > 99 ? '99+' : filter.count }}
                </OuiBox>
              </OuiButton>
            </OuiFlex>

            <!-- Action buttons -->
            <OuiFlex
              justify="end"
              align="center"
              gap="xs"
              class="flex-wrap"
            >
              <OuiButton
                variant="ghost"
                size="sm"
                @click="markAllRead"
                :disabled="unreadCount === 0 || isLoading"
                :loading="isMarkingAllRead"
              >
                Mark all read
              </OuiButton>
              <OuiButton
                variant="ghost"
                size="sm"
                color="danger"
                @click="clearAll"
                :disabled="filteredItems.length === 0 || isLoading"
                :loading="isClearingAll"
              >
                Clear
              </OuiButton>
            </OuiFlex>
          </OuiStack>
        </div>

        <!-- Scrollable content area -->
        <div class="flex-1 min-h-0 overflow-y-auto px-4 md:px-6">
          <!-- Loading state -->
          <div v-if="isLoading && filteredItems.length === 0" class="py-12">
            <OuiStack gap="md" align="center">
              <OuiSpinner size="lg" />
              <OuiText color="secondary" size="md">Loading notifications...</OuiText>
            </OuiStack>
          </div>

          <!-- Empty state -->
          <div
            v-else-if="filteredItems.length === 0"
            class="py-12 text-center"
            role="status"
            aria-live="polite"
          >
            <OuiStack gap="md" align="center">
              <div class="w-16 h-16 rounded-full bg-surface-muted flex items-center justify-center">
                <BellIcon class="w-8 h-8 text-foreground-muted" />
              </div>
              <OuiStack gap="xs" align="center">
                <OuiText size="lg" weight="medium" color="primary">
                  {{ activeFilter === 'all' ? "You're all caught up!" : `No ${filterLabels[activeFilter]} notifications` }}
                </OuiText>
                <OuiText size="md" color="secondary" class="max-w-sm">
                  {{ activeFilter === 'all' 
                    ? "You don't have any notifications right now. We'll notify you when something important happens." 
                    : `You don't have any ${filterLabels[activeFilter]} notifications.` }}
                </OuiText>
              </OuiStack>
            </OuiStack>
          </div>

          <!-- Notifications list -->
          <OuiStack v-else gap="lg" class="py-4">
            <template v-for="(group, groupIndex) in groupedNotifications" :key="group.date">
              <!-- Date group header -->
              <div v-if="group.date" class="py-2">
                <OuiText size="xs" weight="semibold" color="secondary" class="uppercase tracking-wide">
                  {{ group.date }}
                </OuiText>
              </div>

              <!-- Notifications in this group -->
              <div class="space-y-3">
                <article
                  v-for="n in group.items"
                  :key="n.id"
                  :class="getRowClasses(n)"
                  role="listitem"
                  :aria-label="`${n.read ? 'Read' : 'Unread'} ${n.type || 'notification'}: ${n.title}`"
                  tabindex="0"
                  @click="handleNotificationClick(n)"
                  @keydown.enter="handleNotificationClick(n)"
                  @keydown.space.prevent="handleNotificationClick(n)"
                >
                  <OuiStack gap="md" class="min-w-0">
                    <OuiFlex gap="md" class="min-w-0" align="start">
                      <!-- Notification Icon -->
                      <div
                        class="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm"
                        :class="getIconClasses(n)"
                        :aria-hidden="true"
                      >
                        <component :is="getNotificationIcon(n)" class="w-4 h-4" />
                      </div>

                      <OuiStack gap="sm" class="min-w-0 flex-1">
                        <!-- Title and time -->
                        <OuiFlex justify="between" align="start" class="gap-sm flex-wrap">
                          <OuiStack gap="xs" class="flex-1 min-w-0">
                            <OuiText
                              size="lg"
                              :weight="n.read ? 'medium' : 'semibold'"
                              :color="n.read ? 'secondary' : 'primary'"
                              class="leading-tight break-words"
                            >
                              {{ n.title }}
                            </OuiText>
                            <OuiText
                              size="sm"
                              color="secondary"
                              class="break-words leading-relaxed whitespace-pre-line"
                            >
                              {{ n.message }}
                            </OuiText>
                          </OuiStack>
                        </OuiFlex>

                        <!-- Metadata row -->
                        <OuiFlex
                          v-if="n.metadata?.details"
                          gap="md"
                          wrap="wrap"
                          class="text-xs text-secondary"
                        >
                          <span>{{ n.metadata.details }}</span>
                        </OuiFlex>

                      </OuiStack>
                    </OuiFlex>

                    <!-- Secondary actions -->
                    <OuiFlex
                      gap="sm"
                      align="center"
                      class="flex-wrap border-t border-border-muted pt-3 justify-end"
                    >
                      <OuiFlex gap="xs" align="center" class="flex-wrap justify-end">
                        <OuiButton
                          v-if="n.actionUrl && n.actionLabel"
                          variant="ghost"
                          size="xs"
                          @click.stop="handleActionClick(n)"
                          :aria-label="`${n.actionLabel} for ${n.title}`"
                          class="!p-1.5"
                        >
                          <ArrowRightIcon class="w-4 h-4" />
                        </OuiButton>

                        <OuiButton
                          variant="ghost"
                          size="sm"
                          @click.stop="toggleRead(n.id)"
                          :aria-label="n.read ? 'Mark as unread' : 'Mark as read'"
                        >
                          <component
                            :is="n.read ? EnvelopeIcon : EnvelopeOpenIcon"
                            class="w-4 h-4"
                          />
                          <span class="sr-only">
                            {{ n.read ? "Mark as unread" : "Mark as read" }}
                          </span>
                        </OuiButton>
                        <OuiButton
                          variant="ghost"
                          size="sm"
                          color="danger"
                          @click.stop="remove(n.id)"
                          :aria-label="`Dismiss notification: ${n.title}`"
                        >
                          <XMarkIcon class="w-4 h-4" />
                          <span class="sr-only">Dismiss</span>
                        </OuiButton>
                      </OuiFlex>

                      <OuiText size="xs" color="muted" class="flex-shrink-0">
                        <OuiRelativeTime :value="n.timestamp" :style="'short'" />
                      </OuiText>
                    </OuiFlex>
                  </OuiStack>
                </article>
              </div>
            </template>
          </OuiStack>
        </div>
      </div>
    </OuiFloatingPanel>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch, nextTick } from "vue";
import OuiFloatingPanel from "~/components/oui/FloatingPanel.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiSpinner from "~/components/oui/Spinner.vue";
import {
  BellIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  RocketLaunchIcon,
  CreditCardIcon,
  ChartBarIcon,
  UserPlusIcon,
  Cog6ToothIcon,
  InformationCircleIcon,
  ArrowRightIcon,
  EnvelopeIcon,
  EnvelopeOpenIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";

interface NotificationItem {
  id: string;
  title: string;
  message: string;
  timestamp: Date;
  read?: boolean;
  type?: string;
  severity?: string;
  actionUrl?: string;
  actionLabel?: string;
  metadata?: Record<string, string>;
  clientOnly?: boolean;
}

const props = defineProps<{
  modelValue: boolean;
  items: NotificationItem[];
  description?: string;
  anchorElement?: HTMLElement | null;
  isLoading?: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
  "update:items": [items: NotificationItem[]];
  clear: [];
}>();

const open = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit("update:modelValue", v),
});

const router = useRouter();

// Filter state
const activeFilter = ref<"all" | "unread" | "read" | "critical">("all");
const isMarkingAllRead = ref(false);
const isClearingAll = ref(false);

// Filter definitions
const filters = computed(() => [
  { key: "all" as const, label: "All", count: props.items.length },
  { key: "unread" as const, label: "Unread", count: unreadCount.value },
  { key: "read" as const, label: "Read", count: readCount.value },
  { key: "critical" as const, label: "Critical", count: criticalCount.value },
]);

const filterLabels: Record<string, string> = {
  all: "all",
  unread: "unread",
  read: "read",
  critical: "critical",
};

// Filtered items
const filteredItems = computed(() => {
  let items = props.items;

  switch (activeFilter.value) {
    case "unread":
      items = items.filter((n) => !n.read);
      break;
    case "read":
      items = items.filter((n) => n.read);
      break;
    case "critical":
      items = items.filter((n) => n.severity?.toUpperCase() === "CRITICAL");
      break;
  }

  // Sort by timestamp (newest first), then by read status (unread first)
  return items.sort((a, b) => {
    if (a.read !== b.read) {
      return a.read ? 1 : -1;
    }
    return b.timestamp.getTime() - a.timestamp.getTime();
  });
});

// Group notifications by date
const groupedNotifications = computed(() => {
  const groups: Array<{ date: string; items: NotificationItem[] }> = [];
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);
  const thisWeek = new Date(today);
  thisWeek.setDate(thisWeek.getDate() - 7);

  const groupsMap = new Map<string, NotificationItem[]>();

  filteredItems.value.forEach((item) => {
    const itemDate = new Date(item.timestamp);
    itemDate.setHours(0, 0, 0, 0);

    let groupKey: string;
    if (itemDate.getTime() === today.getTime()) {
      groupKey = "Today";
    } else if (itemDate.getTime() === yesterday.getTime()) {
      groupKey = "Yesterday";
    } else if (itemDate.getTime() >= thisWeek.getTime()) {
      groupKey = "This Week";
    } else {
      groupKey = itemDate.toLocaleDateString("en-US", {
        month: "long",
        day: "numeric",
        year: itemDate.getFullYear() !== today.getFullYear() ? "numeric" : undefined,
      });
    }

    if (!groupsMap.has(groupKey)) {
      groupsMap.set(groupKey, []);
    }
    groupsMap.get(groupKey)!.push(item);
  });

  // Convert to array and sort groups
  groupsMap.forEach((items, date) => {
    groups.push({ date, items });
  });

  // Sort groups by date (newest first)
  return groups.sort((a, b) => {
    const dateOrder = ["Today", "Yesterday", "This Week"];
    const aIndex = dateOrder.indexOf(a.date);
    const bIndex = dateOrder.indexOf(b.date);
    if (aIndex !== -1 && bIndex !== -1) return aIndex - bIndex;
    if (aIndex !== -1) return -1;
    if (bIndex !== -1) return 1;
    return b.date.localeCompare(a.date);
  });
});

const unreadCount = computed(() => props.items.filter((n) => !n.read).length);
const readCount = computed(() => props.items.filter((n) => n.read).length);
const criticalCount = computed(() =>
  props.items.filter((n) => n.severity?.toUpperCase() === "CRITICAL" && !n.read).length
);

const handleClose = () => {
  emit("update:modelValue", false);
  emit("close");
};

const handleNotificationClick = (notification: NotificationItem) => {
  // If notification has an action URL, navigate to it
  if (notification.actionUrl) {
    router.push(notification.actionUrl);
    handleClose();
    return;
  }

  // If notification is about invites, navigate to invites page
  if (
    notification.title?.toLowerCase().includes("invitation") ||
    notification.message?.toLowerCase().includes("invited")
  ) {
    router.push("/invites");
    handleClose();
  }
};

const handleActionClick = (notification: NotificationItem) => {
  if (notification.actionUrl) {
    router.push(notification.actionUrl);
    handleClose();
  }
};

// Get notification icon based on severity, then type
const getNotificationIcon = (notification: NotificationItem) => {
  const severity = notification.severity?.toUpperCase();
  if (severity) {
    switch (severity) {
      case "CRITICAL":
        return XCircleIcon;
      case "HIGH":
        return ExclamationTriangleIcon;
      case "MEDIUM":
        return InformationCircleIcon;
      case "LOW":
        return CheckCircleIcon;
    }
  }

  const type = notification.type?.toUpperCase() || "INFO";
  switch (type) {
    case "SUCCESS":
      return CheckCircleIcon;
    case "WARNING":
      return ExclamationTriangleIcon;
    case "ERROR":
      return XCircleIcon;
    case "DEPLOYMENT":
      return RocketLaunchIcon;
    case "BILLING":
      return CreditCardIcon;
    case "QUOTA":
      return ChartBarIcon;
    case "INVITE":
      return UserPlusIcon;
    case "SYSTEM":
      return Cog6ToothIcon;
    default:
      return InformationCircleIcon;
  }
};

const notificationVisuals: Record<
  string,
  { iconBg: string; iconColor: string }
> = {
  SUCCESS: { iconBg: "bg-success/10", iconColor: "text-success" },
  WARNING: { iconBg: "bg-warning/10", iconColor: "text-warning" },
  ERROR: { iconBg: "bg-danger/10", iconColor: "text-danger" },
  DEPLOYMENT: { iconBg: "bg-primary/10", iconColor: "text-primary" },
  BILLING: { iconBg: "bg-accent/10", iconColor: "text-accent" },
  QUOTA: { iconBg: "bg-warning/10", iconColor: "text-warning" },
  INVITE: { iconBg: "bg-info/10", iconColor: "text-info" },
  SYSTEM: { iconBg: "bg-secondary/10", iconColor: "text-secondary" },
  INFO: { iconBg: "bg-primary/10", iconColor: "text-primary" },
};

const getIconClasses = (notification: NotificationItem) => {
  if (notification.read) {
    return "bg-surface-muted text-foreground-muted";
  }
  const type = notification.type?.toUpperCase() || "INFO";
  const visual = notificationVisuals[type] ?? notificationVisuals.INFO!;
  return `${visual.iconBg} ${visual.iconColor}`;
};

const severityAccentClasses: Record<string, string> = {
  CRITICAL: "border-danger/70 bg-danger/5 shadow-danger/20 shadow-sm",
  HIGH: "border-warning/60 bg-warning/5",
  MEDIUM: "border-info/50 bg-info/5",
};

const typeAccentClasses: Record<string, string> = {
  SUCCESS: "border-success/50",
  WARNING: "border-warning/50",
  ERROR: "border-danger/60",
  DEPLOYMENT: "border-primary/50",
  BILLING: "border-accent/50",
  QUOTA: "border-warning/40",
  INVITE: "border-info/50",
  SYSTEM: "border-secondary/40",
  INFO: "border-border-muted",
};

const getRowClasses = (notification: NotificationItem) => {
  const base =
    "notification-row border rounded-xl p-4 focus-within:ring-2 focus-within:ring-primary/30 transition-colors cursor-pointer";
  const background = notification.read ? "bg-surface-base" : "bg-surface-muted";
  const severity = notification.severity?.toUpperCase();
  const type = notification.type?.toUpperCase() || "INFO";
  const accent =
    (severity && severityAccentClasses[severity]) ||
    typeAccentClasses[type] ||
    "border-border-muted";

  return `${base} ${background} ${accent}`;
};

// Calculate default position on the far right
const defaultPosition = ref<{ x: number; y: number }>({ x: 0, y: 80 });
const clientPosition = computed(() => {
  if (import.meta.client) {
    return defaultPosition.value;
  }
  return { x: 0, y: 80 };
});

// Update position when anchor element changes or on mount
const updatePosition = () => {
  if (!import.meta.client) return;

  try {
    if (window && window.innerWidth) {
      const panelWidth = 800;
      const padding = 16;
      // Position on the far right with padding
      const xPos = window.innerWidth - panelWidth - padding;
      
      // Get Y position from anchor if available, otherwise use default
      let yPos = 80;
      const anchor = props.anchorElement;
      if (anchor) {
        const rect = anchor.getBoundingClientRect();
        yPos = rect.bottom + 8;
      }
      
      defaultPosition.value = { x: xPos, y: yPos };
    }
  } catch (e) {
    console.debug("Could not set notification panel position:", e);
  }
};

onMounted(() => {
  if (import.meta.client) {
    updatePosition();
  }
});

watch(
  () => props.anchorElement,
  () => {
    if (import.meta.client) {
      updatePosition();
    }
  },
  { immediate: true }
);

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen && import.meta.client) {
      nextTick(() => {
        updatePosition();
      });
    }
  }
);

async function markAllRead() {
  isMarkingAllRead.value = true;
  try {
    emit(
      "update:items",
      props.items.map((n) => ({ ...n, read: true }))
    );
  } finally {
    isMarkingAllRead.value = false;
  }
}

async function clearAll() {
  isClearingAll.value = true;
  try {
    emit("clear");
    emit("update:items", []);
  } finally {
    isClearingAll.value = false;
  }
}

function toggleRead(id: string) {
  emit(
    "update:items",
    props.items.map((n) => (n.id === id ? { ...n, read: !n.read } : n))
  );
}

function remove(id: string) {
  emit(
    "update:items",
    props.items.filter((n) => n.id !== id)
  );
}
</script>
