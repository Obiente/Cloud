<template>
  <component
    :is="as"
    :class="stackClasses"
    v-bind="$attrs"
  >
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
} from './types';
import {
  alignClass,
  justifyClass,
  backgroundClass,
  borderRadiusClass,
  heightClass,
  marginClass,
  spacingClass,
  widthClass,
} from './classMaps';

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
  direction?: 'vertical' | 'horizontal';

  /**
   * Gap between stack items using OUI spacing scale
   * @default 'md'
   */
  gap?: OUISpacing;

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

  /**
   * Divider between stack items
   */
  divider?: boolean;

  /**
   * Divider color using OUI color system
   */
  dividerColor?: 'default' | 'muted' | 'strong';
}

const props = withDefaults(defineProps<StackProps>(), {
  as: 'div',
  direction: 'vertical',
  gap: 'md',
  align: 'stretch',
  justify: 'start',
  wrap: false,
  divider: false,
  dividerColor: 'muted',
});

const stackClasses = computed(() => {
  const classes = ['oui-stack', 'flex'];

  // Direction classes
  if (props.direction === 'horizontal') {
    classes.push('flex-row');
  } else {
    classes.push('flex-col');
  }

  // Wrap classes
  if (props.wrap) {
    classes.push('flex-wrap');
  }

  // Gap classes
  const gap = spacingClass(props.gap, 'gap');
  if (gap) classes.push(gap);

  // Alignment classes
  const align = alignClass(props.align, 'items');
  if (align) classes.push(align);

  // Justify classes
  const justify = justifyClass(props.justify, 'justify');
  if (justify) classes.push(justify);

  // Divider classes
  if (props.divider) {
    if (props.direction === 'horizontal') {
      classes.push('divide-x');
    } else {
      classes.push('divide-y');
    }

    const dividerColorMap = {
      default: 'divide-default',
      muted: 'divide-muted',
      strong: 'divide-strong',
    };
    classes.push(dividerColorMap[props.dividerColor]);
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