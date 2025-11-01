<template>
  <OuiContainer as="header" class="bg-surface-base outline-border-muted px-6 py-4">
    <OuiFlex align="center" justify="between">
      <!-- Page title -->
      <OuiFlex align="center" gap="md">
        <slot name="leading" />

        <OuiStack gap="xs">
          <OuiText as="h1" size="2xl" weight="bold" color="primary">
            <slot name="title">
              {{ title }}
            </slot>
          </OuiText>
          <OuiText v-if="subtitle" size="sm" color="secondary">
            {{ subtitle }}
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Actions -->
      <OuiFlex align="center" gap="md">
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
          <OuiBox
            v-if="notificationCount > 0"
            class="absolute -top-1 -right-1 w-5 h-5 bg-danger text-foreground text-xs font-medium rounded-full"
          >
            <OuiFlex align="center" justify="center" class="h-full">
              <OuiText size="xs" weight="medium" color="primary">
                {{ notificationCount > 99 ? "99+" : notificationCount }}
              </OuiText>
            </OuiFlex>
          </OuiBox>
        </OuiButton>

        <!-- Additional actions slot -->
        <slot name="actions" />
      </OuiFlex>
    </OuiFlex>
  </OuiContainer>
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
