/** ------------------------------------------------------------------
 *  Numeric helpers
 *  ------------------------------------------------------------------ */

// Allow both numeric literals and their string equivalents
export type NumericWithString<T extends number | string> = T | `${T & number}`;

/** ------------------------------------------------------------------
 *  Range helpers
 *  ------------------------------------------------------------------ */

// Build a tuple of length N
type BuildTuple<N extends number, Acc extends unknown[] = []> =
    Acc["length"] extends N ? Acc : BuildTuple<N, [...Acc, unknown]>;

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
    Suffixes extends string = never,
> = NumericWithString<T> | Suffixes;

/** ------------------------------------------------------------------
 *  Core design tokens
 *  ------------------------------------------------------------------ */

export type OUISpacing =
    | "none"
    | "xs"
    | "sm"
    | "md"
    | "lg"
    | "xl"
    | "2xl";

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

export type OUISize =
    | "xs"
    | "sm"
    | "md"
    | "lg"
    | "xl"
    | "2xl"
    | "3xl"
    | "4xl"
    | "5xl"
    | "6xl"
    | "7xl";

/** ------------------------------------------------------------------
 *  Size scale enum + derived ranges
 *  ------------------------------------------------------------------ */

export enum SizeScale {
    "none" = 0,
    "xs" = 1,
    "sm" = 2,
    "md" = 3,
    "lg" = 4,
    "xl" = 5,
    "2xl" = 6,
    "3xl" = 7,
    "4xl" = 8,
    "5xl" = 9,
    "6xl" = 10,
    "7xl" = 11,
    "full" = 12,
}

type EnumValue<T, K extends keyof T> = T[K];

type KeysMatchingValue<T, V> = {
    [K in keyof T]: T[K] extends V ? K : never;
}[keyof T];

// Map enum values back to key names within a range
export type SizeRange<
    Start extends keyof typeof SizeScale,
    End extends keyof typeof SizeScale,
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
export type FlexJustify =
    | "start"
    | "end"
    | "center"
    | "between"
    | "around"
    | "evenly";
export type FlexAlign = "start" | "end" | "center" | "baseline" | "stretch";
export type FlexAlignContent =
    | "start"
    | "end"
    | "center"
    | "between"
    | "around"
    | "evenly"
    | "stretch";

export type ContainerSize = SizeRange<"xs", "7xl"> | "full" | "default";
export type ContainerBreakpoint = "always" | SizeRange<"sm", "2xl">;

export type DimensionVariant = "auto" | "full" | "fit" | "screen";
export type MarginVariant = OUISpacing | "auto";
