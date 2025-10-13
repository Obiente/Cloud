<template>
  <component :is="as" :class="boxClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
  import type {
    DimensionVariant,
    MarginVariant,
    OUIColor,
    OUIBorderRadius,
    OUISpacing,
    OUIBorderWidth,
    OUIBorderColor,
    OUIShadow,
  } from "./types";
  import {
    backgroundClass,
    borderColorClass,
    borderRadiusClass,
    borderWidthClass,
    heightClass,
    marginClass,
    overflowClass,
    shadowClass,
    spacingClass,
    widthClass,
  } from "./classMaps";

  interface BoxProps {
    /**
     * The HTML element or component to render as
     * @default 'div'
     */
    as?: string;

    /**
     * Padding variant using OUI spacing scale
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
     * Border width
     */
    border?: OUIBorderWidth;

    /**
     * Border color using OUI color system
     */
    borderColor?: OUIBorderColor;

    /**
     * Shadow/elevation variant using OUI elevation system
     */
    shadow?: OUIShadow;

    /**
     * Width variant
     */
    w?: DimensionVariant;

    /**
     * Height variant
     */
    h?: DimensionVariant;

    /**
     * Position variant
     */
    position?: "static" | "relative" | "absolute" | "fixed" | "sticky";

    /**
     * Overflow behavior
     */
    overflow?: "visible" | "hidden" | "auto" | "scroll";

    /**
     * Display type
     */
    display?:
      | "block"
      | "inline"
      | "inline-block"
      | "flex"
      | "inline-flex"
      | "grid"
      | "inline-grid"
      | "hidden";
  }

  const props = withDefaults(defineProps<BoxProps>(), {
    as: "div",
  });

  const boxClasses = computed(() => {
    const classes = ["oui-box"];

    // Padding classes
    const padding = spacingClass(props.p, "p");
    if (padding) classes.push(padding);

    const paddingX = spacingClass(props.px, "px");
    if (paddingX) classes.push(paddingX);

    const paddingY = spacingClass(props.py, "py");
    if (paddingY) classes.push(paddingY);

    // Margin classes
    const margin = marginClass(props.m, "m");
    if (margin) classes.push(margin);

    const marginX = marginClass(props.mx, "mx");
    if (marginX) classes.push(marginX);

    const marginY = marginClass(props.my, "my");
    if (marginY) classes.push(marginY);

    // Background classes
    const bg = backgroundClass(props.bg);
    if (bg) classes.push(bg);

    // Border radius classes
    const rounded = borderRadiusClass(props.rounded);
    if (rounded) classes.push(rounded);

    // Border classes
    const border = borderWidthClass(props.border);
    if (border) classes.push(border);

    const borderColor = borderColorClass(props.borderColor);
    if (borderColor) classes.push(borderColor);

    // Shadow classes
    const shadow = shadowClass(props.shadow);
    if (shadow) classes.push(shadow);

    // Width classes
    const width = widthClass(props.w);
    if (width) classes.push(width);

    // Height classes
    const height = heightClass(props.h);
    if (height) classes.push(height);

    // Position classes
    if (props.position) {
      classes.push(props.position);
    }

    // Overflow classes
    const overflow = overflowClass(props.overflow);
    if (overflow) classes.push(overflow);

    // Display classes
    if (props.display) {
      classes.push(props.display);
    }

    return classes.join(" ");
  });

  defineOptions({
    inheritAttrs: false,
  });
</script>
