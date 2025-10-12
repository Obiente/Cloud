<template>
  <component
    :is="as"
    :class="flexClasses"
    v-bind="$attrs"
  >
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
} from './types';
import {
  alignContentWithStretchClass,
  alignWithBaselineClass,
  justifyClass,
  backgroundClass,
  borderRadiusClass,
  heightClass,
  marginClass,
  spacingClass,
  widthClass,
} from './classMaps';

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
  gap?: OUISpacing;

  /**
   * Gap in X direction (row gap)
   */
  gapX?: OUISpacing;

  /**
   * Gap in Y direction (column gap)
   */
  gapY?: OUISpacing;

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

const props = withDefaults(defineProps<FlexProps>(), {
  as: 'div',
  direction: 'row',
  wrap: 'nowrap',
  justify: 'start',
  align: 'stretch',
  grow: false,
  shrink: true,
});

const flexClasses = computed(() => {
  const classes = ['oui-flex', 'flex'];

  // Direction classes
  const directionMap = {
    row: 'flex-row',
    'row-reverse': 'flex-row-reverse',
    col: 'flex-col',
    'col-reverse': 'flex-col-reverse',
  };
  classes.push(directionMap[props.direction]);

  // Wrap classes
  const wrapMap = {
    nowrap: 'flex-nowrap',
    wrap: 'flex-wrap',
    'wrap-reverse': 'flex-wrap-reverse',
  };
  classes.push(wrapMap[props.wrap]);

  // Justify content classes
  const justify = justifyClass(props.justify, 'justify');
  if (justify) classes.push(justify);

  // Align items classes
  const align = alignWithBaselineClass(props.align, 'items');
  if (align) classes.push(align);

  // Align content classes
  if (props.alignContent) {
    const alignContent = alignContentWithStretchClass(props.alignContent, 'content');
    if (alignContent) classes.push(alignContent);
  }

  // Gap classes
  const gap = spacingClass(props.gap, 'gap');
  if (gap) classes.push(gap);

  const gapX = spacingClass(props.gapX, 'gap-x');
  if (gapX) classes.push(gapX);

  const gapY = spacingClass(props.gapY, 'gap-y');
  if (gapY) classes.push(gapY);

  // Flex grow/shrink
  if (props.grow) {
    classes.push('flex-1');
  }

  if (!props.shrink) {
    classes.push('flex-shrink-0');
  }

  // Padding classes
  const padding = spacingClass(props.p, 'p');
  if (padding) classes.push(padding);

  const paddingX = spacingClass(props.px, 'px');
  if (paddingX) classes.push(paddingX);

  const paddingY = spacingClass(props.py, 'py');
  if (paddingY) classes.push(paddingY);

  // Margin classes
  const margin = marginClass(props.m, 'm');
  if (margin) classes.push(margin);

  const marginX = marginClass(props.mx, 'mx');
  if (marginX) classes.push(marginX);

  const marginY = marginClass(props.my, 'my');
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

  return classes.join(' ');
});

defineOptions({
  inheritAttrs: false,
});
</script>