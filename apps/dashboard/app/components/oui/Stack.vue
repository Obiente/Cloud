<template>
  <component :is="as" :class="stackClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
import type {
  AxisAlign,
  AxisJustify,
  DimensionVariant,
  MarginVariant,
  OUIColor,
  OUIBorderRadius,
  OUISpacing,
  Responsive,
} from "./types";
import {
  alignClass,
  justifyClass,
  backgroundClass,
  borderRadiusClass,
  heightClass,
  marginMap,
  responsiveClass,
  spacingMap,
  widthClass,
    minHeightClass,
    maxHeightClass,
    minWidthClass,
    maxWidthClass,
} from "./classMaps";

interface StackProps {
  /**
   * The HTML element or component to render as
   * @default 'div'
   */
  as?: string;

  /**
   * Stack direction
   * @default 'vertical'
   */
  direction?: "vertical" | "horizontal";

  /**
   * Gap between stack items using OUI spacing scale
   * @default 'md'
   */
  gap?: Responsive<OUISpacing>;

  /**
   * Alignment of items in the stack
   * @default 'stretch'
   */
  align?: AxisAlign;

  /**
   * Justify content (distribution along main axis)
   * @default 'start'
   */
  justify?: AxisJustify;

  /**
   * Whether items should wrap to new lines
   * @default false
   */
  wrap?: boolean;

  /**
   * Padding variant using OUI spacing scale
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
   * Width variant
   */
  w?: DimensionVariant;

  /**
   * Height variant
   */
  h?: DimensionVariant;

  /** Minimum width */
  minW?: DimensionVariant;

  /** Maximum width */
  maxW?: DimensionVariant;

  /** Minimum height */
  minH?: DimensionVariant;

  /** Maximum height */
  maxH?: DimensionVariant;

  /**
   * Divider between stack items
   */
  divider?: boolean;

  /**
   * Divider color using OUI color system
   */
  dividerColor?: "default" | "muted" | "strong";
}

const props = withDefaults(defineProps<StackProps>(), {
  as: "div",
  direction: "vertical",
  gap: "md",
  align: "stretch",
  justify: "start",
  wrap: false,
  divider: false,
  dividerColor: "muted",
});

const stackClasses = computed(() => {
  const classes = ["oui-stack", "flex", "min-w-0"];

  // Direction classes
  if (props.direction === "horizontal") {
    classes.push("flex-row");
    // Auto-wrap on mobile for horizontal stacks
    if (!props.wrap) {
      classes.push("flex-wrap", "md:flex-nowrap");
    }
  } else {
    classes.push("flex-col");
  }

  // Wrap classes
  if (props.wrap) {
    classes.push("flex-wrap");
  }

  // Gap classes
  classes.push(...responsiveClass(props.gap, spacingMap("gap")));

  // Alignment classes
  const align = alignClass(props.align, "items");
  if (align) classes.push(align);

  // Justify classes
  const justify = justifyClass(props.justify, "justify");
  if (justify) classes.push(justify);

  // Divider classes
  if (props.divider) {
    if (props.direction === "horizontal") {
      classes.push("divide-x");
    } else {
      classes.push("divide-y");
    }

    const dividerColorMap = {
      default: "divide-default",
      muted: "divide-muted",
      strong: "divide-strong",
    };
    classes.push(dividerColorMap[props.dividerColor]);
  }

  // Padding classes
  classes.push(...responsiveClass(props.p, spacingMap("p")));
  classes.push(...responsiveClass(props.px, spacingMap("px")));
  classes.push(...responsiveClass(props.py, spacingMap("py")));
  classes.push(...responsiveClass(props.pt, spacingMap("pt")));
  classes.push(...responsiveClass(props.pb, spacingMap("pb")));
  classes.push(...responsiveClass(props.pl, spacingMap("pl")));
  classes.push(...responsiveClass(props.pr, spacingMap("pr")));

  // Margin classes
  classes.push(...responsiveClass(props.m, marginMap("m")));
  classes.push(...responsiveClass(props.mx, marginMap("mx")));
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

  // Width classes
  const width = widthClass(props.w);
  if (width) classes.push(width);

  // Height classes
  const height = heightClass(props.h);
  if (height) classes.push(height);

  const minWidth = minWidthClass(props.minW);
  if (minWidth) classes.push(minWidth);

  const maxWidth = maxWidthClass(props.maxW);
  if (maxWidth) classes.push(maxWidth);

  const minHeight = minHeightClass(props.minH);
  if (minHeight) classes.push(minHeight);

  const maxHeight = maxHeightClass(props.maxH);
  if (maxHeight) classes.push(maxHeight);

  return classes.join(" ");
});

defineOptions({
  inheritAttrs: false,
});
</script>
