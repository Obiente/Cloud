<template>
  <component :is="as" :class="cardClasses" v-bind="$attrs">
    <slot />
  </component>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import type {
    DimensionVariant,
    MarginVariant,
    OUIColor,
    OUIBorderRadius,
    OUIBorderWidth,
    OUIBorderColor,
    OUIShadow,
    OUISpacing,
    Responsive,
  } from "./types";
  import {
    backgroundClass,
    borderRadiusClass,
    borderWidthClass,
    borderColorClass,
    shadowClass,
    spacingMap,
    marginMap,
    widthClass,
    heightClass,
    minWidthClass,
    maxWidthClass,
    minHeightClass,
    maxHeightClass,
    responsiveClass,
  } from "./classMaps";

  interface Props {
    /**
     * Render as a custom element/component
     * @default 'div'
     */
    as?: string;

    /**
     * Card variant
     */
    variant?: "default" | "raised" | "overlay" | "outline";

    /**
     * Enable pointer + hover/active affordances
     */
    interactive?: boolean;

    /**
     * Lift on hover
     */
    hoverable?: boolean;

    /**
     * Status accent
     */
    status?: "success" | "warning" | "danger" | "info";

    // Spacing
    p?: Responsive<OUISpacing>;
    px?: Responsive<OUISpacing>;
    py?: Responsive<OUISpacing>;
    pt?: Responsive<OUISpacing>;
    pb?: Responsive<OUISpacing>;
    pl?: Responsive<OUISpacing>;
    pr?: Responsive<OUISpacing>;
    m?: Responsive<MarginVariant>;
    mx?: Responsive<MarginVariant>;
    my?: Responsive<MarginVariant>;
    mt?: Responsive<MarginVariant>;
    mb?: Responsive<MarginVariant>;
    ml?: Responsive<MarginVariant>;
    mr?: Responsive<MarginVariant>;

    // Visuals
    bg?: OUIColor;
    rounded?: OUIBorderRadius;
    border?: OUIBorderWidth;
    borderColor?: OUIBorderColor;
    shadow?: OUIShadow;

    // Dimensions
    w?: DimensionVariant;
    h?: DimensionVariant;
    minW?: DimensionVariant;
    maxW?: DimensionVariant;
    minH?: DimensionVariant;
    maxH?: DimensionVariant;
  }

  const props = withDefaults(defineProps<Props>(), {
    as: "div",
    variant: "default",
    interactive: false,
    hoverable: false,
  });

  const cardClasses = computed(() => {
    const classes: string[] = ["oui-card-base"];

    // Variant styling
    const variantMap: Record<NonNullable<Props["variant"]>, string> = {
      default: "bg-surface-base border-default",
      raised: "shadow-md bg-surface-raised border-muted",
      overlay: "shadow-lg bg-surface-overlay border-strong",
      outline: "bg-transparent border-default",
    };
    classes.push(variantMap[props.variant]);

    // Spacing
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

    // Visuals
    const bg = backgroundClass(props.bg);
    if (bg) classes.push(bg);
    const rounded = borderRadiusClass(props.rounded);
    if (rounded) classes.push(rounded);
    const border = borderWidthClass(props.border);
    if (border) classes.push(border);
    const borderColor = borderColorClass(props.borderColor);
    if (borderColor) classes.push(borderColor);
    const shadow = shadowClass(props.shadow);
    if (shadow) classes.push(shadow);

    // Dimensions
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

    // Behaviors
    if (props.interactive) {
      classes.push(
        "cursor-pointer transition-all duration-200 hover:shadow-md hover:bg-hover active:shadow active:bg-active oui-focus-ring"
      );
    }

    if (props.hoverable) {
      classes.push(
        "hover:-translate-y-0.5 hover:shadow-lg transition-all duration-200 cursor-pointer"
      );
    }

    if (props.status) {
      classes.push(`oui-card-status-${props.status}`);
    }

    return classes.join(" ");
  });

  defineOptions({
    inheritAttrs: false,
  });
</script>
