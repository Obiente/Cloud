/**
 * Utility composable for generating random skeleton variations
 * Makes skeletons look more natural and less repetitive
 */

/**
 * Generate a random width within a range for text skeletons
 * @param baseWidth Base width (e.g., "8rem")
 * @param variance Percentage variance (0-1, e.g., 0.3 = ±30%)
 * @returns Random width string
 */
export function randomTextWidth(baseWidth: string, variance: number = 0.3): string {
  // Extract numeric value and unit
  const match = baseWidth.match(/^([\d.]+)(\w+)$/);
  if (!match || !match[1] || !match[2]) return baseWidth;
  
  const numStr = match[1];
  const unit = match[2];
  const num = parseFloat(numStr);
  const min = num * (1 - variance);
  const max = num * (1 + variance);
  const random = min + Math.random() * (max - min);
  
  return `${random.toFixed(2)}${unit}`;
}

/**
 * Generate a random width for a text skeleton based on common text lengths
 * @param type Type of text (e.g., "title", "subtitle", "label", "value")
 * @returns Random width string
 */
export function randomTextWidthByType(type: "title" | "subtitle" | "label" | "value" | "short"): string {
  const ranges: Record<string, { min: number; max: number; unit: string }> = {
    title: { min: 10, max: 20, unit: "rem" },
    subtitle: { min: 6, max: 14, unit: "rem" },
    label: { min: 4, max: 8, unit: "rem" },
    value: { min: 3, max: 6, unit: "rem" },
    short: { min: 2, max: 4, unit: "rem" },
  };
  
  const range = ranges[type] || ranges.subtitle;
  if (!range) {
    return "8rem"; // fallback
  }
  const width = range.min + Math.random() * (range.max - range.min);
  return `${width.toFixed(2)}${range.unit}`;
}

/**
 * Get a random icon variation (different icons or opacity)
 * For use in skeletons where icons are dynamic
 */
export function randomIconVariation(): { opacity: number; scale: number } {
  return {
    opacity: 0.2 + Math.random() * 0.3, // 0.2 to 0.5
    scale: 0.9 + Math.random() * 0.2, // 0.9 to 1.1
  };
}

/**
 * Generate a unique variation ID for a skeleton instance
 * Useful for ensuring consistent variations per card
 */
export function useSkeletonVariations(seed?: number) {
  // Use seed for consistent variations if provided
  if (seed !== undefined) {
    // Simple seeded random (not cryptographically secure, just for visual variation)
    let currentSeed = seed;
    const random = () => {
      currentSeed = (currentSeed * 9301 + 49297) % 233280;
      return currentSeed / 233280;
    };
    
    return {
      titleWidth: `${(10 + random() * 10).toFixed(2)}rem`,
      subtitleWidth: `${(6 + random() * 8).toFixed(2)}rem`,
      labelWidth: `${(4 + random() * 4).toFixed(2)}rem`,
      valueWidth: `${(3 + random() * 3).toFixed(2)}rem`,
      shortWidth: `${(2 + random() * 2).toFixed(2)}rem`,
      iconOpacity: 0.2 + random() * 0.3,
      iconScale: 0.9 + random() * 0.2,
    };
  }
  
  const iconVar = randomIconVariation();
  return {
    titleWidth: randomTextWidthByType("title"),
    subtitleWidth: randomTextWidthByType("subtitle"),
    labelWidth: randomTextWidthByType("label"),
    valueWidth: randomTextWidthByType("value"),
    shortWidth: randomTextWidthByType("short"),
    iconOpacity: iconVar.opacity,
    iconScale: iconVar.scale,
  };
}

