<template>
  <svg
    :class="spinnerClasses"
    :width="sizeValue"
    :height="sizeValue"
    viewBox="0 0 24 24"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    aria-label="Loading"
    role="status"
    aria-live="polite"
  >
    <circle
      class="opacity-25"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      stroke-width="4"
    />
    <path
      class="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
    />
  </svg>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { SizeRange } from "./types";

export type OUISpinnerSize = SizeRange<"xs", "xl">;

interface Props {
  /**
   * Size of the spinner
   * @default 'md'
   */
  size?: OUISpinnerSize;

  /**
   * Color of the spinner
   * @default 'primary'
   */
  color?: "primary" | "secondary" | "muted" | "success" | "warning" | "danger";
}

const props = withDefaults(defineProps<Props>(), {
  size: "md",
  color: "primary",
});

const sizeValue = computed(() => {
  const sizeMap: Record<OUISpinnerSize, string> = {
    xs: "1rem", // 16px
    sm: "1.25rem", // 20px
    md: "1.5rem", // 24px
    lg: "2rem", // 32px
    xl: "3rem", // 48px
  };
  return sizeMap[props.size];
});

const spinnerClasses = computed(() => {
  const classes = ["oui-spinner", "animate-spin"];

  // Color classes
  const colorClasses = {
    primary: "text-primary",
    secondary: "text-secondary",
    muted: "text-muted",
    success: "text-success",
    warning: "text-warning",
    danger: "text-danger",
  };
  classes.push(colorClasses[props.color]);

  return classes.join(" ");
});
</script>

<style scoped>
.oui-spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>

