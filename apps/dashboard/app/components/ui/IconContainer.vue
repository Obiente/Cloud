<template>
  <div :class="containerClass">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(defineProps<{
  /** Container size */
  size?: "sm" | "md" | "lg";
  /** Background color tint */
  color?: "default" | "primary" | "success" | "warning" | "danger" | "info";
  /** rounded-full instead of rounded-lg */
  round?: boolean;
}>(), {
  size: "md",
  color: "default",
  round: false,
});

const sizeClass: Record<string, string> = {
  sm: "h-6 w-6",
  md: "h-8 w-8",
  lg: "h-12 w-12",
};

const colorClass: Record<string, string> = {
  default: "bg-surface-muted",
  primary: "bg-accent-primary/10",
  success: "bg-success/10",
  warning: "bg-warning/10",
  danger: "bg-danger/10",
  info: "bg-accent-info/10",
};

const containerClass = computed(() => [
  "flex items-center justify-center shrink-0",
  sizeClass[props.size],
  colorClass[props.color],
  props.round ? "rounded-full" : "rounded-lg",
]);
</script>
