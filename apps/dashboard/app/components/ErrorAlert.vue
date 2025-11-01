<template>
  <OuiContainer
    v-if="error"
    class="bg-danger/10 border border-danger/30 rounded-xl px-4 py-3 my-4"
  >
    <OuiFlex align="start" gap="md">
      <OuiBox class="flex-shrink-0">
        <ExclamationTriangleIcon
          class="h-5 w-5 text-danger"
          aria-hidden="true"
        />
      </OuiBox>
      <OuiStack gap="xs" class="flex-1">
        <OuiText size="sm" weight="medium" color="danger">{{ title || "Error" }}</OuiText>
        <OuiStack gap="xs">
          <OuiText size="sm" color="danger" class="opacity-80">{{ errorMessage }}</OuiText>
          <OuiText v-if="hint" size="sm" color="danger" class="opacity-80">{{ hint }}</OuiText>
        </OuiStack>
      </OuiStack>
    </OuiFlex>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ExclamationTriangleIcon } from "@heroicons/vue/20/solid";
import { ConnectError, Code } from "@connectrpc/connect";

const props = defineProps<{
  error: Error | ConnectError | null | undefined;
  title?: string;
  hint?: string;
}>();

const errorMessage = computed(() => {
  if (!props.error) return "";

  // Handle ConnectRPC errors
  if (props.error instanceof ConnectError) {
    // Check for permission errors
    if (props.error.code === Code.PermissionDenied) {
      return "You don't have permission to perform this action";
    }

    // Check for auth errors
    if (props.error.code === Code.Unauthenticated) {
      return "Authentication required. Please log in and try again.";
    }

    // Return the error message
    return props.error.message || "An error occurred";
  }

  // Handle regular errors
  return props.error.message || "An unexpected error occurred";
});
</script>
