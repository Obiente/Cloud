<template>
  <ClientOnly>
    <Teleport to="body">

      <!-- Mobile backdrop -->
      <Transition
        enter-active-class="transition-opacity duration-200"
        leave-active-class="transition-opacity duration-150"
        enter-from-class="opacity-0"
        leave-to-class="opacity-0"
      >
        <div
          v-if="open"
          class="fixed inset-0 z-998 bg-black/50 md:hidden"
          @click="handleClose"
        />
      </Transition>

      <!-- Panel -->
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        leave-active-class="transition-all duration-200 ease-in"
        enter-from-class="opacity-0 translate-y-8 md:translate-y-0 md:scale-95 md:translate-x-2"
        leave-to-class="opacity-0 translate-y-8 md:translate-y-0 md:scale-95 md:translate-x-2"
      >
        <div
          v-if="open"
          role="dialog"
          aria-label="Notifications"
          class="fixed z-999 flex flex-col overflow-hidden bg-surface-overlay border border-border-default shadow-2xl
                 bottom-0 left-0 right-0 max-h-[88vh] rounded-t-2xl
                 md:top-14 md:right-4 md:bottom-auto md:left-auto md:w-[420px] md:h-[calc(100vh-5rem)] md:max-h-[680px] md:rounded-xl md:border"
        >
          <!-- Mobile drag handle -->
          <div class="flex justify-center pt-2.5 pb-1 md:hidden shrink-0">
            <div class="w-10 h-1 rounded-full bg-border-strong opacity-60" />
          </div>

          <!-- Header -->
          <div class="flex items-center justify-between gap-3 px-4 py-3 border-b border-border-default shrink-0">
            <div class="flex items-center gap-2">
              <BellIcon class="w-5 h-5 text-text-secondary shrink-0" />
              <OuiText as="h3" size="md" weight="semibold">Notifications</OuiText>
              <span
                v-if="unreadCount > 0"
                class="inline-flex items-center justify-center min-w-[1.25rem] h-5 rounded-full bg-danger text-white text-[11px] font-bold px-1.5 shrink-0"
              >{{ unreadCount > 99 ? '99+' : unreadCount }}</span>
            </div>
            <button
              class="flex items-center justify-center w-8 h-8 rounded-lg hover:bg-surface-muted transition-colors"
              title="Close"
              @click="handleClose"
            >
              <XMarkIcon class="w-4.5 h-4.5 text-text-secondary" />
            </button>
          </div>

          <!-- Critical banner -->
          <div
            v-if="criticalCount > 0"
            class="flex items-center gap-2 px-4 py-2.5 bg-danger/10 border-b border-danger/20 shrink-0"
          >
            <XCircleIcon class="w-4 h-4 text-danger shrink-0" />
            <OuiText size="sm" weight="medium" class="flex-1 text-danger leading-snug">
              {{ criticalCount }} critical notification{{ criticalCount !== 1 ? 's' : '' }}
              {{ criticalCount === 1 ? 'requires' : 'require' }} attention
            </OuiText>
            <button
              v-if="activeFilter !== 'critical'"
              class="text-sm font-semibold text-danger hover:underline shrink-0"
              @click="activeFilter = 'critical'"
            >View</button>
          </div>

          <!-- Filter + actions bar -->
          <div class="flex items-center justify-between gap-3 px-3 py-2 border-b border-border-muted shrink-0">
            <div class="flex items-center gap-1 overflow-x-auto min-w-0">
              <button
                v-for="f in filters"
                :key="f.key"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium transition-all whitespace-nowrap shrink-0 border"
                :class="activeFilter === f.key
                  ? 'bg-primary border-primary text-white'
                  : 'bg-transparent border-transparent text-text-secondary hover:bg-surface-muted hover:text-text-primary'"
                @click="activeFilter = f.key"
              >
                {{ f.label }}
                <span
                  v-if="f.count > 0"
                  class="text-[11px] font-bold"
                  :class="activeFilter === f.key ? 'text-white/80' : 'text-text-tertiary'"
                >{{ f.count > 99 ? '99+' : f.count }}</span>
              </button>
            </div>
            <button
              v-if="unreadCount > 0"
              class="text-sm font-medium text-primary whitespace-nowrap shrink-0 hover:opacity-70 transition-opacity disabled:opacity-30"
              :disabled="isMarkingAllRead"
              @click="markAllRead"
            >{{ isMarkingAllRead ? 'Marking…' : 'Mark all read' }}</button>
          </div>

          <!-- Scrollable content -->
          <div class="flex-1 min-h-0 overflow-y-auto">

            <!-- Loading -->
            <div v-if="isLoading && filteredItems.length === 0" class="flex items-center justify-center gap-3 py-20">
              <OuiSpinner size="md" />
              <OuiText size="sm" color="tertiary">Loading notifications…</OuiText>
            </div>

            <!-- Empty -->
            <div v-else-if="filteredItems.length === 0" class="flex flex-col items-center gap-4 py-20 px-6 text-center">
              <div class="w-14 h-14 rounded-full bg-surface-muted flex items-center justify-center">
                <BellIcon class="w-7 h-7 text-text-tertiary" />
              </div>
              <OuiStack gap="xs" align="center">
                <OuiText size="sm" weight="semibold">
                  {{ activeFilter === 'all' ? "You're all caught up!" : `No ${activeFilter} notifications` }}
                </OuiText>
                <OuiText size="sm" color="tertiary" class="max-w-[18rem] leading-relaxed">
                  {{ activeFilter === 'all'
                    ? "We'll notify you when something important happens."
                    : `No ${activeFilter} notifications at the moment.` }}
                </OuiText>
              </OuiStack>
            </div>

            <!-- Grouped list -->
            <div v-else>
              <template v-for="group in groupedNotifications" :key="group.date">

                <!-- Sticky date header -->
                <div class="sticky top-0 z-10 px-4 py-1.5 bg-surface-overlay/95 backdrop-blur-sm border-b border-border-muted">
                  <OuiText size="xs" weight="semibold" color="tertiary" class="uppercase tracking-widest">
                    {{ group.date }}
                  </OuiText>
                </div>

                <!-- Rows -->
                <div class="divide-y divide-border-muted">
                  <div
                    v-for="n in group.items"
                    :key="n.id"
                    class="group relative flex items-start gap-3 px-4 py-3.5 cursor-pointer transition-colors border-l-[3px] hover:bg-surface-muted"
                    :class="n.read ? 'opacity-50 hover:opacity-100' : ''"
                    :style="{ borderLeftColor: n.read ? 'transparent' : getAccentColor(n) }"
                    @click="handleNotificationClick(n)"
                  >
                    <!-- Icon circle -->
                    <div
                      class="shrink-0 mt-0.5 w-8 h-8 rounded-full flex items-center justify-center"
                      :class="getIconClasses(n)"
                    >
                      <component :is="getNotificationIcon(n)" class="w-4 h-4" />
                    </div>

                    <!-- Text content -->
                    <div class="flex-1 min-w-0 pr-8">
                      <OuiText
                        size="sm"
                        :weight="n.read ? 'normal' : 'semibold'"
                        class="leading-snug line-clamp-1"
                      >{{ n.title }}</OuiText>
                      <OuiText size="sm" color="tertiary" class="mt-0.5 line-clamp-2 leading-relaxed">
                        {{ n.message }}
                      </OuiText>
                      <div class="flex items-center gap-2 mt-1.5 flex-wrap">
                        <OuiText size="xs" color="tertiary">
                          <OuiRelativeTime :value="n.timestamp" />
                        </OuiText>
                        <span
                          v-if="n.actionUrl && n.actionLabel"
                          class="text-xs font-medium text-primary hover:underline cursor-pointer"
                          @click.stop="handleActionClick(n)"
                        >{{ n.actionLabel }} →</span>
                      </div>
                    </div>

                    <!-- Hover dismiss -->
                    <button
                      class="absolute right-3 top-3.5 flex items-center justify-center w-6 h-6 rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-surface-base"
                      :aria-label="`Dismiss: ${n.title}`"
                      @click.stop="remove(n.id)"
                    >
                      <XMarkIcon class="w-3.5 h-3.5 text-text-tertiary" />
                    </button>
                  </div>
                </div>

              </template>
            </div>
          </div>

          <!-- Footer -->
          <div class="flex items-center justify-between gap-3 px-4 py-2.5 border-t border-border-muted shrink-0">
            <button
              v-if="items.length > 0"
              class="text-sm font-medium text-text-tertiary hover:text-danger transition-colors disabled:opacity-40"
              :disabled="isClearingAll"
              @click="clearAll"
            >{{ isClearingAll ? 'Clearing…' : 'Clear all' }}</button>
            <div v-else />
            <NuxtLink
              to="/settings?tab=notifications"
              class="text-sm font-medium text-text-tertiary hover:text-primary transition-colors"
              @click="handleClose"
            >Settings →</NuxtLink>
          </div>

        </div>
      </Transition>

    </Teleport>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
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
const activeFilter = ref<"all" | "unread" | "critical">("all");
const isMarkingAllRead = ref(false);
const isClearingAll = ref(false);

