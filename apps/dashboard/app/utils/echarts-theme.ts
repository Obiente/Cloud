/**
 * Utility functions for applying OUI theme to Apache ECharts
 * 
 * This module is client-only and should only be imported dynamically
 */

import { getOUIColors } from "./monaco-theme";

// Track if theme has been registered
let themeRegistered = false;

// Chroma.js instance (lazy loaded)
let chroma: any = null;

/**
 * Lazy loads chroma.js for color manipulation
 */
async function getChroma() {
  if (chroma) return chroma;
  
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const chromaModule = await import("chroma-js");
    chroma = (chromaModule as any).default || chromaModule;
    return chroma;
  } catch (err) {
    console.warn("Failed to load chroma-js:", err);
    return null;
  }
}

/**
 * Helper function to manipulate colors with chroma.js
 * Falls back to simple string operations if chroma is not available
 */
function manipulateColor(
  color: string,
  operations: {
    alpha?: number;
    brighten?: number;
    darken?: number;
    saturate?: number;
    desaturate?: number;
  },
  format: "hex" | "rgba" = "hex"
): string {
  if (!chroma) {
    // Fallback: simple opacity for alpha using hex+alpha format
    if (operations.alpha !== undefined && format === "hex") {
      const alphaHex = Math.round(operations.alpha * 255).toString(16).padStart(2, "0");
      return color + alphaHex;
    }
    // Fallback: use hex color if rgba requested but chroma not available
    if (format === "rgba") {
      // Try to parse hex to rgba manually
      const hex = color.replace("#", "");
      const r = parseInt(hex.substring(0, 2), 16);
      const g = parseInt(hex.substring(2, 4), 16);
      const b = parseInt(hex.substring(4, 6), 16);
      const a = operations.alpha !== undefined ? operations.alpha : 1;
      return `rgba(${r}, ${g}, ${b}, ${a})`;
    }
    return color;
  }

  try {
    let c = chroma(color);
    
    if (operations.brighten !== undefined) {
      c = c.brighten(operations.brighten);
    }
    if (operations.darken !== undefined) {
      c = c.darken(operations.darken);
    }
    if (operations.saturate !== undefined) {
      c = c.saturate(operations.saturate);
    }
    if (operations.desaturate !== undefined) {
      c = c.desaturate(operations.desaturate);
    }
    
    // Apply alpha last if specified
    if (operations.alpha !== undefined) {
      c = c.alpha(operations.alpha);
    }
    
    // Return in requested format
    if (format === "rgba") {
      const rgba = c.rgba();
      return `rgba(${Math.round(rgba[0])}, ${Math.round(rgba[1])}, ${Math.round(rgba[2])}, ${rgba[3]})`;
    }
    return c.hex();
  } catch (err) {
    console.warn("Failed to manipulate color with chroma:", err);
    // Fallback
    if (operations.alpha !== undefined && format === "hex") {
      const alphaHex = Math.round(operations.alpha * 255).toString(16).padStart(2, "0");
      return color + alphaHex;
    }
    if (format === "rgba") {
      const hex = color.replace("#", "");
      const r = parseInt(hex.substring(0, 2), 16);
      const g = parseInt(hex.substring(2, 4), 16);
      const b = parseInt(hex.substring(4, 6), 16);
      const a = operations.alpha !== undefined ? operations.alpha : 1;
      return `rgba(${r}, ${g}, ${b}, ${a})`;
    }
    return color;
  }
}

/**
 * Gets OUI colors formatted for ECharts
 */
export function getOUIEChartsColors() {
  const colors = getOUIColors();
  
  // Get additional colors from CSS variables if available
  let gridBorder = "#2a2347";
  if (typeof window !== "undefined" && typeof document !== "undefined") {
    try {
      const style = getComputedStyle(document.documentElement);
      gridBorder = style.getPropertyValue("--oui-border-muted").trim() || gridBorder;
    } catch (e) {
      // Fallback to default
    }
  }
  
  return {
    background: colors.editorBackground || colors.background || "#171521",
    textPrimary: colors.foreground || colors.editorForeground || "#f5f3ff",
    textSecondary: colors.lineNumberForeground || "#9e8cff",
    textTertiary: colors.lineNumberForeground || "#6b6396",
    border: colors.borderDefault || "#3a2f5c",
    gridBorder,
    
    // Accent colors
    primary: colors.accentPrimary || "#8b5cf6",
    secondary: colors.accentSecondary || "#22d3ee",
    success: colors.accentSuccess || "#22c55e",
    warning: colors.accentWarning || "#f59e0b",
    danger: colors.accentDanger || "#f43f5e",
    info: colors.accentInfo || "#60a5fa",
    
    // Tooltip colors
    tooltipBg: colors.editorWidgetBackground || colors.dropdownBackground || "#242030",
    tooltipBorder: colors.editorWidgetBorder || colors.borderDefault || "#3a2f5c",
    tooltipText: colors.editorWidgetForeground || colors.foreground || "#f5f3ff",
  };
}

