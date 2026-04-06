<template>
  <!--
    The "quick info bar" card shown at the top of overview tabs.
    Left side: icon + primary identifier text + optional secondary label.
    Right side: default slot for badges/actions.

    <UiQuickInfoBar :icon="ServerIcon" primary="192.168.1.1" secondary="Ubuntu 22 · EU West">
      <OuiBadge variant="secondary" size="xs">4 vCPU</OuiBadge>
      <OuiBadge variant="secondary" size="xs">8 GB RAM</OuiBadge>
    </UiQuickInfoBar>
  -->
  <OuiCard variant="outline">
    <OuiCardBody>
      <OuiFlex align="center" justify="between" gap="md">
        <!-- Left: icon + text -->
        <OuiFlex align="center" gap="md" class="min-w-0">
          <UiIconContainer size="md">
            <component :is="icon" :class="['h-4 w-4', iconColorClass]" />
          </UiIconContainer>
          <OuiStack gap="xs" class="min-w-0">
            <OuiText
              size="sm"
              weight="semibold"
              :class="mono ? 'font-mono' : ''"
              class="truncate"
            >
              {{ primary }}
            </OuiText>
            <OuiText v-if="secondary" size="xs" color="tertiary" class="truncate">
              {{ secondary }}
            </OuiText>
          </OuiStack>
        </OuiFlex>

        <!-- Right: badges slot -->
        <OuiFlex v-if="$slots.default" align="center" gap="xs" class="shrink-0 flex-wrap justify-end">
          <slot />
        </OuiFlex>
      </OuiFlex>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Component } from "vue";

const props = withDefaults(defineProps<{
  /** Heroicon component for the icon container */
  icon: Component;
  /** Primary identifier (IP address, domain, name…) */
  primary: string;
  /** Secondary descriptor shown below primary */
  secondary?: string;
  /** Use font-mono for primary text */
  mono?: boolean;
  /** Icon color */
  color?: "primary" | "success" | "warning" | "danger" | "info" | "secondary" | "default";
}>(), {
  mono: false,
  color: "default",
});

const colorMap: Record<string, string> = {
  primary: "text-accent-primary",
  success: "text-success",
  warning: "text-warning",
  danger: "text-danger",
  info: "text-accent-info",
  secondary: "text-accent-secondary",
  default: "text-secondary",
};

const iconColorClass = computed(() => colorMap[props.color] ?? "text-secondary");
</script>
