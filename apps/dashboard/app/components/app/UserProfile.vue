<template>
  <OuiFlex align="center" gap="sm" class="px-2 py-2 rounded-lg">
    <!-- Avatar -->
    <OuiBox position="relative" :shrink="false">
      <OuiBox rounded="full" class="h-8 w-8 bg-surface-muted flex items-center justify-center">
        <UserIcon class="w-4 h-4 text-secondary" />
      </OuiBox>
      <OuiBox
        as="span"
        position="absolute"
        rounded="full"
        class="-bottom-0.5 -right-0.5 h-2.5 w-2.5 border-2 border-surface-base bg-success"
      />
    </OuiBox>

    <!-- User info -->
    <OuiStack gap="none" grow class="min-w-0">
      <OuiText size="sm" weight="medium" color="primary" truncate>
        {{ auth.user?.name || auth.user?.preferred_username }}
      </OuiText>
      <OuiText size="xs" color="tertiary" truncate>
        {{
          auth.user?.name
            ? `@${auth.user.preferred_username}`
            : auth.user?.email
        }}
      </OuiText>
    </OuiStack>

    <!-- Logout -->
    <OuiButton
      variant="ghost"
      size="sm"
      @click="auth.logout()"
      title="Logout"
      class="p-1.5! hover:text-danger"
    >
      <ArrowRightOnRectangleIcon class="w-4 h-4" />
    </OuiButton>
  </OuiFlex>
</template>

<script setup lang="ts">
  import {
    UserIcon,
    ArrowRightOnRectangleIcon,
  } from "@heroicons/vue/24/outline";
  import OuiText from "../oui/Text.vue";
  const auth = useAuth();
</script>