// Filter definitions
const filters = computed(() => [
  { key: "all" as const, label: "All", count: props.items.length },
  { key: "unread" as const, label: "Unread", count: unreadCount.value },
  { key: "critical" as const, label: "Critical", count: criticalCount.value },
]);

// Filtered items
const filteredItems = computed(() => {
  let items = props.items;

  switch (activeFilter.value) {
    case "unread":
      items = items.filter((n) => !n.read);
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
const criticalCount = computed(() =>
  props.items.filter((n) => n.severity?.toUpperCase() === "CRITICAL" && !n.read).length
);

const handleClose = () => {
  emit("update:modelValue", false);
  emit("close");
};

const handleNotificationClick = (notification: NotificationItem) => {
  // Mark as read on click
  if (!notification.read) {
    markRead(notification.id);
  }
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

const getAccentColor = (n: NotificationItem): string => {
  const severity = n.severity?.toUpperCase();
  if (severity === "CRITICAL") return "var(--color-danger)";
  if (severity === "HIGH") return "var(--color-warning)";
  if (severity === "MEDIUM") return "var(--color-info, var(--color-primary))";
  const type = n.type?.toUpperCase() || "INFO";
  switch (type) {
    case "ERROR": return "var(--color-danger)";
    case "WARNING":
    case "QUOTA": return "var(--color-warning)";
    case "SUCCESS": return "var(--color-success)";
    case "DEPLOYMENT": return "var(--color-primary)";
    case "BILLING": return "var(--color-accent, var(--color-primary))";
    case "INVITE": return "var(--color-info, var(--color-primary))";
    default: return "var(--color-primary)";
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
    return "bg-surface-muted text-text-tertiary";
  }
  const type = notification.type?.toUpperCase() || "INFO";
  const visual = notificationVisuals[type] ?? notificationVisuals.INFO!;
  return `${visual.iconBg} ${visual.iconColor}`;
};

// Calculate default position on the far right
// (no longer needed — panel uses CSS fixed positioning)

function markRead(id: string) {
  emit(
    "update:items",
    props.items.map((n) => (n.id === id ? { ...n, read: true } : n))
  );
}

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
