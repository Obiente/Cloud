<template>
  <!--
    Icon + section title for use inside card bodies.

    <UiSectionHeader :icon="ServerIcon" color="primary">Details</UiSectionHeader>
  -->
  <OuiFlex align="center" gap="xs">
    <component
      :is="icon"
      :class="['shrink-0', sizeClass, colorClass]"
    />
    <OuiText size="sm" weight="semibold">
      <slot />
    </OuiText>
  </OuiFlex>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Component } from "vue";

const props = withDefaults(defineProps<{
  icon: Component;
  /** Icon accent color */
  color?: "primary" | "success" | "warning" | "danger" | "info" | "secondary" | "default";
  /** Icon size */
  size?: "sm" | "md";
}>(), {
  color: "primary",
  size: "sm",
});

const sizeClass = computed(() =>
  props.size === "sm" ? "h-3.5 w-3.5" : "h-4 w-4"
);

const colorClass = computed(() => {
  const map: Record<string, string> = {
    primary: "text-accent-primary",
    success: "text-success",
    warning: "text-warning",
    danger: "text-danger",
    info: "text-accent-info",
    secondary: "text-accent-secondary",
    default: "text-secondary",
  };
  return map[props.color] ?? "text-secondary";
});
</script>
