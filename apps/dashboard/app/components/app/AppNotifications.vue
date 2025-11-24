<template>
  <ClientOnly>
    <OuiFloatingPanel
      v-model="open"
      title="Notifications"
      :description="description || `${unreadCount} unread`"
      :default-position="clientPosition"
      :persist-rect="true"
      content-class="max-w-[600px] w-full"
      @close="handleClose"
      role="dialog"
      aria-labelledby="notifications-title"
      aria-describedby="notifications-description"
    >
      <div class="w-full" role="list" aria-label="Notifications list">
        <!-- Header with filters and actions -->
        <div class="sticky top-0 z-10 bg-surface-base/95 backdrop-blur-sm border-b border-border-muted pb-3 mb-4 -mx-4 px-4 pt-2">
          <OuiFlex justify="between" align="center" gap="sm" class="mb-3">
            <!-- Filter tabs -->
            <OuiFlex gap="xs" class="flex-1 overflow-x-auto">
              <OuiButton
                v-for="filter in filters"
                :key="filter.key"
                :variant="activeFilter === filter.key ? 'soft' : 'ghost'"
                :color="activeFilter === filter.key ? 'primary' : 'neutral'"
                size="xs"
                @click="activeFilter = filter.key"
                class="whitespace-nowrap"
                :aria-label="`Filter by ${filter.label}`"
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
          </OuiFlex>

          <!-- Action buttons -->
          <OuiFlex justify="end" align="center" gap="xs">
            <OuiButton
              variant="ghost"
              size="xs"
              @click="markAllRead"
              :disabled="unreadCount === 0 || isLoading"
              :loading="isMarkingAllRead"
              aria-label="Mark all notifications as read"
              class="text-xs"
            >
              Mark all read
            </OuiButton>
            <OuiButton
              variant="ghost"
              size="xs"
              color="danger"
              @click="clearAll"
              :disabled="filteredItems.length === 0 || isLoading"
              :loading="isClearingAll"
              aria-label="Clear all notifications"
              class="text-xs"
            >
              Clear
            </OuiButton>
          </OuiFlex>
        </div>

        <!-- Loading state -->
        <div v-if="isLoading && filteredItems.length === 0" class="py-12">
          <OuiStack gap="md" align="center">
            <div class="animate-spin rounded-full h-8 w-8 border-2 border-primary border-t-transparent" />
            <OuiText color="secondary" size="sm">Loading notifications...</OuiText>
          </OuiStack>
        </div>

        <!-- Empty state -->
        <div
          v-else-if="filteredItems.length === 0"
          class="py-12 text-center"
          role="status"
          aria-live="polite"
        >
          <OuiStack gap="sm" align="center">
            <div class="w-16 h-16 rounded-full bg-surface-muted flex items-center justify-center">
              <BellIcon class="w-8 h-8 text-foreground-muted" />
            </div>
            <OuiStack gap="xs" align="center">
              <OuiText size="lg" weight="medium" color="primary">
                {{ activeFilter === 'all' ? "You're all caught up!" : `No ${filterLabels[activeFilter]} notifications` }}
              </OuiText>
              <OuiText size="sm" color="secondary" class="max-w-sm">
                {{ activeFilter === 'all' 
                  ? "You don't have any notifications right now. We'll notify you when something important happens." 
                  : `You don't have any ${filterLabels[activeFilter]} notifications.` }}
              </OuiText>
            </OuiStack>
          </OuiStack>
        </div>

        <!-- Notifications list with grouping -->
        <OuiStack v-else gap="md" class="max-h-[600px] overflow-y-auto">
          <template v-for="(group, groupIndex) in groupedNotifications" :key="group.date">
            <!-- Date group header -->
            <div v-if="group.date" class="sticky top-0 z-10 -mx-2 px-2 py-1 bg-surface-base/95 backdrop-blur-sm">
              <OuiText size="xs" weight="semibold" color="secondary" class="uppercase tracking-wide">
                {{ group.date }}
              </OuiText>
            </div>

            <!-- Notifications in this group -->
            <TransitionGroup
              name="notification"
              tag="div"
              class="space-y-2"
            >
              <OuiCard
                v-for="n in group.items"
                :key="n.id"
                variant="overlay"
                :class="[
                  'notification-item transition-all duration-200 ease-out',
                  'ring-1 cursor-pointer group',
                  'hover:shadow-md hover:scale-[1.01]',
                  'focus-within:ring-2 focus-within:ring-primary focus-within:shadow-lg',
                  n.read 
                    ? 'opacity-70 ring-border-muted bg-surface-base hover:opacity-90' 
                    : 'ring-border-default bg-surface-elevated shadow-sm',
                  getNotificationClasses(n),
                  isRemoving.has(n.id) ? 'opacity-0 scale-95' : ''
                ]"
                role="listitem"
                :aria-label="`${n.read ? 'Read' : 'Unread'} ${n.type || 'notification'}: ${n.title}`"
                tabindex="0"
                @click="handleNotificationClick(n)"
                @keydown.enter="handleNotificationClick(n)"
                @keydown.space.prevent="handleNotificationClick(n)"
              >
                <OuiCardBody class="p-4">
                  <OuiFlex justify="between" align="start" gap="md" class="min-w-0">
                    <OuiFlex gap="md" class="min-w-0 flex-1">
                      <!-- Notification Icon with animation -->
                      <div
                        :class="[
                          'flex-shrink-0 w-12 h-12 rounded-xl flex items-center justify-center transition-all duration-200',
                          'group-hover:scale-110',
                          getIconClasses(n)
                        ]"
                        :aria-hidden="true"
                      >
                        <component :is="getNotificationIcon(n)" class="w-6 h-6" />
                      </div>

                      <OuiStack gap="xs" class="min-w-0 flex-1">
                        <!-- Title and unread indicator -->
                        <OuiFlex align="center" gap="xs" class="flex-wrap">
                          <OuiText
                            size="sm"
                            weight="semibold"
                            :color="n.read ? 'secondary' : 'primary'"
                            class="truncate flex-1 min-w-0"
                          >
                            {{ n.title }}
                          </OuiText>
                          <!-- Unread indicator with pulse animation -->
                          <div
                            v-if="!n.read"
                            class="w-2.5 h-2.5 rounded-full bg-primary flex-shrink-0 animate-pulse"
                            :aria-label="'Unread notification'"
                          />
                        </OuiFlex>

                        <!-- Message -->
                        <OuiText
                          size="sm"
                          :color="n.read ? 'secondary' : 'primary'"
                          class="line-clamp-2 break-words"
                        >
                          {{ n.message }}
                        </OuiText>

                        <!-- Metadata row -->
                        <OuiFlex align="center" gap="sm" class="flex-wrap">
                          <OuiText size="xs" color="muted">
                            <OuiRelativeTime :value="n.timestamp" :style="'short'" />
                          </OuiText>
                          <!-- Severity badge -->
                          <OuiBox
                            v-if="n.severity && n.severity !== 'MEDIUM'"
                            :class="[
                              'px-2 py-0.5 rounded-md text-xs font-medium border',
                              getSeverityBadgeClasses(n.severity)
                            ]"
                          >
                            {{ n.severity }}
                          </OuiBox>
                          <!-- Type badge -->
                          <OuiBox
                            v-if="n.type && n.type !== 'INFO'"
                            :class="[
                              'px-2 py-0.5 rounded-md text-xs font-medium',
                              'bg-surface-muted text-foreground-muted'
                            ]"
                          >
                            {{ n.type }}
                          </OuiBox>
                        </OuiFlex>

                        <!-- Action button -->
                        <OuiButton
                          v-if="n.actionUrl && n.actionLabel"
                          variant="soft"
                          size="xs"
                          @click.stop="handleActionClick(n)"
                          class="self-start mt-1"
                          :aria-label="`${n.actionLabel} for ${n.title}`"
                        >
                          {{ n.actionLabel }}
                          <ArrowRightIcon class="w-3.5 h-3.5 ml-1" />
                        </OuiButton>
                      </OuiStack>
                    </OuiFlex>

                    <!-- Action buttons -->
                    <OuiFlex gap="xs" class="flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click.stop="toggleRead(n.id)"
                        :aria-label="n.read ? 'Mark as unread' : 'Mark as read'"
                        class="!p-1.5"
                      >
                        <component
                          :is="n.read ? EnvelopeIcon : EnvelopeOpenIcon"
                          class="w-4 h-4"
                        />
                      </OuiButton>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        color="danger"
                        @click.stop="remove(n.id)"
                        :aria-label="`Dismiss notification: ${n.title}`"
                        class="!p-1.5"
                      >
                        <XMarkIcon class="w-4 h-4" />
                      </OuiButton>
                    </OuiFlex>
                  </OuiFlex>
                </OuiCardBody>
              </OuiCard>
            </TransitionGroup>
          </template>
        </OuiStack>
      </div>
    </OuiFloatingPanel>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch, nextTick } from "vue";
