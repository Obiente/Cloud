<template>
  <component
    :is="componentTag"
    :type="componentType"
    :class="buttonClasses"
    :data-variant="variantToken"
    :data-color="colorToken"
    :data-size="sizeToken"
    :data-loading="loading || undefined"
    :aria-busy="loading ? 'true' : undefined"
    :aria-disabled="ariaDisabled"
    :disabled="disabledAttr"
    v-bind="$attrs"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Component } from "vue";
import type { SizeRange } from "./types";

export type OUIButtonSize = SizeRange<"xs", "xl">;
export type OUIButtonTone =
  | "primary"
  | "secondary"
  | "success"
  | "warning"
  | "danger"
  | "neutral";
export type OUIButtonVariant = "solid" | "soft" | "outline" | "ghost";

interface Props {
  /**
   * Element or component to render
   * @default 'button'
   */
  as?: string | Component;

  /**
   * Color intent for the button
   * @default 'primary'
   */
  color?: OUIButtonTone;

  /**
   * Visual treatment variant
   * @default 'solid'
   */
  variant?: OUIButtonVariant;

  /**
   * Sizing scale for spacing and typography
   * @default 'md'
   */
  size?: OUIButtonSize;

  /**
   * Expand button to full width
   * @default false
   */
  block?: boolean;

  /**
   * Disable the button
   * @default false
   */
  disabled?: boolean;

  /**
   * Display loading affordance
   * @default false
   */
  loading?: boolean;

  /**
   * Button type when rendering a native button element
   * @default 'button'
   */
  type?: "button" | "submit" | "reset";
}

const props = withDefaults(defineProps<Props>(), {
  as: "button",
  color: "primary",
  variant: "solid",
  size: "md",
  block: false,
  disabled: false,
  loading: false,
  type: "button",
});

const BUTTON_SIZE_CLASS_MAP: Record<OUIButtonSize, readonly string[]> = {
  xs: ["px-2", "py-1", "text-xs"],
  sm: ["px-3", "py-1.5", "text-sm"],
  md: ["px-4", "py-2", "text-sm"],
  lg: ["px-6", "py-3", "text-base"],
  xl: ["px-8", "py-4", "text-lg"],
};

const BUTTON_TONE_CLASS_MAP: Record<
  OUIButtonVariant,
  Record<OUIButtonTone, readonly string[]>
