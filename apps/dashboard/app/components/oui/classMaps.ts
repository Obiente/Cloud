import type {
  AxisAlign,
  AxisAlignContent,
  AxisAlignContentWithStretch,
  AxisAlignWithBaseline,
  AxisJustify,
  DimensionVariant,
  MarginVariant,
  OUIColor,
  OUIBorderRadius,
  OUISpacing,
} from './types';

const SPACING_SCALE: Record<OUISpacing, string> = {
  none: '0',
  xs: '1',
  sm: '2',
  md: '4',
  lg: '6',
  xl: '8',
  '2xl': '12',
};

const BORDER_RADIUS_CLASSES: Record<OUIBorderRadius, string> = {
  none: 'rounded-none',
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  xl: 'rounded-xl',
  full: 'rounded-full',
};

const SHADOW_CLASSES = {
  none: 'shadow-none',
  sm: 'shadow-sm',
  md: 'shadow-md',
  lg: 'shadow-lg',
  xl: 'shadow-xl',
  '2xl': 'shadow-2xl',
} as const;

const DIMENSION_CLASSES: Record<DimensionVariant, string> = {
  auto: 'w-auto',
  full: 'w-full',
  fit: 'w-fit',
  screen: 'w-screen',
};

const HEIGHT_CLASSES: Record<DimensionVariant, string> = {
  auto: 'h-auto',
  full: 'h-full',
  fit: 'h-fit',
  screen: 'h-screen',
};

const BORDER_WIDTH_CLASSES = {
  none: 'border-0',
  '1': 'border',
  '2': 'border-2',
  '4': 'border-4',
  '8': 'border-8',
} as const;

const BORDER_COLOR_CLASSES = {
  default: 'border-default',
  muted: 'border-muted',
  strong: 'border-strong',
  'accent-primary': 'border-accent-primary',
  'accent-secondary': 'border-accent-secondary',
  success: 'border-success',
  warning: 'border-warning',
  danger: 'border-danger',
  info: 'border-info',
} as const;

const OVERFLOW_CLASSES = {
  visible: 'overflow-visible',
  hidden: 'overflow-hidden',
  auto: 'overflow-auto',
  scroll: 'overflow-scroll',
} as const;

const AXIS_ALIGN_SUFFIXES: Record<AxisAlign, string> = {
  start: 'start',
  end: 'end',
  center: 'center',
  stretch: 'stretch',
};

const AXIS_ALIGN_WITH_BASELINE_SUFFIXES: Record<AxisAlignWithBaseline, string> = {
  ...AXIS_ALIGN_SUFFIXES,
  baseline: 'baseline',
};

const AXIS_JUSTIFY_SUFFIXES: Record<AxisJustify, string> = {
  start: 'start',
  end: 'end',
  center: 'center',
  between: 'between',
  around: 'around',
  evenly: 'evenly',
};

const AXIS_ALIGN_CONTENT_SUFFIXES: Record<AxisAlignContent, string> = {
  start: 'start',
  end: 'end',
  center: 'center',
  between: 'between',
  around: 'around',
  evenly: 'evenly',
};

const AXIS_ALIGN_CONTENT_WITH_STRETCH_SUFFIXES: Record<AxisAlignContentWithStretch, string> = {
  ...AXIS_ALIGN_CONTENT_SUFFIXES,
  stretch: 'stretch',
};

const prefixedAxisClass = <T extends string>(
  value: T | undefined,
  prefix: string,
  map: Record<T, string>,
) => {
  if (!value) return undefined;
  return `${prefix}-${map[value]}`;
};

export const spacingClass = (value: OUISpacing | undefined, prefix: string) => {
  if (!value) return undefined;
  return `${prefix}-${SPACING_SCALE[value]}`;
};

export const marginClass = (value: MarginVariant | undefined, prefix: 'm' | 'mx' | 'my') => {
  if (!value) return undefined;
  if (value === 'auto') {
    return `${prefix}-auto`;
  }
  return `${prefix}-${SPACING_SCALE[value]}`;
};

export const backgroundClass = (value: OUIColor | undefined) =>
  value ? `bg-${value}` : undefined;

export const borderRadiusClass = (value: OUIBorderRadius | undefined) =>
  value ? BORDER_RADIUS_CLASSES[value] : undefined;

export const shadowClass = (value: keyof typeof SHADOW_CLASSES | undefined) =>
  value ? SHADOW_CLASSES[value] : undefined;

export const widthClass = (value: DimensionVariant | undefined) =>
  value ? DIMENSION_CLASSES[value] : undefined;

export const heightClass = (value: DimensionVariant | undefined) =>
  value ? HEIGHT_CLASSES[value] : undefined;

export const borderWidthClass = (
  value: keyof typeof BORDER_WIDTH_CLASSES | undefined,
) => (value ? BORDER_WIDTH_CLASSES[value] : undefined);

export const borderColorClass = (
  value: keyof typeof BORDER_COLOR_CLASSES | undefined,
) => (value ? BORDER_COLOR_CLASSES[value] : undefined);

export const overflowClass = (
  value: keyof typeof OVERFLOW_CLASSES | undefined,
) => (value ? OVERFLOW_CLASSES[value] : undefined);

export const alignClass = (value: AxisAlign | undefined, prefix: string) =>
  prefixedAxisClass(value, prefix, AXIS_ALIGN_SUFFIXES);

export const alignWithBaselineClass = (
  value: AxisAlignWithBaseline | undefined,
  prefix: string,
) => prefixedAxisClass(value, prefix, AXIS_ALIGN_WITH_BASELINE_SUFFIXES);

export const justifyClass = (
  value: AxisJustify | AxisAlignContent | undefined,
  prefix: string,
) => prefixedAxisClass(value, prefix, AXIS_JUSTIFY_SUFFIXES);

export const alignContentClass = (
  value: AxisAlignContent | undefined,
  prefix: string,
) => prefixedAxisClass(value, prefix, AXIS_ALIGN_CONTENT_SUFFIXES);

export const alignContentWithStretchClass = (
  value: AxisAlignContentWithStretch | undefined,
  prefix: string,
) => prefixedAxisClass(value, prefix, AXIS_ALIGN_CONTENT_WITH_STRETCH_SUFFIXES);