import OuiFloatingPanel from "~/components/oui/FloatingPanel.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
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
import {
  BellIcon as BellIconSolid,
  CheckCircleIcon as CheckCircleIconSolid,
  ExclamationTriangleIcon as ExclamationTriangleIconSolid,
  XCircleIcon as XCircleIconSolid,
  RocketLaunchIcon as RocketLaunchIconSolid,
  CreditCardIcon as CreditCardIconSolid,
  ChartBarIcon as ChartBarIconSolid,
  UserPlusIcon as UserPlusIconSolid,
  Cog6ToothIcon as Cog6ToothIconSolid,
  InformationCircleIcon as InformationCircleIconSolid,
} from "@heroicons/vue/24/solid";

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
const isRemoving = ref(new Set<string>());

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

// Get notification icon based on type
const getNotificationIcon = (notification: NotificationItem) => {
  const isUnread = !notification.read;
  const type = notification.type?.toUpperCase() || "INFO";

  switch (type) {
    case "SUCCESS":
      return isUnread ? CheckCircleIconSolid : CheckCircleIcon;
    case "WARNING":
      return isUnread ? ExclamationTriangleIconSolid : ExclamationTriangleIcon;
    case "ERROR":
      return isUnread ? XCircleIconSolid : XCircleIcon;
    case "DEPLOYMENT":
      return isUnread ? RocketLaunchIconSolid : RocketLaunchIcon;
    case "BILLING":
      return isUnread ? CreditCardIconSolid : CreditCardIcon;
    case "QUOTA":
      return isUnread ? ChartBarIconSolid : ChartBarIcon;
    case "INVITE":
      return isUnread ? UserPlusIconSolid : UserPlusIcon;
    case "SYSTEM":
      return isUnread ? Cog6ToothIconSolid : Cog6ToothIcon;
    default:
      return isUnread ? InformationCircleIconSolid : InformationCircleIcon;
  }
};

