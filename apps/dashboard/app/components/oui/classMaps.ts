import type {
  AxisAlign,
  AxisAlignContent,
  AxisAlignContentWithStretch,
  AxisAlignWithBaseline,
  AxisJustify,
  DimensionVariant,
  MarginVariant,
  OUIBorderColor,
  OUIBorderRadius,
  OUIBorderWidth,
  OUIColor,
  OUIOverflow,
  OUIShadow,
  OUISpacing,
} from "./types";

const spacingScaleMap: Record<Exclude<OUISpacing, "none">, string> = {
  xs: "1",
  sm: "2",
  md: "4",
  lg: "6",
  xl: "8",
  "2xl": "12",
};

const mapSpacingToken = (value: OUISpacing) =>
  value === "none" ? "0" : spacingScaleMap[value] ?? value;

const dimensionUtility = (
  value: DimensionVariant | undefined,
  prefix: "w" | "h"
) => {
  if (!value) return undefined;

  if (value === "auto" || value === "fit" || value === "full") {
    return `${prefix}-${value}`;
  }

  if (value === "screen") {
    return `${prefix}-screen`;
  }

  if (prefix === "w") {
    return `max-w-${value}`;
  }

  return undefined;
};

const prefixedAxisClass = (value: string | undefined, prefix: string) =>
  value ? `${prefix}-${value}` : undefined;

export const spacingClass = (value: OUISpacing | undefined, prefix: string) => {
  if (!value) return undefined;
  return `${prefix}-${mapSpacingToken(value)}`;
};

export const marginClass = (
  value: MarginVariant | undefined,
  prefix: "m" | "mx" | "my"
) => {
  if (!value) return undefined;
  if (value === "auto") {
    return `${prefix}-auto`;
  }
  return `${prefix}-${mapSpacingToken(value)}`;
};

export const backgroundClass = (value: OUIColor | undefined) =>
  value ? `bg-${value}` : undefined;

export const borderRadiusClass = (value: OUIBorderRadius | undefined) =>
  value
    ? value === "none" || value === "full"
      ? `rounded-${value}`
      : `rounded-${value}`
    : undefined;

export const shadowClass = (value: OUIShadow | undefined) => {
  if (!value) return undefined;
  return value === "none" ? "shadow-none" : `shadow-${value}`;
};

export const widthClass = (value: DimensionVariant | undefined) =>
  dimensionUtility(value, "w");

export const heightClass = (value: DimensionVariant | undefined) =>
  dimensionUtility(value, "h");

export const borderWidthClass = (value: OUIBorderWidth | undefined) => {
  if (!value) return undefined;
  if (value === "none") return "border-0";
  if (value === "1") return "border";
  return `border-${value}`;
};

export const borderColorClass = (value: OUIBorderColor | undefined) =>
  value ? `border-${value}` : undefined;

export const overflowClass = (value: OUIOverflow | undefined) =>
  value ? `overflow-${value}` : undefined;

export const alignClass = (value: AxisAlign | undefined, prefix: string) =>
  prefixedAxisClass(value, prefix);

export const alignWithBaselineClass = (
  value: AxisAlignWithBaseline | undefined,
  prefix: string
) => prefixedAxisClass(value, prefix);

export const justifyClass = (
  value: AxisJustify | AxisAlignContent | undefined,
  prefix: string
) => prefixedAxisClass(value, prefix);

export const alignContentClass = (
  value: AxisAlignContent | undefined,
  prefix: string
) => prefixedAxisClass(value, prefix);

export const alignContentWithStretchClass = (
  value: AxisAlignContentWithStretch | undefined,
  prefix: string
) => prefixedAxisClass(value, prefix);
