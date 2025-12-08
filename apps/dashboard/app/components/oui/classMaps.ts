import type {
  AxisAlign,
  AxisAlignContent,
  AxisAlignContentWithStretch,
  AxisAlignWithBaseline,
  AxisJustify,
  Breakpoint,
  DimensionVariant,
  MarginVariant,
  OUIBorderColor,
  OUIBorderRadius,
  OUIBorderWidth,
  OUIColor,
  OUIOverflow,
  OUIShadow,
  OUISpacing,
  Responsive,
} from "./types";

const spacingScaleMap: Record<Exclude<OUISpacing, "none">, string> = {
  xs: "1",
  sm: "2",
  md: "4",
  lg: "6",
  xl: "8",
  "2xl": "12",
  "3xl": "16",
  "4xl": "20",
  "5xl": "24",
  "6xl": "32",
  "7xl": "40",
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

const overflowMap: Record<OUIOverflow, string> = {
  auto: "overflow-auto",
  hidden: "overflow-hidden",
  scroll: "overflow-scroll",
  visible: "overflow-visible",
};

const overflowXMap: Record<OUIOverflow, string> = {
  auto: "overflow-x-auto",
  hidden: "overflow-x-hidden",
  scroll: "overflow-x-scroll",
  visible: "overflow-x-visible",
};

const overflowYMap: Record<OUIOverflow, string> = {
  auto: "overflow-y-auto",
  hidden: "overflow-y-hidden",
  scroll: "overflow-y-scroll",
  visible: "overflow-y-visible",
};

export const overflowClass = (value: Responsive<OUIOverflow> | undefined) =>
  responsiveClass(value, overflowMap);

export const overflowXClass = (value: Responsive<OUIOverflow> | undefined) =>
  responsiveClass(value, overflowXMap);

export const overflowYClass = (value: Responsive<OUIOverflow> | undefined) =>
  responsiveClass(value, overflowYMap);

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

/**
 * Map a Responsive<T> to Tailwind classes using a provided map of T -> class.
 * - value: e.g. "auto" | { sm: "hidden", lg: "auto" }
 * - map: e.g. { auto: "overflow-auto", hidden: "overflow-hidden", ... }
 *
 * Returns an array of class strings (already prefixed for breakpoints).
 */
export function responsiveClass<T extends string | number>(
  value: Responsive<T> | undefined,
  map: Record<string, string>
): string[] {
  if (!value) return [];

  // Single value (string or number)
  if (typeof value === "string" || typeof value === "number") {
    const mapped = map[String(value)];
    return mapped ? [mapped] : [];
  }

  // Now TypeScript knows value is Partial<Record<Breakpoint, T>>
  const classes: string[] = [];
  const breakpointPrefix: Record<Breakpoint, string> = {
    sm: "sm:",
    md: "md:",
    lg: "lg:",
    xl: "xl:",
    "2xl": "2xl:",
  };

  Object.entries(value).forEach(([k, candidate]) => {
    if (candidate === undefined) return;
    const bp = k as Breakpoint;
    const mapped = map[String(candidate)];
    if (mapped) classes.push(`${breakpointPrefix[bp]}${mapped}`);
  });

  return classes;
}

// Alignment maps
export const alignMap = (prefix: string) => ({
  start: `${prefix}-start`,
  end: `${prefix}-end`,
  center: `${prefix}-center`,
  stretch: `${prefix}-stretch`,
  baseline: `${prefix}-baseline`,
  between: `${prefix}-between`,
  around: `${prefix}-around`,
  evenly: `${prefix}-evenly`,
});

// Gap / spacing map helper
export const spacingMap = (prefix: string) => {
  const map: Record<OUISpacing, string> = {
    none: `${prefix}-0`,
    xs: `${prefix}-1`,
    sm: `${prefix}-2`,
    md: `${prefix}-4`,
    lg: `${prefix}-6`,
    xl: `${prefix}-8`,
    "2xl": `${prefix}-12`,
    "3xl": `${prefix}-16`,
    "4xl": `${prefix}-20`,
    "5xl": `${prefix}-24`,
    "6xl": `${prefix}-32`,
    "7xl": `${prefix}-40`,
  };
  return map;
};

// Auto-fit / auto-fill maps
export const autoFitMap: Record<"xs" | "sm" | "md" | "lg" | "xl", string> = {
  xs: "grid-cols-[repeat(auto-fit,minmax(16rem,1fr))]",
  sm: "grid-cols-[repeat(auto-fit,minmax(20rem,1fr))]",
  md: "grid-cols-[repeat(auto-fit,minmax(24rem,1fr))]",
  lg: "grid-cols-[repeat(auto-fit,minmax(28rem,1fr))]",
  xl: "grid-cols-[repeat(auto-fit,minmax(32rem,1fr))]",
};

export const autoFillMap: Record<"xs" | "sm" | "md" | "lg" | "xl", string> = {
  xs: "grid-cols-[repeat(auto-fill,minmax(16rem,1fr))]",
  sm: "grid-cols-[repeat(auto-fill,minmax(20rem,1fr))]",
  md: "grid-cols-[repeat(auto-fill,minmax(24rem,1fr))]",
  lg: "grid-cols-[repeat(auto-fill,minmax(28rem,1fr))]",
  xl: "grid-cols-[repeat(auto-fill,minmax(32rem,1fr))]",
};

export const marginMap = (prefix: string) => {
  const baseMap = spacingMap(prefix);
  return { ...baseMap, auto: `${prefix}-auto` };
};
