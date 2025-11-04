<template>
  <OuiContainer
    class="absolute bottom-0 left-0 right-0 p-6 border-t border-border-muted bg-surface-base"
  >
    <OuiFlex align="center" gap="md">
      <!-- Avatar -->
      <OuiBox class="relative">
        <OuiBox class="w-8 h-8 bg-surface-muted rounded-full">
          <OuiFlex align="center" justify="center" class="h-full">
            <UserIcon class="w-5 h-5 text-text-secondary" />
          </OuiFlex>
        </OuiBox>
        <!-- Online status indicator -->
        <OuiBox
          class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-success border-2 border-surface-base rounded-full"
        />
      </OuiBox>

      <!-- User info -->
      <OuiStack gap="none" class="flex-1 min-w-0">
        <OuiText size="sm" weight="medium" color="primary" truncate>
          {{ auth.user?.name || auth.user?.preferred_username }}
        </OuiText>
        <OuiText size="xs" color="secondary" truncate>
          {{
            auth.user?.name
              ? `@${auth.user.preferred_username}`
              : auth.user?.email
          }}
        </OuiText>
      </OuiStack>

      <!-- Logout button -->
      <OuiButton
        variant="ghost"
        size="sm"
        @click="auth.logout()"
        title="Logout"
        class="!p-1.5 hover:bg-danger/10 hover:text-danger"
      >
        <ArrowRightOnRectangleIcon class="w-4 h-4" />
      </OuiButton>
    </OuiFlex>
  </OuiContainer>
</template>

<script setup lang="ts">
  import {
    UserIcon,
    ArrowRightOnRectangleIcon,
  } from "@heroicons/vue/24/outline";
  import OuiText from "../oui/Text.vue";
  const auth = useAuth();
</script>