// Get notification classes based on type and severity
const getNotificationClasses = (notification: NotificationItem) => {
  const classes: string[] = [];
  const type = notification.type?.toUpperCase() || "INFO";
  const severity = notification.severity?.toUpperCase() || "MEDIUM";

  if (!notification.read) {
    switch (type) {
      case "SUCCESS":
        classes.push("border-l-4 border-l-success");
        break;
      case "WARNING":
        classes.push("border-l-4 border-l-warning");
        break;
      case "ERROR":
        classes.push("border-l-4 border-l-danger");
        break;
      case "DEPLOYMENT":
        classes.push("border-l-4 border-l-primary");
        break;
      case "BILLING":
        classes.push("border-l-4 border-l-accent");
        break;
      case "QUOTA":
        classes.push("border-l-4 border-l-warning");
        break;
      case "INVITE":
        classes.push("border-l-4 border-l-info");
        break;
      case "SYSTEM":
        classes.push("border-l-4 border-l-secondary");
        break;
    }

    if (severity === "CRITICAL") {
      classes.push("ring-2 ring-danger/50 shadow-danger/20");
    } else if (severity === "HIGH") {
      classes.push("ring-1 ring-warning/50");
    }
  }

  return classes.join(" ");
};

// Get icon container classes
const getIconClasses = (notification: NotificationItem) => {
  const classes: string[] = [];
  const type = notification.type?.toUpperCase() || "INFO";
  const isUnread = !notification.read;

  if (isUnread) {
    switch (type) {
      case "SUCCESS":
        classes.push("bg-success/15 text-success ring-1 ring-success/20");
        break;
      case "WARNING":
        classes.push("bg-warning/15 text-warning ring-1 ring-warning/20");
        break;
      case "ERROR":
        classes.push("bg-danger/15 text-danger ring-1 ring-danger/20");
        break;
      case "DEPLOYMENT":
        classes.push("bg-primary/15 text-primary ring-1 ring-primary/20");
        break;
      case "BILLING":
        classes.push("bg-accent/15 text-accent ring-1 ring-accent/20");
        break;
      case "QUOTA":
        classes.push("bg-warning/15 text-warning ring-1 ring-warning/20");
        break;
      case "INVITE":
        classes.push("bg-info/15 text-info ring-1 ring-info/20");
        break;
      case "SYSTEM":
        classes.push("bg-secondary/15 text-secondary ring-1 ring-secondary/20");
        break;
      default:
        classes.push("bg-primary/15 text-primary ring-1 ring-primary/20");
    }
  } else {
    classes.push("bg-surface-muted text-foreground-muted");
  }

  return classes.join(" ");
};