/**
 * Creates an ECharts theme configuration using OUI colors
 */
export async function createOUIEChartsTheme() {
  // Ensure chroma is loaded
  await getChroma();
  
  const colors = getOUIEChartsColors();
  
  // Helper to create gradient stops with chroma (using rgba format for proper alpha)
  const createGradientStops = (baseColor: string) => {
    return [
      { offset: 0, color: manipulateColor(baseColor, { alpha: 0.25 }, "rgba") },
      { offset: 1, color: manipulateColor(baseColor, { alpha: 0.02 }, "rgba") },
    ];
  };
  
  return {
    color: [
      colors.primary,      // Primary (CPU)
      colors.success,      // Success (Memory)
      colors.secondary,    // Secondary (Network Rx)
      colors.warning,      // Warning (Network Tx)
      colors.danger,       // Danger (Disk Read)
      colors.info,         // Info (Disk Write)
    ],
    backgroundColor: "transparent",
    textStyle: {
      color: colors.textPrimary,
      fontFamily: "system-ui, -apple-system, sans-serif",
    },
    title: {
      textStyle: {
        color: colors.textPrimary,
        fontSize: 14,
        fontWeight: "600",
      },
      subtextStyle: {
        color: colors.textSecondary,
      },
    },
    line: {
      itemStyle: {
        borderWidth: 2,
      },
      lineStyle: {
        width: 2,
      },
      symbolSize: 4,
      symbol: "circle",
    },
    tooltip: {
      backgroundColor: colors.tooltipBg,
      borderColor: colors.tooltipBorder,
      borderWidth: 1,
      textStyle: {
        color: colors.tooltipText,
        fontSize: 12,
      },
      axisPointer: {
        lineStyle: {
          color: colors.textSecondary,
          width: 1,
          type: "dashed",
        },
        crossStyle: {
          color: colors.textSecondary,
          width: 1,
        },
      },
    },
    legend: {
      textStyle: {
        color: colors.textPrimary,
        fontSize: 12,
      },
      inactiveColor: colors.textTertiary,
    },
    dataZoom: {
      textStyle: {
        color: colors.textSecondary,
      },
      borderColor: colors.border,
      dataBackground: {
        areaStyleColor: manipulateColor(colors.primary, { alpha: 0.125 }, "rgba"),
        lineStyleColor: colors.primary,
      },
      fillerColor: manipulateColor(colors.primary, { alpha: 0.188 }, "rgba"),
      handleStyle: {
        color: colors.primary,
        borderColor: colors.primary,
      },
      moveHandleStyle: {
        color: colors.secondary,
      },
    },
    grid: {
      borderColor: colors.gridBorder,
      borderWidth: 1,
    },
    categoryAxis: {
      axisLine: {
        lineStyle: {
          color: colors.border,
          width: 1,
        },
      },
      axisTick: {
        lineStyle: {
          color: colors.border,
        },
      },
      axisLabel: {
        color: colors.textSecondary,
        fontSize: 11,
      },
      splitLine: {
        show: false,
      },
    },
    valueAxis: {
      axisLine: {
        lineStyle: {
          color: colors.border,
          width: 1,
        },
      },
      axisTick: {
        lineStyle: {
          color: colors.border,
        },
      },
      axisLabel: {
        color: colors.textSecondary,
        fontSize: 11,
      },
      splitLine: {
        lineStyle: {
          color: colors.gridBorder,
          type: "dashed",
        },
      },
    },
  };
}

/**
 * Registers OUI theme with ECharts
 */
export async function registerOUIEChartsTheme(echarts: any) {
  if (themeRegistered || !echarts) {
    return;
  }

  const theme = await createOUIEChartsTheme();
  
  try {
    echarts.registerTheme("oui", theme);
    themeRegistered = true;
  } catch (err) {
    console.warn("Failed to register OUI ECharts theme:", err);
  }
}

/**
 * Creates gradient color stops for area fills using chroma.js
 */
export function createAreaGradient(color: string): { type: string; x: number; y: number; x2: number; y2: number; colorStops: Array<{ offset: number; color: string }> } {
  return {
    type: "linear",
    x: 0,
    y: 0,
    x2: 0,
    y2: 1,
    colorStops: [
      { offset: 0, color: manipulateColor(color, { alpha: 0.25 }, "rgba") },
      { offset: 1, color: manipulateColor(color, { alpha: 0.02 }, "rgba") },
    ],
  };
}