> = {
  solid: {
    primary: [
      "bg-primary",
      "text-foreground",
      "border",
      "border-transparent",
      "hover:bg-primary/90",
    ],
    secondary: [
      "bg-secondary",
      "text-foreground",
      "border",
      "border-transparent",
      "hover:bg-secondary/90",
    ],
    success: [
      "bg-success",
      "text-foreground",
      "border",
      "border-transparent",
      "hover:bg-success/90",
    ],
    warning: [
      "bg-warning",
      "text-background",
      "border",
      "border-transparent",
      "hover:bg-warning/90",
    ],
    danger: [
      "bg-danger",
      "text-foreground",
      "border",
      "border-transparent",
      "hover:bg-danger/90",
    ],
    neutral: [
      "bg-surface-raised",
      "text-primary",
      "border",
      "border-border-muted",
      "hover:bg-surface-overlay",
    ],
  },
  soft: {
    primary: [
      "bg-primary/20",
      "text-primary",
      "border",
      "border-primary/25",
      "hover:bg-primary/25",
    ],
    secondary: [
      "bg-secondary/20",
      "text-secondary",
      "border",
      "border-secondary/25",
      "hover:bg-secondary/25",
    ],
    success: [
      "bg-success/20",
      "text-success",
      "border",
      "border-success/25",
      "hover:bg-success/25",
    ],
    warning: [
      "bg-warning/20",
      "text-warning",
      "border",
      "border-warning/30",
      "hover:bg-warning/25",
    ],
    danger: [
      "bg-danger/20",
      "text-danger",
      "border",
      "border-danger/30",
      "hover:bg-danger/25",
    ],
    neutral: [
      "bg-surface-muted/60",
      "text-primary",
      "border",
      "border-border-muted",
      "hover:bg-surface-muted/70",
    ],
  },
  outline: {
    primary: [
      "bg-transparent",
      "border",
      "border-primary",
      "text-primary",
      "hover:bg-primary/10",
    ],
    secondary: [
      "bg-transparent",
      "border",
      "border-secondary",
      "text-secondary",
      "hover:bg-secondary/10",
    ],
    success: [
      "bg-transparent",
      "border",
      "border-success",
      "text-success",
      "hover:bg-success/10",
    ],
    warning: [
      "bg-transparent",
      "border",
      "border-warning",
      "text-warning",
      "hover:bg-warning/10",
    ],
    danger: [
      "bg-transparent",
      "border",
      "border-danger",
      "text-danger",
      "hover:bg-danger/10",
    ],
    neutral: [
      "bg-transparent",
      "border",
      "border-border-muted",
      "text-primary",
      "hover:bg-surface-muted/60",
    ],
  },
  ghost: {
    primary: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-accent-primary",
      "hover:bg-primary/10",
    ],
    secondary: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-accent-secondary",
      "hover:bg-secondary/10",
    ],
    success: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-accent-success",
      "hover:bg-success/10",
    ],
    warning: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-accent-warning",
      "hover:bg-warning/10",
    ],
    danger: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-accent-danger",
      "hover:bg-danger/10",
    ],
    neutral: [
      "bg-transparent",
      "border",
      "border-transparent",
      "text-primary",
      "hover:bg-surface-muted/60",
    ],
  },
};

const variantToken = computed<OUIButtonVariant>(() => props.variant ?? "solid");

const variantToneMap = computed<Record<OUIButtonTone, readonly string[]>>(
  () => BUTTON_TONE_CLASS_MAP[variantToken.value] ?? BUTTON_TONE_CLASS_MAP.solid
);

const colorToken = computed<OUIButtonTone>(() => {
  const toneMap = variantToneMap.value;
  const candidate = props.color ?? "primary";
  return (candidate in toneMap ? candidate : "primary") as OUIButtonTone;
});

const sizeToken = computed<OUIButtonSize>(() => {
  const candidate = props.size ?? "md";
  return (
    candidate in BUTTON_SIZE_CLASS_MAP ? candidate : "md"
  ) as OUIButtonSize;
});

const componentTag = computed(() => props.as ?? "button");
const isNativeButton = computed(
  () =>
    typeof componentTag.value === "string" && componentTag.value === "button"
);
const isDisabled = computed(() => props.disabled || props.loading);
const componentType = computed(() =>
  isNativeButton.value ? props.type : undefined
);
const disabledAttr = computed(() =>
  isNativeButton.value ? isDisabled.value : undefined
);
const ariaDisabled = computed(() =>
  !isNativeButton.value && isDisabled.value ? "true" : undefined
);

const variantClasses = computed<readonly string[]>(() => {
  const toneMap = variantToneMap.value;
  return toneMap[colorToken.value] ?? toneMap.primary;
});

const sizeClasses = computed<readonly string[]>(() => {
  return BUTTON_SIZE_CLASS_MAP[sizeToken.value] ?? BUTTON_SIZE_CLASS_MAP.md;
});

const buttonClasses = computed(() => {
  const classes = [
    "oui-btn-base",
    ...sizeClasses.value,
    ...variantClasses.value,
  ];

  if (props.block) {
    classes.push("w-full", "justify-center");
  }

  if (isDisabled.value) {
    classes.push("cursor-not-allowed", "opacity-60");
  }

  if (props.loading) {
    classes.push("cursor-wait", "animate-pulse");
  }

  return classes;
});

defineOptions({
  inheritAttrs: false,
});
</script>
