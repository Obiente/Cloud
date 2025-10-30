<template>
  <header class="bg-surface-base outline-border-muted px-6 py-4">
    <div class="flex items-center justify-between">
      <!-- Page title -->
      <div>
        <div class="flex items-center gap-3">
          <slot name="leading" />

          <div>
            <OuiText as="h1" size="2xl" weight="bold" color="primary">
              <slot name="title">
                {{ title }}
              </slot>
            </OuiText>
            <OuiText v-if="subtitle" size="sm" color="secondary" class="mt-1">
              {{ subtitle }}
            </OuiText>
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center space-x-4">
        <!-- Notifications -->
        <OuiButton
          variant="ghost"
          size="sm"
          title="Notifications"
          class="!p-2 relative"
          @click="handleNotificationsClick"
        >
          <BellIcon class="w-5 h-5" />
          <!-- Notification badge -->
          <span
            v-if="notificationCount > 0"
            class="absolute -top-1 -right-1 w-5 h-5 bg-danger text-foreground text-xs font-medium rounded-full flex items-center justify-center"
          >
            {{ notificationCount > 99 ? "99+" : notificationCount }}
          </span>
        </OuiButton>

        <!-- Additional actions slot -->
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { BellIcon } from "@heroicons/vue/24/outline";
import OuiText from "../oui/Text.vue";

interface Props {
  title?: string;
  subtitle?: string;
  notificationCount?: number;
}

const props = withDefaults(defineProps<Props>(), {
  title: "Dashboard",
  notificationCount: 0,
});

const emit = defineEmits<{
  "notifications-click": [];
}>();

const handleNotificationsClick = () => {
  emit("notifications-click");
};
</script>
