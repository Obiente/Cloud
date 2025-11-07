<template>
  <component :is="as" :class="containerClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
import type {
  ContainerBreakpoint,
  ContainerSize,
  MarginVariant,
  OUIColor,
  OUIBorderRadius,
  OUISpacing,
} from "./types";
import {
  backgroundClass,
  borderRadiusClass,
  marginClass,
  shadowClass,
  spacingClass,
} from "./classMaps";

interface ContainerProps {
  /**
   * The HTML element or component to render as
   * @default 'div'
   */
  as?: string;

  /**
   * Container size variant
   * @default 'default'
   */
  size?: ContainerSize;

  /**
   * Whether container should be centered
   * @default true
   */
  centered?: boolean;

  /**
   * Padding variant using OUI spacing scale
   * @default 'md'
   */
  p?: OUISpacing;

  /**
   * Padding X (horizontal) variant
   */
  px?: OUISpacing;

  /**
   * Padding Y (vertical) variant
   */
  py?: OUISpacing;

  /**
   * Margin variant using OUI spacing scale
   */
  m?: MarginVariant;

  /**
   * Margin X (horizontal) variant
   */
  mx?: MarginVariant;

  /**
   * Margin Y (vertical) variant
   */
  my?: MarginVariant;

  /**
   * Background color using OUI color system
   */
  bg?: OUIColor;

  /**
   * Border radius variant
   */
  rounded?: OUIBorderRadius;

  /**
   * Shadow/elevation variant using OUI elevation system
   */
  shadow?: "none" | "sm" | "md" | "lg" | "xl" | "2xl";

  /**
   * Whether the container should be fluid (100% width)
   * @default false
   */
  fluid?: boolean;

  /**
   * Responsive breakpoint behavior
   */
  breakpoint?: ContainerBreakpoint;
}

const props = withDefaults(defineProps<ContainerProps>(), {
  as: "div",
  size: "full",
  centered: true,
  p: "md",
  fluid: false,
  breakpoint: "always",
});

const containerClasses = computed(() => {
  const classes = ["oui-container", "min-w-0", "w-full"];

  // Base container behavior
  if (props.fluid) {
    classes.push("w-full");
  } else {
    // Max width classes based on size
    const sizeMap = {
      xs: "max-w-xs", // 20rem (320px)
      sm: "max-w-sm", // 24rem (384px)
      md: "max-w-md", // 28rem (448px)
      lg: "max-w-lg", // 32rem (512px)
      xl: "max-w-xl", // 36rem (576px)
      "2xl": "max-w-2xl", // 42rem (672px)
      "3xl": "max-w-3xl", // 48rem (768px)
      "4xl": "max-w-4xl", // 56rem (896px)
      "5xl": "max-w-5xl", // 64rem (1024px)
      "6xl": "max-w-6xl", // 72rem (1152px)
      "7xl": "max-w-7xl", // 80rem (1280px)
      full: "max-w-full", // 100%
    };
    classes.push(sizeMap[props.size]);
  }

  // Centering
  if (props.centered) {
    classes.push("mx-auto");
  }

  // Responsive behavior
  if (props.breakpoint !== "always") {
    const breakpointMap = {
      sm: "sm:container",
      md: "md:container",
      lg: "lg:container",
      xl: "xl:container",
      "2xl": "2xl:container",
    };
    classes.push(breakpointMap[props.breakpoint as keyof typeof breakpointMap]);
  }

  // Padding classes
  const padding = spacingClass(props.p, "p");
  if (padding) classes.push(padding);

  const paddingX = spacingClass(props.px, "px");
  if (paddingX) classes.push(paddingX);

  const paddingY = spacingClass(props.py, "py");
  if (paddingY) classes.push(paddingY);

  // Margin classes (only if not using auto centering)
  if (!props.centered) {
    const margin = marginClass(props.m, "m");
    if (margin) classes.push(margin);

    const marginX = marginClass(props.mx, "mx");
    if (marginX) classes.push(marginX);
  }

  const marginY = marginClass(props.my, "my");
  if (marginY) classes.push(marginY);

  // Background classes
  const bg = backgroundClass(props.bg);
  if (bg) classes.push(bg);

  // Border radius classes
  const rounded = borderRadiusClass(props.rounded);
  if (rounded) classes.push(rounded);

  // Shadow classes
  const shadow = shadowClass(props.shadow);
  if (shadow) classes.push(shadow);

  return classes.join(" ");
});

defineOptions({
  inheritAttrs: false,
});
</script>
