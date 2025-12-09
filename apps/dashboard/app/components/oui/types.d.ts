/** ------------------------------------------------------------------
 *  Numeric helpers
 *  ------------------------------------------------------------------ */

// Allow both numeric literals and their string equivalents
export type NumericWithString<T extends number | string> = T | `${T & number}`;

/** ------------------------------------------------------------------
 *  Range helpers
 *  ------------------------------------------------------------------ */

// Build a tuple of length N
type BuildTuple<
  N extends number,
  Acc extends unknown[] = []
> = Acc["length"] extends N ? Acc : BuildTuple<N, [...Acc, unknown]>;

// Extract only numeric indices from a tuple
type Indices<T extends unknown[]> = Exclude<keyof T, keyof any[]>;

// Convert string indices to number literals
type ToNumber<T> = T extends `${infer N extends number}` ? N : never;

// Enumerate 0..N-1
type Enumerate<N extends number> = ToNumber<Indices<BuildTuple<N>>>;

// Inclusive range: Start..End
export type Range<Start extends number, End extends number> =
  | Exclude<Enumerate<End>, Enumerate<Start>>
  | Start
  | End;

// Range with optional string suffixes (e.g. "subgrid")
export type NumericRangeWithSuffixes<
  T extends number,
  Suffixes extends string = never
> = NumericWithString<T> | Suffixes;

/** ------------------------------------------------------------------
 *  Core design tokens
 *  ------------------------------------------------------------------ */

export type OUIShadow = "none" | "sm" | "md" | "lg" | "xl" | "2xl";
export type OUIBorderWidth = "none" | "1" | "2" | "4" | "8";
export type OUIBorderColor =
  | "default"
  | "muted"
  | "strong"
  | "accent-primary"
  | "accent-secondary"
  | "success"
  | "warning"
  | "danger"
  | "info";
export type OUIOverflow = "visible" | "hidden" | "auto" | "scroll";

export type OUISpacing = SizeRange<"none", "7xl">;

export type OUIColor =
  | "transparent"
  | "background"
  | "surface-base"
  | "surface-raised"
  | "surface-overlay"
  | "surface-muted"
  | "accent-primary"
  | "accent-secondary"
  | "success"
  | "warning"
  | "danger"
  | "info";

export type OUIBorderRadius = "none" | "sm" | "md" | "lg" | "xl" | "full";
export type OUIOverflow = "visible" | "hidden" | "auto" | "scroll";
export type OUISize = SizeRange<"3xs", "7xl">;

    
/** ------------------------------------------------------------------
 *  Size scale enum + derived ranges
 *  ------------------------------------------------------------------ */

export enum SizeScale {
  "none" = 0,
  "3xs" = 1,
  "2xs" = 2,
  "xs" = 3,
  "sm" = 4,
  "md" = 5,
  "lg" = 6,
  "xl" = 7,
  "2xl" = 8,
  "3xl" = 9,
  "4xl" = 10,
  "5xl" = 11,
  "6xl" = 12,
  "7xl" = 13,
  "full" = 14,
}

type EnumValue<T, K extends keyof T> = T[K];

type KeysMatchingValue<T, V> = {
  [K in keyof T]: T[K] extends V ? K : never;
}[keyof T];

// Map enum values back to key names within a range
export type SizeRange<
  Start extends keyof typeof SizeScale,
  End extends keyof typeof SizeScale
> = KeysMatchingValue<
  typeof SizeScale,
  Range<EnumValue<typeof SizeScale, Start>, EnumValue<typeof SizeScale, End>>
>;

/** ------------------------------------------------------------------
 *  Layout primitives
 *  ------------------------------------------------------------------ */

export type GridColumns = NumericRangeWithSuffixes<
  Range<1, 12>,
  "none" | "subgrid"
>;
export type GridRows = NumericRangeWithSuffixes<
  Range<1, 6>,
  "none" | "subgrid"
>;

export type FlexDirection = "row" | "row-reverse" | "col" | "col-reverse";
export type FlexWrap = "nowrap" | "wrap" | "wrap-reverse";

export type AxisJustify =
  | "start"
  | "end"
  | "center"
  | "between"
  | "around"
  | "evenly";

export type AxisAlign = "start" | "end" | "center" | "stretch";
export type AxisAlignWithBaseline = AxisAlign | "baseline";

export type AxisAlignContent = AxisJustify;
export type AxisAlignContentWithStretch = AxisJustify | "stretch";

export type ContainerSize = SizeRange<"xs", "7xl"> | "full";
export type ContainerBreakpoint = "always" | SizeRange<"sm", "2xl">;

export type DimensionVariant =
  | "0"
  | "auto"
  | "fit"
  | "screen"
  | SizeRange<"3xs", "full">
  | NumericWithString<Range<1, 256>>;
export type MarginVariant = OUISpacing | "auto";

export type Breakpoint = "sm" | "md" | "lg" | "xl" | "2xl";

/**
 * Responsive<T> - either a single value T or an object keyed by breakpoints.
 * Example:
 *   T
 *   | { sm?: T; md?: T; lg?: T; ... }
 */
export type Responsive<T extends string | number> = T | Partial<Record<Breakpoint, T>>;