// Get severity badge classes
const getSeverityBadgeClasses = (severity: string) => {
  const severityUpper = severity.toUpperCase();
  switch (severityUpper) {
    case "CRITICAL":
      return "bg-danger/15 text-danger border-danger/30 ring-1 ring-danger/20";
    case "HIGH":
      return "bg-warning/15 text-warning border-warning/30 ring-1 ring-warning/20";
    case "MEDIUM":
      return "bg-info/15 text-info border-info/30 ring-1 ring-info/20";
    case "LOW":
      return "bg-secondary/15 text-secondary border-secondary/30 ring-1 ring-secondary/20";
    default:
      return "bg-surface-muted text-foreground-muted border-border-muted";
  }
};

// Calculate default position underneath the notification button
const defaultPosition = ref<{ x: number; y: number }>({ x: 100, y: 80 });
const clientPosition = computed(() => {
  if (import.meta.client) {
    return defaultPosition.value;
  }
  return { x: 100, y: 80 };
});

// Update position when anchor element changes or on mount
const updatePosition = () => {
  if (!import.meta.client) return;

  try {
    const anchor = props.anchorElement;
    if (anchor && window) {
      const rect = anchor.getBoundingClientRect();
      const panelWidth = 600;
      const xPos = Math.max(16, rect.right - panelWidth);
      defaultPosition.value = {
        x: xPos,
        y: rect.bottom + 8,
      };
    } else if (window && window.innerWidth) {
      defaultPosition.value = { x: window.innerWidth - 620, y: 80 };
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
    // Small delay for visual feedback
    await new Promise((resolve) => setTimeout(resolve, 300));
  } finally {
    isMarkingAllRead.value = false;
  }
}

async function clearAll() {
  isClearingAll.value = true;
  try {
    // Animate out before clearing
    filteredItems.value.forEach((item) => {
      isRemoving.value.add(item.id);
    });
    await new Promise((resolve) => setTimeout(resolve, 200));
    emit("update:items", []);
    isRemoving.value.clear();
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
  isRemoving.value.add(id);
  setTimeout(() => {
    emit(
      "update:items",
      props.items.filter((n) => n.id !== id)
    );
    isRemoving.value.delete(id);
  }, 200);
}
</script>

<style scoped>
/* Notification enter/leave animations */
.notification-enter-active {
  transition: all 0.3s ease-out;
}

.notification-leave-active {
  transition: all 0.2s ease-in;
}

.notification-enter-from {
  opacity: 0;
  transform: translateY(-10px) scale(0.95);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(20px) scale(0.95);
}

.notification-move {
  transition: transform 0.3s ease-out;
}

/* Smooth scrollbar */
:deep(.max-h-\[600px\]) {
  scrollbar-width: thin;
  scrollbar-color: rgba(var(--color-border-muted), 0.5) transparent;
}

:deep(.max-h-\[600px\])::-webkit-scrollbar {
  width: 6px;
}

:deep(.max-h-\[600px\])::-webkit-scrollbar-track {
  background: transparent;
}

:deep(.max-h-\[600px\])::-webkit-scrollbar-thumb {
  background-color: rgba(var(--color-border-muted), 0.5);
  border-radius: 3px;
}

:deep(.max-h-\[600px\])::-webkit-scrollbar-thumb:hover {
  background-color: rgba(var(--color-border-default), 0.7);
}
</style>
