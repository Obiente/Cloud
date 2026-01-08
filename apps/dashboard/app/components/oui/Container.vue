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
  Responsive,
} from "./types";
import {
  backgroundClass,
  borderRadiusClass,
  marginMap,
  responsiveClass,
  shadowClass,
  spacingMap,
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
  p?: Responsive<OUISpacing>;

  /**
   * Padding X (horizontal) variant
   */
  px?: Responsive<OUISpacing>;

  /**
   * Padding Y (vertical) variant
   */
  py?: Responsive<OUISpacing>;

  /**
   * Padding top
   */
  pt?: Responsive<OUISpacing>;

  /**
   * Padding bottom
   */
  pb?: Responsive<OUISpacing>;

  /**
   * Padding left
   */
  pl?: Responsive<OUISpacing>;

  /**
   * Padding right
   */
  pr?: Responsive<OUISpacing>;

  /**
   * Margin variant using OUI spacing scale
   */
  m?: Responsive<MarginVariant>;

  /**
   * Margin X (horizontal) variant
   */
  mx?: Responsive<MarginVariant>;

  /**
   * Margin Y (vertical) variant
   */
  my?: Responsive<MarginVariant>;

  /**
   * Margin top
   */
  mt?: Responsive<MarginVariant>;

  /**
   * Margin bottom
   */
  mb?: Responsive<MarginVariant>;

  /**
   * Margin left
   */
  ml?: Responsive<MarginVariant>;

  /**
   * Margin right
   */
  mr?: Responsive<MarginVariant>;

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
  classes.push(...responsiveClass(props.p, spacingMap("p")));
  classes.push(...responsiveClass(props.px, spacingMap("px")));
  classes.push(...responsiveClass(props.py, spacingMap("py")));
  classes.push(...responsiveClass(props.pt, spacingMap("pt")));
  classes.push(...responsiveClass(props.pb, spacingMap("pb")));
  classes.push(...responsiveClass(props.pl, spacingMap("pl")));
  classes.push(...responsiveClass(props.pr, spacingMap("pr")));

  // Margin classes (only if not using auto centering)
  if (!props.centered) {
    classes.push(...responsiveClass(props.m, marginMap("m")));
    classes.push(...responsiveClass(props.mx, marginMap("mx")));
  }

  classes.push(...responsiveClass(props.my, marginMap("my")));
  classes.push(...responsiveClass(props.mt, marginMap("mt")));
  classes.push(...responsiveClass(props.mb, marginMap("mb")));
  classes.push(...responsiveClass(props.ml, marginMap("ml")));
  classes.push(...responsiveClass(props.mr, marginMap("mr")));

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
