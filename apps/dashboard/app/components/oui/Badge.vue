<template>
  <component
    :is="as"
    :class="badgeClasses"
    :style="variantStyles"
    :data-variant="normalizedVariant"
    :data-tone="tone"
    :data-size="size"
    v-bind="$attrs"
  >
    <component
      v-if="showIcon && iconPosition === 'start'"
      :is="icon"
      class="oui-badge__icon"
      aria-hidden="true"
    />
    <slot />
    <component
      v-if="showIcon && iconPosition === 'end'"
      :is="icon"
      class="oui-badge__icon"
      aria-hidden="true"
    />
  </component>
</template>

<script setup lang="ts">
import { Comment, computed, useSlots } from "vue";
import type { Component } from "vue";

type BadgeVariant =
  | "primary"
  | "secondary"
  | "success"
  | "warning"
  | "danger"
  | "outline";

type BadgeTone = "soft" | "solid" | "outline";

type BadgeSize = "xs" | "sm" | "md";

interface BadgeProps {
  /**
   * Element or component to render
   * @default 'span'
   */
  as?: string;

  /**
   * Visual variant token
   * @default 'secondary'
   */
  variant?: BadgeVariant;

  /**
   * Tonal treatment for the badge surface
   * @default 'soft'
   */
  tone?: BadgeTone;

  /**
   * Sizing scale for padding and typography
   * @default 'sm'
   */
  size?: BadgeSize;

  /**
   * Render an icon component within the badge
   */
  icon?: Component | null;

  /**
   * Choose which edge the icon should render on
   * @default 'start'
   */
  iconPosition?: "start" | "end";

  /**
   * Control rounded treatment
   * @default true
   */
  pill?: boolean;

  /**
   * Enable uppercase typography helper
   * @default false
   */
  uppercase?: boolean;

  /**
   * Apply hover and focus affordances
   * @default false
   */
  interactive?: boolean;
}

const props = withDefaults(defineProps<BadgeProps>(), {
  as: "span",
  variant: "secondary",
  tone: "soft",
  size: "sm",
  icon: null,
  iconPosition: "start",
  pill: true,
  uppercase: false,
  interactive: false,
});

const slots = useSlots();

const normalizedVariant = computed<BadgeVariant>(() => props.variant ?? "secondary");

const badgeClasses = computed(() => {
  const classes = [
    "oui-badge",
    `oui-badge-${normalizedVariant.value}`,
    `oui-badge-size-${props.size}`,
    `oui-badge-tone-${props.tone}`,
    props.pill ? "oui-badge-shape-pill" : "oui-badge-shape-rounded",
  ];

  if (props.uppercase) {
    classes.push("oui-badge-uppercase");
  }

  if (props.interactive) {
    classes.push("oui-badge-interactive");
  }

  if (showIcon.value && !hasContent.value) {
    classes.push("oui-badge-icon-only");
  }

  return classes;
});

const showIcon = computed(() => Boolean(props.icon));

const hasContent = computed(() => {
  const slot = slots.default?.();
  if (!slot) {
    return false;
  }

  return slot.some((node) => {
    if (node.type === Comment) {
      return false;
    }

    if (typeof node.children === "string") {
      return node.children.trim().length > 0;
    }

    if (Array.isArray(node.children)) {
      return node.children.some((child) => {
        if (typeof child === "string") {
          return child.trim().length > 0;
        }
        if (typeof child === "object" && child !== null && "children" in child) {
          const value = (child as { children?: unknown }).children;
          return typeof value === "string" ? value.trim().length > 0 : true;
        }
        return Boolean(child);
      });
    }

    return true;
  });
});

const variantStyleMap: Record<BadgeVariant, Record<string, string>> = {
  primary: {
    "--oui-badge-accent": "var(--oui-accent-primary)",
    "--oui-badge-soft-surface": "color-mix(in srgb, var(--oui-accent-primary) 12%, transparent)",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-accent-primary) 24%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-accent-primary) 40%, transparent)",
    "--oui-badge-on-solid": "var(--oui-foreground)",
  },
  secondary: {
    "--oui-badge-accent": "var(--oui-text-secondary)",
    "--oui-badge-soft-surface": "var(--oui-surface-muted)",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-text-secondary) 20%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-text-secondary) 35%, transparent)",
    "--oui-badge-on-solid": "var(--oui-background)",
  },
  success: {
    "--oui-badge-accent": "var(--oui-accent-success)",
    "--oui-badge-soft-surface": "color-mix(in srgb, var(--oui-accent-success) 12%, transparent)",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-accent-success) 24%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-accent-success) 40%, transparent)",
    "--oui-badge-on-solid": "var(--oui-foreground)",
  },
  warning: {
    "--oui-badge-accent": "var(--oui-accent-warning)",
    "--oui-badge-soft-surface": "color-mix(in srgb, var(--oui-accent-warning) 14%, transparent)",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-accent-warning) 28%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-accent-warning) 45%, transparent)",
    "--oui-badge-on-solid": "var(--oui-background)",
  },
  danger: {
    "--oui-badge-accent": "var(--oui-accent-danger)",
    "--oui-badge-soft-surface": "color-mix(in srgb, var(--oui-accent-danger) 14%, transparent)",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-accent-danger) 28%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-accent-danger) 45%, transparent)",
    "--oui-badge-on-solid": "var(--oui-foreground)",
  },
  outline: {
    "--oui-badge-accent": "var(--oui-text-secondary)",
    "--oui-badge-soft-surface": "transparent",
    "--oui-badge-soft-border": "color-mix(in srgb, var(--oui-text-secondary) 25%, transparent)",
    "--oui-badge-outline-border": "color-mix(in srgb, var(--oui-text-secondary) 40%, transparent)",
    "--oui-badge-on-solid": "var(--oui-foreground)",
  },
};

const variantStyles = computed(() => {
  return variantStyleMap[normalizedVariant.value] ?? variantStyleMap.secondary;
});

defineOptions({
  inheritAttrs: false,
});
</script>
