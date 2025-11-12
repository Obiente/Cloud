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
      class="oui-spinner-track"
      cx="12"
      cy="12"
      r="10"
      :stroke="trackColor"
      stroke-width="2"
      fill="none"
    />
    <circle
      class="oui-spinner-arc"
      cx="12"
      cy="12"
      r="10"
      :stroke="arcColor"
      stroke-width="2"
      fill="none"
      stroke-linecap="round"
      :stroke-dasharray="circumference"
      :stroke-dashoffset="circumference * 0.75"
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

// Calculate circumference for stroke-dasharray (2 * π * radius)
const circumference = computed(() => 2 * Math.PI * 10); // radius is 10

const spinnerClasses = computed(() => {
  return ["oui-spinner", `oui-spinner-${props.color}`];
});

// Color mappings using CSS custom properties for better theme integration
const trackColor = computed(() => {
  const colorMap = {
    primary: "var(--oui-accent-primary)",
    secondary: "var(--oui-accent-secondary)",
    muted: "var(--oui-text-tertiary)",
    success: "var(--oui-accent-success)",
    warning: "var(--oui-accent-warning)",
    danger: "var(--oui-accent-danger)",
  };
  return colorMap[props.color];
});

const arcColor = computed(() => {
  const colorMap = {
    primary: "var(--oui-accent-primary)",
    secondary: "var(--oui-accent-secondary)",
    muted: "var(--oui-text-secondary)",
    success: "var(--oui-accent-success)",
    warning: "var(--oui-accent-warning)",
    danger: "var(--oui-accent-danger)",
  };
  return colorMap[props.color];
});
</script>

<style scoped>
.oui-spinner {
  display: inline-block;
  vertical-align: middle;
}

.oui-spinner-track {
  opacity: 0.2;
}

.oui-spinner-arc {
  transform-origin: center;
  animation: oui-spinner-rotate 0.8s linear infinite;
}

@keyframes oui-spinner-rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Color-specific opacity adjustments for better visibility */
.oui-spinner-primary .oui-spinner-arc {
  opacity: 1;
}

.oui-spinner-secondary .oui-spinner-arc {
  opacity: 1;
}

.oui-spinner-muted .oui-spinner-arc {
  opacity: 0.6;
}

.oui-spinner-success .oui-spinner-arc {
  opacity: 1;
}

.oui-spinner-warning .oui-spinner-arc {
  opacity: 1;
}

.oui-spinner-danger .oui-spinner-arc {
  opacity: 1;
}
</style>

