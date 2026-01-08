<template>
  <component :is="as" :class="flexClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
  import type {
    AxisAlignContentWithStretch,
    AxisAlignWithBaseline,
    AxisJustify,
    DimensionVariant,
    FlexDirection,
    FlexWrap,
    MarginVariant,
    OUIColor,
    OUIBorderRadius,
    OUISpacing,
    OUIOverflow,
    OUISize,
    Responsive,
  } from "./types";
  import {
    alignContentWithStretchClass,
    alignWithBaselineClass,
    justifyClass,
    backgroundClass,
    borderRadiusClass,
    heightClass,
    marginMap,
    responsiveClass,
    spacingMap,
    widthClass,
    overflowClass,
    overflowXClass,
    overflowYClass,
    minHeightClass,
    maxHeightClass,
    minWidthClass,
    maxWidthClass,
  } from "./classMaps";

  interface FlexProps {
    /**
     * The HTML element or component to render as
     * @default 'div'
     */
    as?: string;

    /**
     * Flex direction
     * @default 'row'
     */
    direction?: FlexDirection;

    /**
     * Flex wrap behavior
     * @default 'nowrap'
     */
    wrap?: FlexWrap;

    /**
     * Justify content (main axis alignment)
     * @default 'start'
     */
    justify?: AxisJustify;

    /**
     * Align items (cross axis alignment)
     * @default 'stretch'
     */
    align?: AxisAlignWithBaseline;

    /**
     * Align content (for wrapped flex containers)
     */
    alignContent?: AxisAlignContentWithStretch;

    /**
     * Gap between flex items using OUI spacing scale
     */
    gap?: Responsive<OUISpacing>;

    /**
     * Gap in X direction (row gap)
     */
    gapX?: Responsive<OUISpacing>;

    /**
     * Gap in Y direction (column gap)
     */
    gapY?: Responsive<OUISpacing>;

    /**
     * Whether items should grow to fill available space
     * @default false
     */
    grow?: boolean;

    /**
     * Whether items should shrink when space is limited
     * @default true
     */
    shrink?: boolean;

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

    /**
     * Minimum width
     */
    minW?: DimensionVariant;

    /**
     * Maximum width
     */
    maxW?: DimensionVariant;

    /**
     * Minimum height
     */
    minH?: DimensionVariant;

    /**
     * Maximum height
     */
    maxH?: DimensionVariant;

    /**
     * Overflow behavior; supports responsive object e.g. { sm: 'hidden', lg: 'auto' }
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
  }

  const props = withDefaults(defineProps<FlexProps>(), {
    as: "div",
    direction: "row",
    wrap: "nowrap",
    justify: "start",
    align: "stretch",
    grow: false,
    shrink: true,
  });

  const flexClasses = computed(() => {
    const classes = ["oui-flex", "flex", "min-w-0"];

    // Direction classes
    const directionMap = {
      row: "flex-row",
      "row-reverse": "flex-row-reverse",
      col: "flex-col",
      "col-reverse": "flex-col-reverse",
    };
    classes.push(directionMap[props.direction]);

    // Wrap classes - default to wrap on mobile for row direction
    const wrapMap = {
      nowrap: "flex-nowrap",
      wrap: "flex-wrap",
      "wrap-reverse": "flex-wrap-reverse",
    };

    // If row direction and nowrap, allow wrapping on mobile
    if (props.direction === "row" && props.wrap === "nowrap") {
      classes.push("flex-wrap", "md:flex-nowrap");
    } else {
      classes.push(wrapMap[props.wrap]);
    }

    // Justify content classes
    const justify = justifyClass(props.justify, "justify");
    if (justify) classes.push(justify);

    // Align items classes
    const align = alignWithBaselineClass(props.align, "items");
    if (align) classes.push(align);

    // Align content classes
    if (props.alignContent) {
      const alignContent = alignContentWithStretchClass(
        props.alignContent,
        "content"
      );
      if (alignContent) classes.push(alignContent);
    }

    // Gap classes
    classes.push(...responsiveClass(props.gap, spacingMap("gap")));
    classes.push(...responsiveClass(props.gapX, spacingMap("gap-x")));
    classes.push(...responsiveClass(props.gapY, spacingMap("gap-y")));

    // Flex grow/shrink
    if (props.grow) {
      classes.push("flex-1");
    }

    if (!props.shrink) {
      classes.push("flex-shrink-0");
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

    classes.push(...overflowClass(props.overflow));
    classes.push(...overflowXClass(props.overflowX));
    classes.push(...overflowYClass(props.overflowY));

    return classes.join(" ");
  });

  defineOptions({
    inheritAttrs: false,
  });
</script>
