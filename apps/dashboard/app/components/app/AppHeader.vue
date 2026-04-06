<template>
  <OuiContainer as="header" class="app-topbar px-4 py-2.5 min-w-0 overflow-hidden">
    <OuiFlex align="center" justify="between" gap="sm" class="min-w-0">
      <!-- Page title -->
      <OuiFlex align="center" gap="sm" class="flex-1 min-w-0 overflow-hidden">
        <slot name="leading" />

        <OuiStack gap="none" class="min-w-0 flex-1">
          <OuiText as="h1" size="sm" weight="semibold" color="primary" class="truncate">
            <slot name="title">
              {{ title }}
            </slot>
          </OuiText>
          <OuiText v-if="subtitle" size="xs" color="tertiary" class="hidden sm:block">
            {{ subtitle }}
          </OuiText>
        </OuiStack>
      </OuiFlex>

      <!-- Actions -->
      <OuiFlex align="center" gap="xs" class="app-topbar-actions shrink-0">
        <OuiThemeSwitcher />

        <OuiButton
          ref="notificationButtonRef"
          variant="ghost"
          size="sm"
          title="Notifications"
          class="p-2! relative"
          @click="handleNotificationsClick"
        >
          <BellIcon class="w-4.5 h-4.5" />
          <OuiText
            v-if="notificationCount > 0"
            as="span"
            size="3xs"
            weight="medium"
            color="white"
            class="absolute -top-0.5 -right-0.5 min-w-[1.125rem] h-[1.125rem] bg-danger rounded-full flex items-center justify-center px-1"
          >
            {{ notificationCount > 99 ? "99+" : notificationCount }}
          </OuiText>
        </OuiButton>

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
