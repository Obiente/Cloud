<template>
  <div class="absolute bottom-0 left-0 right-0 p-6 border-t border-border-muted bg-surface-base">
    <div class="flex items-center space-x-3">
      <!-- Avatar -->
      <div class="relative">
        <div class="w-8 h-8 bg-surface-muted rounded-full flex items-center justify-center">
          <UserIcon class="w-5 h-5 text-text-secondary" />
        </div>
        <!-- <img
          v-else
          :src="user.avatarUrl"
          :alt="`${user.name || 'User'} avatar`"
          class="w-8 h-8 rounded-full object-cover"
        /> -->
        <!-- Online status indicator -->
        <div
          class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-success border-2 border-surface-base rounded-full"
        ></div>
      </div>

      <!-- User info -->
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-text-primary truncate">
          {{ user?.userName || 'User' }}
        </p>
        <p class="text-xs text-text-secondary truncate">
          {{ user?.preferredLoginName || 'user@example.com' }}
        </p>
      </div>

      <!-- Logout button -->
      <OuiButton
        variant="ghost"
        size="sm"
        @click="handleLogout"
        title="Logout"
        class="!p-1.5 hover:bg-danger/10 hover:text-danger"
      >
        <ArrowRightOnRectangleIcon class="w-4 h-4" />
      </OuiButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { UserIcon, ArrowRightOnRectangleIcon } from '@heroicons/vue/24/outline';

const { user } = useUser();

const emit = defineEmits<{
  logout: [];
}>();

const handleLogout = () => {
  emit('logout');
};
</script>
