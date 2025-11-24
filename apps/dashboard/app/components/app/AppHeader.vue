<template>
  <OuiContainer as="header" class="bg-surface-base outline-border-muted px-3 md:px-6 py-3 md:py-4 min-w-0 overflow-hidden">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="sm" class="min-w-0">
      <!-- Page title -->
      <OuiFlex align="center" gap="sm" class="flex-1 min-w-0 md:gap-4 overflow-hidden">
        <slot name="leading" />

        <OuiStack gap="xs" class="min-w-0 flex-1">
          <OuiText as="h1" size="xl" weight="bold" color="primary" class="truncate md:text-2xl">
            <slot name="title">
              {{ title }}
            </slot>
          </OuiText>
          <OuiText v-if="subtitle" size="xs" color="secondary" class="hidden sm:block md:text-sm">
            {{ subtitle }}
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Actions -->
      <OuiFlex align="center" gap="sm" class="shrink-0 md:gap-4">
        <!-- Theme Switcher -->
        <OuiThemeSwitcher />

        <!-- Notifications -->
        <OuiButton
          ref="notificationButtonRef"
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
import { ref } from "vue";
import { BellIcon } from "@heroicons/vue/24/outline";
import OuiText from "../oui/Text.vue";
import OuiThemeSwitcher from "../oui/ThemeSwitcher.vue";

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

const notificationButtonRef = ref<HTMLElement | null>(null);

// Expose the button ref for parent components
defineExpose({
  notificationButtonRef,
});

const handleNotificationsClick = () => {
  emit("notifications-click");
};
</script>
