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
    Responsive,
    OUIOverflow,
  } from "./types";
  import {
    backgroundClass,
    borderColorClass,
    borderRadiusClass,
    borderWidthClass,
    heightClass,
    marginMap,
    overflowClass,
    overflowXClass,
    overflowYClass,
    shadowClass,
    spacingMap,
    widthClass,
    responsiveClass,
    minHeightClass,
    maxHeightClass,
    minWidthClass,
    maxWidthClass,
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
    overflow?: Responsive<OUIOverflow>;

    /**
     * Overflow X behavior
     */
    overflowX?: Responsive<OUIOverflow>;

    /**
     * Overflow Y behavior
     */
    overflowY?: Responsive<OUIOverflow>;

    /** Minimum width */
    minW?: DimensionVariant;

    /** Maximum width */
    maxW?: DimensionVariant;

    /** Minimum height */
    minH?: DimensionVariant;

    /** Maximum height */
    maxH?: DimensionVariant;

    /**
     * Display type (supports responsive values)
     */
    display?:
      | Responsive<
          | "block"
          | "inline"
          | "inline-block"
          | "flex"
          | "inline-flex"
          | "grid"
          | "inline-grid"
          | "hidden"
        >;

    /**
     * Add flex: 1 to allow the box to grow
     */
    grow?: boolean;

    /**
     * Prevent shrinking in flex layouts
     */
    shrink?: boolean;
  }

  const props = withDefaults(defineProps<BoxProps>(), {
    as: "div",
    grow: false,
    shrink: true,
  });

  const boxClasses = computed(() => {
    const classes = ["oui-box"];

    if (props.grow) classes.push("flex-1");
    if (!props.shrink) classes.push("flex-shrink-0");

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
    const minWidth = minWidthClass(props.minW);
    if (minWidth) classes.push(minWidth);
    const maxWidth = maxWidthClass(props.maxW);
    if (maxWidth) classes.push(maxWidth);

    // Height classes
    const height = heightClass(props.h);
    if (height) classes.push(height);
    const minHeight = minHeightClass(props.minH);
    if (minHeight) classes.push(minHeight);
    const maxHeight = maxHeightClass(props.maxH);
    if (maxHeight) classes.push(maxHeight);

    // Overflow classes
    classes.push(...overflowClass(props.overflow));
    classes.push(...overflowXClass(props.overflowX));
    classes.push(...overflowYClass(props.overflowY));

    // Position classes
    if (props.position) {
      const positionMap = {
        static: "static",
        relative: "relative",
        absolute: "absolute",
        fixed: "fixed",
        sticky: "sticky",
      } as const;
      classes.push(...responsiveClass(props.position, positionMap));
    }

    // Display classes
    if (props.display) {
      const displayMap = {
        block: "block",
        inline: "inline",
        "inline-block": "inline-block",
        flex: "flex",
        "inline-flex": "inline-flex",
        grid: "grid",
        "inline-grid": "inline-grid",
        hidden: "hidden",
      } as const;
      classes.push(...responsiveClass(props.display, displayMap));
    }

    return classes.join(" ");
  });

  defineOptions({
    inheritAttrs: false,
  });
</script>
