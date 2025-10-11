<template>
  <component :is="as" :class="textClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from "vue";

export interface TextProps {
  /**
   * The HTML element or component to render as
   * @default 'p'
   */
  as?: string;

  /**
   * Text size variant
   * @default 'base'
   */
  size?:
    | "xs"
    | "sm"
    | "base"
    | "lg"
    | "xl"
    | "2xl"
    | "3xl"
    | "4xl"
    | "5xl"
    | "6xl";

  /**
   * Text weight variant
   * @default 'normal'
   */
  weight?: "light" | "normal" | "medium" | "semibold" | "bold" | "extrabold";

  /**
   * Text color variant
   * @default 'primary'
   */
  color?:
    | "primary"
    | "secondary"
    | "muted"
    | "accent"
    | "success"
    | "warning"
    | "danger"
    | "white"
    | "inherit";

  /**
   * Text alignment
   * @default undefined
   */
  align?: "left" | "center" | "right" | "justify";

  /**
   * Text decoration
   * @default undefined
   */
  decoration?: "underline" | "line-through" | "none";

  /**
   * Text transform
   * @default undefined
   */
  transform?: "uppercase" | "lowercase" | "capitalize" | "none";

  /**
   * Whether text should truncate with ellipsis
   * @default false
   */
  truncate?: boolean;

  /**
   * Line height variant
   * @default undefined
   */
  leading?: "none" | "tight" | "snug" | "normal" | "relaxed" | "loose";

  /**
   * Whether text should wrap or not
   * @default undefined
   */
  wrap?: "wrap" | "nowrap" | "balance";
}

const props = withDefaults(defineProps<TextProps>(), {
  as: "p",
  size: "base",
  weight: "normal",
  color: "primary",
});

const textClasses = computed(() => {
  const classes = ["oui-text"];

  // Size classes
  const sizeClasses = {
    xs: "text-xs",
    sm: "text-sm",
    base: "text-base",
    lg: "text-lg",
    xl: "text-xl",
    "2xl": "text-2xl",
    "3xl": "text-3xl",
    "4xl": "text-4xl",
    "5xl": "text-5xl",
    "6xl": "text-6xl",
  };
  classes.push(sizeClasses[props.size]);

  // Weight classes
  const weightClasses = {
    light: "font-light",
    normal: "font-normal",
    medium: "font-medium",
    semibold: "font-semibold",
    bold: "font-bold",
    extrabold: "font-extrabold",
  };
  classes.push(weightClasses[props.weight]);

  // Color classes
  const colorClasses = {
    primary: "text-primary",
    secondary: "text-secondary",
    muted: "text-muted",
    accent: "text-accent-primary",
    success: "text-success",
    warning: "text-warning",
    danger: "text-danger",
    white: "text-foreground",
    inherit: "text-inherit",
  };
  classes.push(colorClasses[props.color]);

  // Alignment classes
  if (props.align) {
    const alignClasses = {
      left: "text-left",
      center: "text-center",
      right: "text-right",
      justify: "text-justify",
    };
    classes.push(alignClasses[props.align]);
  }

  // Decoration classes
  if (props.decoration) {
    const decorationClasses = {
      underline: "underline",
      "line-through": "line-through",
      none: "no-underline",
    };
    classes.push(decorationClasses[props.decoration]);
  }

  // Transform classes
  if (props.transform) {
    const transformClasses = {
      uppercase: "uppercase",
      lowercase: "lowercase",
      capitalize: "capitalize",
      none: "normal-case",
    };
    classes.push(transformClasses[props.transform]);
  }

  // Truncate
  if (props.truncate) {
    classes.push("truncate");
  }

  // Leading (line height) classes
  if (props.leading) {
    const leadingClasses = {
      none: "leading-none",
      tight: "leading-tight",
      snug: "leading-snug",
      normal: "leading-normal",
      relaxed: "leading-relaxed",
      loose: "leading-loose",
    };
    classes.push(leadingClasses[props.leading]);
  }

  // Wrap classes
  if (props.wrap) {
    const wrapClasses = {
      wrap: "whitespace-normal",
      nowrap: "whitespace-nowrap",
      balance: "text-balance",
    };
    classes.push(wrapClasses[props.wrap]);
  }

  return classes.join(" ");
});
</script>
