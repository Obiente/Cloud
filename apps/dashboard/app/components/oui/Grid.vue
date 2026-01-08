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
    Responsive,
  } from "./types";
  import {
    alignMap,
    backgroundClass,
    borderRadiusClass,
    marginMap,
    spacingMap,
    autoFitMap,
    autoFillMap,
    widthClass,
    heightClass,
    responsiveClass,
    minHeightClass,
    maxHeightClass,
    minWidthClass,
    maxWidthClass,
  } from "./classMaps";

  interface GridProps {
    /**
     * The HTML element or component to render as
     * @default 'div'
     */
    as?: string;

    /**
     * Number of columns in the grid.
     * Supports responsive values, e.g. `{ sm: "2", lg: "4" }`
     */
    cols?: Responsive<GridColumns>;

    /**
     * Number of rows in the grid.
     * Supports responsive values.
     */
    rows?: Responsive<GridRows>;

    /**
     * Gap between grid items using OUI spacing scale.
     * Supports responsive values.
     */
    gap?: Responsive<OUISpacing>;

    /**
     * Gap in X direction (column gap)
     * Supports responsive values.
     */
    gapX?: Responsive<OUISpacing>;

    /**
     * Gap in Y direction (row gap)
     * Supports responsive values.
     */
    gapY?: Responsive<OUISpacing>;

    /**
     * Justify items (horizontal alignment within grid cells)
     * Supports responsive values.
     */
    justifyItems?: Responsive<AxisAlign>;

    /**
     * Align items (vertical alignment within grid cells)
     * Supports responsive values.
     */
    alignItems?: Responsive<AxisAlignWithBaseline>;

    /**
     * Justify content (horizontal alignment of the grid within container)
     * Supports responsive values.
     */
    justifyContent?: Responsive<AxisJustify>;

    /**
     * Align content (vertical alignment of the grid within container)
     * Supports responsive values.
     */
    alignContent?: Responsive<AxisAlignContent>;

    /**
     * Auto-fit columns with a minimum width.
     * Supports responsive values.
     */
    autoFit?: Responsive<"xs" | "sm" | "md" | "lg" | "xl" | "2xl" | "3xl">;

    /**
     * Auto-fill columns with a minimum width.
     * Supports responsive values.
     */
    autoFill?: Responsive<"xs" | "sm" | "md" | "lg" | "xl" | "2xl" | "3xl">;

    /**
     * Padding variant using OUI spacing scale
     * Supports responsive values.
     */
    p?: Responsive<OUISpacing>;

    /**
     * Padding X (horizontal) variant
     * Supports responsive values.
     */
    px?: Responsive<OUISpacing>;

    /**
     * Padding Y (vertical) variant
     * Supports responsive values.
     */
    py?: Responsive<OUISpacing>;

    /**
     * Padding top
     * Supports responsive values.
     */
    pt?: Responsive<OUISpacing>;

    /**
     * Padding bottom
     * Supports responsive values.
     */
    pb?: Responsive<OUISpacing>;

    /**
     * Padding left
     * Supports responsive values.
     */
    pl?: Responsive<OUISpacing>;

    /**
     * Padding right
     * Supports responsive values.
     */
    pr?: Responsive<OUISpacing>;

    /**
     * Margin variant using OUI spacing scale
     * Supports responsive values.
     */
    m?: Responsive<MarginVariant>;

    /**
     * Margin X (horizontal) variant
     * Supports responsive values.
     */
    mx?: Responsive<MarginVariant>;

    /**
     * Margin Y (vertical) variant
     * Supports responsive values.
     */
    my?: Responsive<MarginVariant>;

    /**
     * Margin top
     * Supports responsive values.
     */
    mt?: Responsive<MarginVariant>;

    /**
     * Margin bottom
     * Supports responsive values.
     */
    mb?: Responsive<MarginVariant>;

    /**
     * Margin left
     * Supports responsive values.
     */
    ml?: Responsive<MarginVariant>;

    /**
     * Margin right
     * Supports responsive values.
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
  }

  const props = withDefaults(defineProps<GridProps>(), {
    as: "div",
  });

  const gridClasses = computed(() => {
    const classes = ["oui-grid", "grid", "min-w-0", "w-full"];

    // Columns
    classes.push(
      ...responsiveClass(props.cols, {
        none: "grid-cols-none",
        subgrid: "grid-cols-subgrid",
        "1": "grid-cols-1",
        "2": "grid-cols-2",
        "3": "grid-cols-3",
        "4": "grid-cols-4",
        "5": "grid-cols-5",
        "6": "grid-cols-6",
        "7": "grid-cols-7",
        "8": "grid-cols-8",
        "9": "grid-cols-9",
        "10": "grid-cols-10",
        "11": "grid-cols-11",
        "12": "grid-cols-12",
      })
    );

    // Rows
    classes.push(
      ...responsiveClass(props.rows, {
        none: "grid-rows-none",
        subgrid: "grid-rows-subgrid",
        "1": "grid-rows-1",
        "2": "grid-rows-2",
        "3": "grid-rows-3",
        "4": "grid-rows-4",
        "5": "grid-rows-5",
        "6": "grid-rows-6",
      })
    );

    // Gaps
    classes.push(...responsiveClass(props.gap, spacingMap("gap")));
    classes.push(...responsiveClass(props.gapX, spacingMap("gap-x")));
    classes.push(...responsiveClass(props.gapY, spacingMap("gap-y")));

    // Alignment
    if (props.justifyItems)
      classes.push(
        ...responsiveClass(props.justifyItems, alignMap("justify-items"))
      );
    if (props.alignItems)
      classes.push(...responsiveClass(props.alignItems, alignMap("items")));
    if (props.justifyContent)
      classes.push(
        ...responsiveClass(props.justifyContent, alignMap("justify"))
      );
    if (props.alignContent)
      classes.push(...responsiveClass(props.alignContent, alignMap("content")));

    // Auto-fit / auto-fill
    if (props.autoFit)
      classes.push(...responsiveClass(props.autoFit, autoFitMap));
    if (props.autoFill)
      classes.push(...responsiveClass(props.autoFill, autoFillMap));

    // Padding / margin
    classes.push(...responsiveClass(props.p, spacingMap("p")));
    classes.push(...responsiveClass(props.px, spacingMap("px")));
    classes.push(...responsiveClass(props.py, spacingMap("py")));
    classes.push(...responsiveClass(props.pt, spacingMap("pt")));
    classes.push(...responsiveClass(props.pb, spacingMap("pb")));
    classes.push(...responsiveClass(props.pl, spacingMap("pl")));
    classes.push(...responsiveClass(props.pr, spacingMap("pr")));
    classes.push(...responsiveClass(props.m, marginMap("m")));
    classes.push(...responsiveClass(props.mx, marginMap("mx")));
    classes.push(...responsiveClass(props.my, marginMap("my")));
    classes.push(...responsiveClass(props.mt, marginMap("mt")));
    classes.push(...responsiveClass(props.mb, marginMap("mb")));
    classes.push(...responsiveClass(props.ml, marginMap("ml")));
    classes.push(...responsiveClass(props.mr, marginMap("mr")));

    // Background / radius / width / height
    const bg = backgroundClass(props.bg);
    if (bg) classes.push(bg);
    const rounded = borderRadiusClass(props.rounded);
    if (rounded) classes.push(rounded);
    const width = widthClass(props.w);
    if (width) classes.push(width);
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
