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
        <OuiText size="sm" weight="medium" color="danger">{{
          displayTitle
        }}</OuiText>
        <OuiStack gap="xs">
          <OuiText size="sm" color="danger" class="opacity-80">{{
            errorMessage
          }}</OuiText>
          <OuiText v-if="hint" size="sm" color="danger" class="opacity-80">{{
            hint
          }}</OuiText>
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

  const isServiceUnavailable = computed(() => {
    if (!props.error) return false;

    if (props.error instanceof ConnectError) {
      return (
        props.error.code === Code.Unavailable &&
        (props.error.message?.includes("All service replicas are unhealthy") ||
          props.error.message?.includes("service_unavailable"))
      );
    }

    const errorMsg = props.error.message || "";
    return (
      errorMsg.includes("service_unavailable") ||
      errorMsg.includes("All service replicas are unhealthy")
    );
  });

  const displayTitle = computed(() => {
    if (props.title) return props.title;
    if (isServiceUnavailable.value) return "Service Temporarily Unavailable";
    return "Error";
  });

  const errorMessage = computed(() => {
    if (!props.error) return "";

    // Handle ConnectRPC errors
    if (props.error instanceof ConnectError) {
      if (props.error.code === Code.PermissionDenied) {
        if (props.error.message && props.error.message.trim() !== "") {
          return props.error.message;
        }
        return "You don't have permission to perform this action";
      }

      // Check for auth errors
      if (props.error.code === Code.Unauthenticated) {
        return "Authentication required. Please log in and try again.";
      }

      // Check for service unavailable errors (503)
      if (props.error.code === Code.Unavailable) {
        // Check if it's the specific "service_unavailable" error from API gateway
        if (
          props.error.message?.includes("All service replicas are unhealthy") ||
          props.error.message?.includes("service_unavailable")
        ) {
          return "The service is temporarily unavailable. All service replicas are currently unhealthy. Please try again in a few moments.";
        }
        return (
          props.error.message ||
          "The service is temporarily unavailable. Please try again later."
        );
      }

      // Return the error message
      return props.error.message || "An error occurred";
    }

    // Handle regular errors - check for service_unavailable in error message
    const errorMsg = props.error.message || "";
    if (
      errorMsg.includes("service_unavailable") ||
      errorMsg.includes("All service replicas are unhealthy")
    ) {
      return "The service is temporarily unavailable. All service replicas are currently unhealthy. Please try again in a few moments.";
    }

    return errorMsg || "An unexpected error occurred";
  });
</script>
