<template>
  <component :is="as" :class="textClasses" :style="textStyles" v-bind="$attrs">
    <OuiSkeleton
      v-if="skeleton"
      :width="skeletonWidth"
      :height="skeletonHeight"
      variant="text"
    />
    <slot v-else />
  </component>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import type { OUISize, SizeRange } from "./types";
  import OuiSkeleton from "./Skeleton.vue";

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
    size?: OUISize;

    /**
     * Mobile text size variant (overrides size on small screens)
     */
    sizeMobile?: OUISize;

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
     * Line height variant (preset) or numeric value
     * @default undefined
     */
    leading?:
      | "none"
      | "tight"
      | "snug"
      | "normal"
      | "relaxed"
      | "loose"
      | number;

    /**
     * Whether text should wrap or not
     * @default undefined
     */
    wrap?: "wrap" | "nowrap" | "balance";

    /**
     * Show skeleton loading state instead of content
     * @default false
     */
    skeleton?: boolean;

    /**
     * Skeleton width (when skeleton prop is true)
     */
    skeletonWidth?: string;

    /**
     * Skeleton height (when skeleton prop is true)
     */
    skeletonHeight?: string;
  }

  const props = withDefaults(defineProps<TextProps>(), {
    as: "p",
    size: "md",
    weight: "normal",
    color: "primary",
    skeleton: false,
    skeletonWidth: "8rem",
    skeletonHeight: "1rem",
  });

  const textClasses = computed(() => {
    const classes = ["oui-text", "break-words"];

    // Size classes
    const sizeClasses: Record<OUISize, string> = {
      xs: "text-xs",
      sm: "text-sm",
      md: "text-base",
      lg: "text-lg",
      xl: "text-xl",
      "2xl": "text-2xl",
      "3xl": "text-3xl",
      "4xl": "text-4xl",
      "5xl": "text-5xl",
      "6xl": "text-6xl",
      "7xl": "text-7xl",
    };

    // Mobile size (if provided, use it on mobile, otherwise use base size)
    if (props.sizeMobile) {
      classes.push(sizeClasses[props.sizeMobile]);
      // Add responsive class for larger screens
      classes.push(`md:${sizeClasses[props.size]}`);
    } else {
      classes.push(sizeClasses[props.size]);
    }

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
      if (typeof props.leading === "number") {
        // Custom numeric line height value
        // Use inline style for custom values
        // Note: We'll handle this via style binding instead of classes
      } else {
        // Preset line height classes
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

  // Handle custom numeric line height values via inline styles
  const textStyles = computed(() => {
    const styles: Record<string, string> = {};

    if (props.leading && typeof props.leading === "number") {
      styles.lineHeight = String(props.leading);
    }

    return Object.keys(styles).length > 0 ? styles : undefined;
  });
</script>
