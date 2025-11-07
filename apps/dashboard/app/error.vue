<template>
  <OuiBox
    bg="surface-base"
    class="min-h-screen flex items-center justify-center"
    p="md"
  >
    <OuiContainer size="2xl" class="text-center">
      <OuiStack gap="2xl" align="center">
        <!-- Error Icon -->
        <OuiBox position="relative">
          <OuiBox
            position="absolute"
            class="inset-0 bg-danger/20 blur-3xl rounded-full"
          />
          <ExclamationTriangleIcon
            class="relative h-24 w-24 text-danger"
            :class="{ 'animate-pulse': error.statusCode === 500 }"
          />
        </OuiBox>

        <!-- Error Content -->
        <OuiStack gap="lg" align="center" class="max-w-xl">
          <OuiStack gap="sm" align="center">
            <OuiText as="h1" size="6xl" weight="bold" color="primary">
              {{ error.statusCode || "Error" }}
            </OuiText>
            <OuiText size="xl" color="secondary" weight="medium">
              {{ errorTitle }}
            </OuiText>
          </OuiStack>

          <OuiText size="md" color="muted" class="max-w-md">
            {{ errorMessage }}
          </OuiText>
        </OuiStack>

        <!-- Error Details (for development) -->
        <OuiCard
          v-if="error.message && isDevelopment"
          variant="outline"
          class="max-w-2xl w-full text-left"
        >
          <OuiCardHeader>
            <OuiFlex align="center" gap="sm">
              <CodeBracketIcon class="h-5 w-5 text-muted" />
              <OuiText size="sm" weight="semibold" color="muted">
                Error Details
              </OuiText>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiBox
              bg="surface-muted"
              rounded="md"
              p="md"
              class="font-mono text-xs overflow-x-auto"
            >
              <OuiText size="xs" color="danger">{{ error.message }}</OuiText>
              <OuiText
                v-if="error.stack"
                size="xs"
                color="muted"
                class="mt-2 whitespace-pre-wrap"
              >
                {{ error.stack }}
              </OuiText>
            </OuiBox>
          </OuiCardBody>
        </OuiCard>

        <!-- Actions -->
        <OuiFlex gap="md" wrap="wrap" justify="center">
          <OuiButton size="lg" @click="handleError">
            <ArrowPathIcon class="h-5 w-5 mr-2" />
            Try Again
          </OuiButton>
          <OuiButton variant="outline" size="lg" @click="goHome">
            <HomeIcon class="h-5 w-5 mr-2" />
            Go Home
          </OuiButton>
          <OuiButton
            v-if="error.statusCode === 500"
            variant="ghost"
            size="lg"
            @click="goBack"
          >
            <ArrowLeftIcon class="h-5 w-5 mr-2" />
            Go Back
          </OuiButton>
        </OuiFlex>

        <!-- Support Link -->
        <OuiText size="sm" color="muted">
          Need help?
          <NuxtLink
            to="/support"
            class="text-primary hover:text-accent-primary underline"
          >
            Contact Support
          </NuxtLink>
        </OuiText>
      </OuiStack>
    </OuiContainer>
  </OuiBox>
</template>

<script setup lang="ts">
import {
  ExclamationTriangleIcon,
  ArrowPathIcon,
  HomeIcon,
  ArrowLeftIcon,
  CodeBracketIcon,
} from "@heroicons/vue/24/outline";

interface Props {
  error: {
    statusCode?: number;
    statusMessage?: string;
    message?: string;
    stack?: string;
  };
}

const props = defineProps<Props>();

const isDevelopment = computed(() => import.meta.dev);

const errorTitle = computed(() => {
  const statusCode = props.error.statusCode;
  switch (statusCode) {
    case 404:
      return "Page Not Found";
    case 403:
      return "Access Forbidden";
    case 500:
      return "Internal Server Error";
    case 503:
      return "Service Unavailable";
    default:
      return props.error.statusMessage || "Something went wrong";
  }
});

const errorMessage = computed(() => {
  const statusCode = props.error.statusCode;
  switch (statusCode) {
    case 404:
      return "The page you're looking for doesn't exist or has been moved.";
    case 403:
      return "You don't have permission to access this resource.";
    case 500:
      return "We're experiencing technical difficulties. Our team has been notified and is working on a fix.";
    case 503:
      return "The service is temporarily unavailable. Please try again in a few moments.";
    default:
      return (
        props.error.message ||
        "An unexpected error occurred. Please try again later."
      );
  }
});

const handleError = () => {
  clearError({ redirect: useRoute().fullPath });
};

const goHome = () => {
  navigateTo("/");
};

const goBack = () => {
  if (import.meta.client && window.history.length > 1) {
    window.history.back();
  } else {
    navigateTo("/");
  }
};

// Set page meta
useHead({
  title: `${props.error.statusCode || "Error"} - ${errorTitle.value}`,
});
</script>

