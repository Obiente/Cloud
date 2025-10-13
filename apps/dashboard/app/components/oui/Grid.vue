<template>
  <component :is="as" :class="gridClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
import type {
  AxisAlign,
  AxisAlignContent,
  AxisAlignWithBaseline,
  AxisJustify,
  DimensionVariant,
  GridColumns,
  GridRows,
  MarginVariant,
  OUIColor,
  OUIBorderRadius,
  OUISpacing,
} from "./types";
import {
  alignClass,
  alignContentClass,
  alignWithBaselineClass,
  justifyClass,
  backgroundClass,
  borderRadiusClass,
  marginClass,
  spacingClass,
  widthClass,
  heightClass,
} from "./classMaps";

interface GridProps {
  /**
   * The HTML element or component to render as
   * @default 'div'
   */
  as?: string;

  /**
   * Number of columns in the grid
   */
  cols?: GridColumns;

  /**
   * Number of rows in the grid
   */
  rows?: GridRows;

  /**
   * Responsive columns - small screens
   */
  colsSm?: GridColumns;

  /**
   * Responsive columns - medium screens
   */
  colsMd?: GridColumns;

  /**
   * Responsive columns - large screens
   */
  colsLg?: GridColumns;

  /**
   * Responsive columns - extra large screens
   */
  colsXl?: GridColumns;

  /**
   * Responsive columns - 2xl screens
   */
  cols2xl?: GridColumns;

  /**
   * Gap between grid items using OUI spacing scale
   */
  gap?: OUISpacing;

  /**
   * Gap in X direction (column gap)
   */
  gapX?: OUISpacing;

  /**
   * Gap in Y direction (row gap)
   */
  gapY?: OUISpacing;

  /**
   * Justify items (horizontal alignment within grid cells)
   */
  justifyItems?: AxisAlign;

  /**
   * Align items (vertical alignment within grid cells)
   */
  alignItems?: AxisAlignWithBaseline;

  /**
   * Justify content (horizontal alignment of the grid within container)
   */
  justifyContent?: AxisJustify;

  /**
   * Align content (vertical alignment of the grid within container)
   */
  alignContent?: AxisAlignContent;

  /**
   * Auto-fit columns with minimum width
   */
  autoFit?: "xs" | "sm" | "md" | "lg" | "xl";

  /**
   * Auto-fill columns with minimum width
   */
  autoFill?: "xs" | "sm" | "md" | "lg" | "xl";

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
   * Width variant
   */
  w?: DimensionVariant;

  /**
   * Height variant
   */
  h?: DimensionVariant;
}

const props = withDefaults(defineProps<GridProps>(), {
  as: "div",
});

const gridClasses = computed(() => {
  const classes = ["oui-grid", "grid"];

  // Column classes
  if (props.cols) {
    if (props.cols === "none") {
      classes.push("grid-cols-none");
    } else if (props.cols === "subgrid") {
      classes.push("grid-cols-subgrid");
    } else {
      classes.push(`grid-cols-${props.cols}`);
    }
  }

  // Row classes
  if (props.rows) {
    if (props.rows === "none") {
      classes.push("grid-rows-none");
    } else if (props.rows === "subgrid") {
      classes.push("grid-rows-subgrid");
    } else {
      classes.push(`grid-rows-${props.rows}`);
    }
  }

  // Responsive columns
  const responsiveCols: Record<string, GridColumns | undefined> = {
    sm: props.colsSm,
    md: props.colsMd,
    lg: props.colsLg,
    xl: props.colsXl,
    "2xl": props.cols2xl,
  };

  Object.entries(responsiveCols).forEach(([breakpoint, value]) => {
    if (!value) return;
    const prefix = breakpoint === "2xl" ? "2xl" : breakpoint;
    if (value === "none") {
      classes.push(`${prefix}:grid-cols-none`);
    } else if (value === "subgrid") {
      classes.push(`${prefix}:grid-cols-subgrid`);
    } else {
      classes.push(`${prefix}:grid-cols-${value}`);
    }
  });

  // Auto-fit and auto-fill
  if (props.autoFit) {
    const autoFitMap = {
      xs: "grid-cols-[repeat(auto-fit,minmax(16rem,1fr))]",
      sm: "grid-cols-[repeat(auto-fit,minmax(20rem,1fr))]",
      md: "grid-cols-[repeat(auto-fit,minmax(24rem,1fr))]",
      lg: "grid-cols-[repeat(auto-fit,minmax(28rem,1fr))]",
      xl: "grid-cols-[repeat(auto-fit,minmax(32rem,1fr))]",
    };
    classes.push(autoFitMap[props.autoFit]);
  }

  if (props.autoFill) {
    const autoFillMap = {
      xs: "grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]",
      sm: "grid-cols-[repeat(auto-fill,minmax(20rem,1fr))]",
      md: "grid-cols-[repeat(auto-fill,minmax(24rem,1fr))]",
      lg: "grid-cols-[repeat(auto-fill,minmax(28rem,1fr))]",
      xl: "grid-cols-[repeat(auto-fill,minmax(32rem,1fr))]",
    };
    classes.push(autoFillMap[props.autoFill]);
  }

  // Gap classes
  const gap = spacingClass(props.gap, "gap");
  if (gap) classes.push(gap);

  const gapX = spacingClass(props.gapX, "gap-x");
  if (gapX) classes.push(gapX);

  const gapY = spacingClass(props.gapY, "gap-y");
  if (gapY) classes.push(gapY);

  // Justify and align
  if (props.justifyItems) {
    const justifyItems = alignClass(props.justifyItems, "justify-items");
    if (justifyItems) classes.push(justifyItems);
  }

  if (props.alignItems) {
    const alignItems = alignWithBaselineClass(props.alignItems, "items");
    if (alignItems) classes.push(alignItems);
  }

  if (props.justifyContent) {
    const justifyContent = justifyClass(props.justifyContent, "justify");
    if (justifyContent) classes.push(justifyContent);
  }

  if (props.alignContent) {
    const alignContent = alignContentClass(props.alignContent, "content");
    if (alignContent) classes.push(alignContent);
  }

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

  // Width classes
  const width = widthClass(props.w);
  if (width) classes.push(width);

  // Height classes
  const height = heightClass(props.h);
  if (height) classes.push(height);

  return classes.join(" ");
});

defineOptions({
  inheritAttrs: false,
});
</script>
