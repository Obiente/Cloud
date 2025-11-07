<template>
  <ClientOnly>
  <OuiFloatingPanel
    v-model="open"
    title="Notifications"
    :description="description"
      :default-position="clientPosition"
    :persist-rect="true"
    content-class="max-w-[560px]"
    @close="handleClose"
  >
    <div class="w-full">
      <OuiFlex justify="end" align="center" class="mb-3">
        <OuiFlex gap="sm">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="markAllRead"
            :disabled="unreadCount === 0"
            >Mark all read</OuiButton
          >
          <OuiButton
            variant="ghost"
            size="sm"
            color="danger"
            @click="clearAll"
            :disabled="items.length === 0"
            >Clear</OuiButton
          >
        </OuiFlex>
      </OuiFlex>

      <div v-if="items.length === 0" class="py-8 text-center">
        <OuiText color="secondary">You're all caught up.</OuiText>
      </div>

      <OuiStack v-else gap="sm">
        <OuiCard
          v-for="n in items"
          :key="n.id"
          variant="overlay"
          class="ring-1 ring-border-muted hover:ring-border-default transition cursor-pointer"
          :class="n.read ? 'opacity-75' : ''"
          @click="handleNotificationClick(n)"
        >
          <OuiCardBody>
            <OuiFlex justify="between" align="start" gap="md">
              <OuiStack gap="xs" class="min-w-0">
                <OuiText
                  size="sm"
                  weight="medium"
                  color="primary"
                  truncate
                  >{{ n.title }}</OuiText
                >
                <OuiText
                  size="xs"
                  color="secondary"
                  class="wrap-break-word"
                  >{{ n.message }}</OuiText
                >
                <OuiText size="xs" color="secondary">
                  <OuiRelativeTime :value="n.timestamp" :style="'short'" />
                </OuiText>
              </OuiStack>
              <OuiFlex gap="xs">
                <OuiButton
                  variant="ghost"
                  size="xs"
                  @click.stop="toggleRead(n.id)"
                  >{{ n.read ? "Unread" : "Read" }}</OuiButton
                >
                <OuiButton
                  variant="ghost"
                  size="xs"
                  color="danger"
                  @click.stop="remove(n.id)"
                  >Dismiss</OuiButton
                >
              </OuiFlex>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>
    </div>
  </OuiFloatingPanel>
  </ClientOnly>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch, nextTick } from "vue";
import OuiFloatingPanel from "~/components/oui/FloatingPanel.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

interface NotificationItem {
  id: string;
  title: string;
  message: string;
  timestamp: Date;
  read?: boolean;
}

const props = defineProps<{
  modelValue: boolean;
  items: NotificationItem[];
  description?: string;
  anchorElement?: HTMLElement | null;
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

const handleClose = () => {
  emit("update:modelValue", false);
  emit("close");
};

const handleNotificationClick = (notification: NotificationItem) => {
  // If notification is about invites, navigate to invites page
  if (notification.title?.toLowerCase().includes("invitation") || notification.message?.toLowerCase().includes("invited")) {
    router.push("/invites");
    handleClose();
  }
};

// Calculate default position underneath the notification button
// Use a ref with safe SSR default, then update on client
const defaultPosition = ref<{ x: number; y: number }>({ x: 100, y: 80 });
const clientPosition = computed(() => {
  // Only use calculated position on client, otherwise use SSR-safe default
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
      // Position underneath the button, aligned to the right edge
      // Ensure panel doesn't go off-screen on the left
      const panelWidth = 560;
      const xPos = Math.max(16, rect.right - panelWidth); // At least 16px from left edge
      defaultPosition.value = {
        x: xPos,
        y: rect.bottom + 8, // 8px gap below button
      };
    } else if (window && window.innerWidth) {
      // Fallback to top-right if no anchor
      defaultPosition.value = { x: window.innerWidth - 580, y: 80 };
    }
  } catch (e) {
    // Ignore errors during SSR/hydration
    console.debug("Could not set notification panel position:", e);
  }
};

// Register lifecycle hooks unconditionally (required by Vue)
onMounted(() => {
if (import.meta.client) {
    updatePosition();
  }
  });
  
  // Watch for anchor element changes
  watch(() => props.anchorElement, () => {
  if (import.meta.client) {
    updatePosition();
  }
  }, { immediate: true });
  
  // Also update when panel opens
  watch(() => props.modelValue, (isOpen) => {
  if (isOpen && import.meta.client) {
      // Small delay to ensure DOM is ready
      nextTick(() => {
        updatePosition();
      });
    }
  });

const unreadCount = computed(() => props.items.filter((n) => !n.read).length);

function markAllRead() {
  emit(
    "update:items",
    props.items.map((n) => ({ ...n, read: true }))
  );
}

function clearAll() {
  emit("update:items", []);
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
