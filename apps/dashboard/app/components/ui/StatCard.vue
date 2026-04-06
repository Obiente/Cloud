<template>
  <!--
    A metric/stat card with an icon, label, prominent value, optional mini progress bar,
    and optional subtitle. Built on OuiCard variant="outline".

    <UiStatCard label="CPU" :icon="CpuChipIcon" color="primary" value="4 cores" :bar="65" />
    <UiStatCard label="Memory" :icon="CircleStackIcon" color="info" value="8 GB" />
    <UiStatCard label="CPU" :icon="CpuChipIcon" :streaming="isStreaming" :bar="cpu" :bar-class="cpuColor" />
  -->
  <OuiCard variant="outline">
    <OuiCardBody>
      <OuiStack gap="sm">
        <!-- Label row (icon + label, optional streaming dot) -->
        <OuiFlex align="center" justify="between">
          <OuiFlex align="center" gap="xs">
            <component :is="icon" :class="['shrink-0 h-3.5 w-3.5', iconColorClass]" />
            <OuiText size="xs" color="tertiary">{{ label }}</OuiText>
          </OuiFlex>
          <span
            v-if="streaming"
            class="h-1.5 w-1.5 rounded-full bg-success animate-pulse shrink-0"
          />
        </OuiFlex>

        <!-- Value -->
        <OuiText :size="valueSize" weight="semibold">
          <slot>{{ value }}</slot>
        </OuiText>

        <!-- Mini bar -->
        <div v-if="bar !== undefined" class="h-1 rounded-full bg-surface-muted overflow-hidden">
          <div
            class="h-full rounded-full transition-all"
            :class="barClass ?? barColorClass"
            :style="{ width: `${Math.min(100, Math.max(0, bar))}%` }"
          />
        </div>

        <!-- Subtitle -->
        <OuiText v-if="subtitle" size="xs" color="tertiary">{{ subtitle }}</OuiText>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Component } from "vue";

const props = withDefaults(defineProps<{
  /** The label shown above the value */
  label: string;
  /** Heroicon component */
  icon: Component;
  /** Icon/bar accent color */
  color?: "primary" | "success" | "warning" | "danger" | "info" | "secondary";
  /** The value to display (use slot for complex values like OuiByte) */
  value?: string | number;
  /** 0–100 percentage for mini bar. Omit to hide bar. */
  bar?: number;
  /** Direct CSS class override for bar color (e.g. for dynamic threshold colors) */
  barClass?: string;
  /** Show the animated pulse streaming indicator dot */
  streaming?: boolean;
  /** Value text size */
  valueSize?: "sm" | "md" | "lg" | "xl" | "2xl";
  /** Optional extra line below the value/bar */
  subtitle?: string;
}>(), {
  color: "primary",
  valueSize: "xl",
  streaming: false,
});

const colorMap: Record<string, { icon: string; bar: string }> = {
  primary: { icon: "text-accent-primary", bar: "bg-accent-primary/60" },
  success: { icon: "text-success", bar: "bg-success/60" },
  warning: { icon: "text-warning", bar: "bg-warning/60" },
  danger: { icon: "text-danger", bar: "bg-danger/60" },
  info: { icon: "text-accent-info", bar: "bg-accent-info/60" },
  secondary: { icon: "text-accent-secondary", bar: "bg-accent-secondary/60" },
};

const iconColorClass = computed(() => colorMap[props.color]?.icon ?? "text-secondary");
const barColorClass = computed(() => colorMap[props.color]?.bar ?? "bg-surface-muted");
</script>

