<template>
  <FloatingPanel.Root v-model:open="open" :draggable="false" :resizable="false">
    <Teleport to="body">
      <FloatingPanel.Positioner>
        <FloatingPanel.Content
          class="z-60"
          :style="panelStyle"
        >
          <FloatingPanel.DragTrigger>
            <FloatingPanel.Header>
              <FloatingPanel.Title>
                <OuiText as="h3" size="lg" weight="semibold">Notifications</OuiText>
                <template v-if="description">
                  <OuiText as="p" size="xs" color="secondary">{{ description }}</OuiText>
                </template>
              </FloatingPanel.Title>
              <FloatingPanel.Control>
                <FloatingPanel.CloseTrigger>
                  <OuiButton variant="ghost" size="xs" @click="emit('close')">Close</OuiButton>
                </FloatingPanel.CloseTrigger>
              </FloatingPanel.Control>
            </FloatingPanel.Header>
          </FloatingPanel.DragTrigger>

          <FloatingPanel.Body>
            <div class="min-w-[320px] max-w-[560px] p-3">
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
                  class="ring-1 ring-border-muted hover:ring-border-default transition"
                  :class="n.read ? 'opacity-75' : ''"
                >
                  <OuiCardBody>
                    <OuiFlex justify="between" align="start" gap="md">
                      <OuiStack gap="xs" class="min-w-0">
                        <OuiText size="sm" weight="medium" color="primary" truncate>{{
                          n.title
                        }}</OuiText>
                        <OuiText size="xs" color="secondary" class="break-words">{{
                          n.message
                        }}</OuiText>
                        <OuiText size="xs" color="secondary">{{
                          formatRelativeTime(n.timestamp)
                        }}</OuiText>
                      </OuiStack>
                      <OuiFlex gap="xs">
                        <OuiButton
                          variant="ghost"
                          size="xs"
                          @click="toggleRead(n.id)"
                          >{{ n.read ? 'Unread' : 'Read' }}</OuiButton
                        >
                        <OuiButton
                          variant="ghost"
                          size="xs"
                          color="danger"
                          @click="remove(n.id)"
                          >Dismiss</OuiButton
                        >
                      </OuiFlex>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
              </OuiStack>
            </div>
          </FloatingPanel.Body>

          <!-- Optional resize handles if needed later -->
          <!-- <FloatingPanel.ResizeTrigger axis="n" /> -->
        </FloatingPanel.Content>
      </FloatingPanel.Positioner>
    </Teleport>
  </FloatingPanel.Root>
</template>

<script setup lang="ts">
  import { computed, watch, type CSSProperties } from "vue";
  import { FloatingPanel } from "@ark-ui/vue/floating-panel";

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

  // Position panel at top-right of the viewport with some padding
  const panelStyle = computed<CSSProperties>(() => ({
    position: 'fixed',
    top: '80px',
    right: '20px',
    maxHeight: 'calc(100vh - 120px)',
    overflow: 'auto',
    borderRadius: '0.75rem',
    background: 'var(--color-surface-overlay)',
    boxShadow: '0 20px 50px rgba(0,0,0,0.5)',
    border: '1px solid var(--color-border-muted)'
  }));

  // Emit close event when panel closes to preserve existing API contract
  watch(open, (val, oldVal) => {
    if (!val && oldVal) emit('close')
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

  function formatRelativeTime(date: Date) {
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);
    if (diffSec < 60) return "just now";
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHour < 24) return `${diffHour}h ago`;
    if (diffDay < 7) return `${diffDay}d ago`;
    return new Intl.DateTimeFormat("en-US", {
      month: "short",
      day: "numeric",
    }).format(date);
  }
</script>
