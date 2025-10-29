<template>
  <div v-if="error" class="bg-danger/10 border border-danger/30 rounded-lg px-4 py-3 my-4">
    <div class="flex items-start">
      <div class="flex-shrink-0">
        <ExclamationTriangleIcon class="h-5 w-5 text-danger" aria-hidden="true" />
      </div>
      <div class="ml-3 flex-1">
        <h3 class="text-sm font-medium text-danger">{{ title || 'Error' }}</h3>
        <div class="mt-1 text-sm text-danger/80">
          <p>{{ errorMessage }}</p>
          <div v-if="hint" class="mt-2 text-sm">
            <p>{{ hint }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ExclamationTriangleIcon } from '@heroicons/vue/20/solid';
import { ConnectError, Code } from '@connectrpc/connect';

const props = defineProps<{
  error: Error | ConnectError | null | undefined;
  title?: string;
  hint?: string;
}>();

const errorMessage = computed(() => {
  if (!props.error) return '';
  
  // Handle ConnectRPC errors
  if (props.error instanceof ConnectError) {
    // Check for permission errors
    if (props.error.code === Code.PermissionDenied) {
      return 'You don\'t have permission to perform this action';
    }
    
    // Check for auth errors
    if (props.error.code === Code.Unauthenticated) {
      return 'Authentication required. Please log in and try again.';
    }
    
    // Return the error message
    return props.error.message || 'An error occurred';
  }
  
  // Handle regular errors
  return props.error.message || 'An unexpected error occurred';
});
</script>
